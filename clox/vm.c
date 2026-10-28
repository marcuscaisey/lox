#include "vm.h"

#include <math.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "builtins.h"
#include "bytecode.h"
#include "compiler.h"
#include "debug.h"
#include "dynamic_array.h"
#include "errors.h"
#include "line_number.h"
#include "map.h"
#include "memory.h"
#include "object.h"
#include "strings.h"
#include "value.h"
#include "vm_memory.h"

line_number frame_last_instruction_line(const struct frame *frame)
{
    // ip-1 either points at the opcode of the previous instruction or one of its operands. Either
    // way, they should both be associated with the same line.
    size_t offset = frame->_ip - 1 - frame->_chunk->instructions;
    line_number line = frame->_chunk->offset_lines[offset];
    return line;
}

// Reads the u8 at the instruction pointer and moves the instruction pointer past it.
static uint8_t frame_read_u8(struct frame *frame)
{
    return *frame->_ip++;
}

// Reads the u16 at the instruction pointer and moves the instruction pointer past it.
static uint16_t frame_read_u16(struct frame *frame)
{
    return (frame_read_u8(frame) << 8) | (frame_read_u8(frame));
}

// Reads the u24 at the instruction pointer and moves the instruction pointer past it.
static uint32_t frame_read_u24(struct frame *frame)
{
    return (frame_read_u8(frame) << 16) | (frame_read_u8(frame) << 8) | (frame_read_u8(frame));
}

// Reads the opcode at the instruction pointer and moves the instruction pointer past it.
static enum opcode frame_read_opcode(struct frame *frame)
{
    return *frame->_ip++;
}

// Reads the constant index at the instruction pointer and moves the instruction pointer past it.
static constant_index frame_read_constant_index(struct frame *frame)
{
    uint8_t index = frame_read_u8(frame);
    if (index == 0xFF)
        return frame_read_u24(frame);
    return index;
}

// Reads the pop count at the instruction pointer and moves the instruction pointer past it.
static pop_count frame_read_pop_count(struct frame *frame)
{
    uint8_t count = frame_read_u8(frame);
    if (count == 0xFF)
        return frame_read_u24(frame);
    return count;
}

// Reads the stack index at the instruction pointer and moves the instruction pointer past it.
static stack_index frame_read_stack_index(struct frame *frame)
{
    uint8_t index = frame_read_u8(frame);
    if (index == 0xFF)
        return frame_read_u24(frame);
    return index;
}

// Reads the jump offset at the instruction pointer and moves the instruction pointer past it.
static jump_offset frame_read_jump_offset(struct frame *frame)
{
    return frame_read_u16(frame);
}

// Reads the argument count at the instruction pointer and moves the instruction pointer past it.
static argument_count frame_read_argument_count(struct frame *frame)
{
    uint8_t count = frame_read_u8(frame);
    if (count == 0xFF)
        return frame_read_u24(frame);
    return count;
}

void vm_init(vm *vm)
{
    vm->_frames = NULL;
    vm->_frames_len = 0;
    vm->_frames_cap = 0;
    vm->_stack = NULL;
    vm->_stack_cap = 0;
    vm->_stack_top = NULL;
    vm->_objects = NULL;
    map_init(&vm->_strings);
    map_init(&vm->_globals);
    map_init(&vm->_global_final_vars);
    vm->_local_final_vars = NULL;

    for (size_t i = 0; i < builtin_functions_len; i++) {
        struct builtin_function function = builtin_functions[i];
        struct object_native_function *function_object = object_native_function_new(
            vm, function.name, strlen(function.name), function.arity, function.body);
        map_set(vm, &vm->_globals, string_value(function_object->name),
                native_function_value(function_object));
    }
}

// Frees the objects managed by the VM and memory associated with them.
void vm_free_objects(vm *vm)
{
    struct object *object = vm->_objects;
    while (object != NULL) {
        struct object *next = object->next;
        object_free(vm, object);
        object = next;
    }
}

void vm_deinit(vm *vm)
{
    free(vm->_stack);
    vm_free_objects(vm);
    map_deinit(vm, &vm->_strings);
    map_deinit(vm, &vm->_globals);
    map_deinit(vm, &vm->_global_final_vars);
    free(vm->_local_final_vars);
}

// Ensures that there's enough space in the value stack so that it can hold `n` elements by
// reallocating more space if necessary. If the stack already has enough space, then this is a
// no-op. If the stack was grown, then the local final variables array will be as well.
static void vm_stack_grow(vm *vm, size_t n)
{
    if (vm->_stack_cap < n) {
        ptrdiff_t current_stack_depth = vm->_stack_top - vm->_stack;
        size_t stack_size = DYNAMIC_ARRAY_SIZE(vm->_stack, n);
        size_t local_final_var_size = DYNAMIC_ARRAY_SIZE(vm->_local_final_vars, n);
        vm->_stack = xrealloc(vm->_stack, stack_size);
        vm->_local_final_vars = xrealloc(vm->_local_final_vars, local_final_var_size);
        vm->_stack_cap = n;
        vm->_stack_top = vm->_stack + current_stack_depth;
    }
}

// Adds an element to the top of the stack.
static void vm_stack_push(vm *vm, value value)
{
    *vm->_stack_top++ = value;
}

// Removes the element from the top of the stack and returns it.
static value vm_stack_pop(vm *vm)
{
    return *(--vm->_stack_top);
}

// Removes `count` elements from the top of the stack.
static void vm_stack_pop_n(vm *vm, pop_count count)
{
    vm->_stack_top -= count;
}

// Returns the element `distance` places from the top of the stack.
static value vm_stack_peek(vm *vm, int distance)
{
    return vm->_stack_top[-1 - distance];
}

#ifdef DEBUG_TRACE_EXECUTION
// Prints the contents of the value stack and the frame's current instruction.
static void vm_print_trace_info(const vm *vm, const struct frame *frame)
{
    // Lines up the start of the stack with the start of the instruction in the
    // `disassemble_instruction()` output
    printf("          ");
    for (const value *p = vm->_stack; p < vm->_stack_top; p++) {
        char *string = value_string(*p);
        printf("[ %s ]", string);
        free(string);
    }
    printf("\n");
    int offset = frame->_ip - frame->_chunk->instructions;
    disassemble_instruction(frame->_chunk, offset);
}
#endif

// Prints an error related to the most recently read instruction in the current frame with a
// formatted message. `source` is the source that the instruction was compiled from.
static __attribute__((format(printf, 3, 4))) void vm_report_errorf(const vm *vm, const char *source,
                                                                   const char *format, ...)
{
    char *msg;
    va_list args;
    va_start(args, format);
    xvasprintf(&msg, format, args);
    va_end(args);
    struct frame *current_frame = &vm->_frames[vm->_frames_len - 1];
    line_number line = frame_last_instruction_line(current_frame);
    print_invalid_line_error(msg, source, line);
    free(msg);

    if (vm->_frames_len < 2)
        return;

    fprintf(stderr, "\n");
    print_stack_trace(vm->_frames, vm->_frames_len, source);
}

// Returns a new string created by concatenating `a` and `b`.
struct object_string *vm_concat_strings(vm *vm, struct object_string *a, struct object_string *b)
{
    size_t len = a->len + b->len;
    char *data = vm_malloc(vm, len + 1);
    memcpy(data, a->data, a->len);
    memcpy(data + a->len, b->data, b->len);
    data[len] = '\0';
    return object_string_take(vm, data, len);
}

// Returns a new string created by repeating `s` `n` times.
struct object_string *vm_repeat_string(vm *vm, struct object_string *s, int64_t n)
{
    size_t len = s->len * n;
    char *data = vm_malloc(vm, len + 1);
    for (int i = 0; i < n; i++)
        memcpy(data + s->len * i, s->data, s->len);
    data[len] = '\0';
    return object_string_take(vm, data, len);
}

// Number of active frames that space will initially be allocated for
#define INITIAL_FRAMES_CAPACITY 64

// Number of values in that space will initially be allocated for in the value stack
// This will allow for a stack of `INITIAL_FRAME_CAPACITY` calls using up to 256 simultaneous values
#define INITIAL_STACK_CAPACITY (INITIAL_FRAMES_CAPACITY * 256)

// Executes `chunk` which was compiled from `source` and reports whether execution was successful.
static bool vm_execute(vm *vm, const struct bytecode_chunk *chunk, const char *source)
{
    DYNAMIC_ARRAY_GROW(vm->_frames, vm->_frames_cap, INITIAL_FRAMES_CAPACITY);
    vm_stack_grow(vm, INITIAL_STACK_CAPACITY);

    struct frame *frame = &vm->_frames[vm->_frames_len++];
    frame->name = NULL;
    frame->_chunk = chunk;
    frame->_ip = chunk->instructions;
    frame->_stack_offset = 0;

    while (true) {
#ifdef DEBUG_TRACE_EXECUTION
        vm_print_trace_info(vm, frame);
#endif

        enum opcode opcode = frame_read_opcode(frame);
        switch (opcode) {
        case OP_CONSTANT: {
            constant_index index = frame_read_constant_index(frame);
            value value = frame->_chunk->constants.values[index];
            vm_stack_push(vm, value);
            break;
        }

        case OP_NIL:
            vm_stack_push(vm, nil_value);
            break;

        case OP_TRUE:
            vm_stack_push(vm, bool_value(true));
            break;

        case OP_FALSE:
            vm_stack_push(vm, bool_value(false));
            break;

        case OP_POP:
            vm_stack_pop(vm);
            break;

        case OP_POP_N: {
            pop_count count = frame_read_pop_count(frame);
            vm_stack_pop_n(vm, count);
            break;
        }

        case OP_DUP: {
            value value = vm_stack_peek(vm, 0);
            vm_stack_push(vm, value);
            break;
        }

        case OP_DEFINE_GLOBAL:
        case OP_DEFINE_GLOBAL_FINAL: {
            constant_index name_index = frame_read_constant_index(frame);
            value name = frame->_chunk->constants.values[name_index];
            // map_set could trigger an allocation causing garbage collection, so don't pop the
            // initialiser until we've set the variable
            value initialiser = vm_stack_peek(vm, 0);
            map_set(vm, &vm->_globals, name, initialiser);
            vm_stack_pop(vm);
            if (opcode == OP_DEFINE_GLOBAL_FINAL)
                map_set(vm, &vm->_global_final_vars, name, nil_value);
            break;
        }

        case OP_GET_GLOBAL: {
            constant_index name_index = frame_read_constant_index(frame);
            value name = frame->_chunk->constants.values[name_index];
            value value;
            if (!map_get(&vm->_globals, name, &value)) {
                vm_report_errorf(vm, source, "'%s' has not been declared",
                                 value_as_string(name)->data);
                goto out_error;
            }
            vm_stack_push(vm, value);
            break;
        }

        case OP_SET_GLOBAL: {
            constant_index name_index = frame_read_constant_index(frame);
            value name = frame->_chunk->constants.values[name_index];
            if (map_get(&vm->_global_final_vars, name, NULL)) {
                vm_report_errorf(vm, source, "final variable '%s' can't be reassigned",
                                 value_as_string(name)->data);
                goto out_error;
            }
            value value = vm_stack_peek(vm, 0);
            bool is_new_name = map_set(vm, &vm->_globals, name, value);
            if (is_new_name) {
                map_delete(&vm->_globals, name);
                vm_report_errorf(vm, source, "'%s' has not been declared",
                                 value_as_string(name)->data);
                goto out_error;
            }
            break;
        }

        case OP_DEFINE_LOCAL: {
            stack_index top_stack_index = vm->_stack_top - vm->_stack - 1;
            vm->_local_final_vars[top_stack_index] = MAX_CONSTANT_INDEX + 1;
            break;
        }

        case OP_DEFINE_LOCAL_FINAL: {
            stack_index top_stack_index = vm->_stack_top - vm->_stack - 1;
            constant_index name_index = frame_read_constant_index(frame);
            vm->_local_final_vars[top_stack_index] = name_index;
            break;
        }

        case OP_GET_LOCAL: {
            stack_index stack_index = frame_read_stack_index(frame);
            value value = vm->_stack[frame->_stack_offset + stack_index];
            vm_stack_push(vm, value);
            break;
        }

        case OP_SET_LOCAL: {
            stack_index stack_index = frame_read_stack_index(frame);
            constant_index name_index = vm->_local_final_vars[frame->_stack_offset + stack_index];
            if (name_index <= MAX_CONSTANT_INDEX) {
                value name = frame->_chunk->constants.values[name_index];
                vm_report_errorf(vm, source, "final variable '%s' can't be reassigned",
                                 value_as_string(name)->data);
                goto out_error;
            }
            value value = vm_stack_peek(vm, 0);
            vm->_stack[frame->_stack_offset + stack_index] = value;
            break;
        }

        case OP_EQUAL: {
            value a = vm_stack_pop(vm);
            value b = vm_stack_pop(vm);
            bool result = values_are_equal(a, b);
            vm_stack_push(vm, bool_value(result));
            break;
        }

        case OP_NOT_EQUAL: {
            value a = vm_stack_pop(vm);
            value b = vm_stack_pop(vm);
            bool result = !values_are_equal(a, b);
            vm_stack_push(vm, bool_value(result));
            break;
        }

#define EXECUTE_BINARY_NUMBER_OP(op, construct_result)                                            \
    do {                                                                                          \
        value b = vm_stack_peek(vm, 0);                                                           \
        value a = vm_stack_peek(vm, 1);                                                           \
        if (value_type(a) != VALUE_NUMBER || value_type(b) != VALUE_NUMBER) {                     \
            vm_report_errorf(vm, source, "'%s' operator cannot be used with types '%s' and '%s'", \
                             #op, value_type_string(value_type(a)),                               \
                             value_type_string(value_type(b)));                                   \
            goto out_error;                                                                       \
        }                                                                                         \
        vm_stack_pop(vm);                                                                         \
        vm_stack_pop(vm);                                                                         \
        vm_stack_push(vm, construct_result(value_as_number(a) op value_as_number(b)));            \
    } while (false)

#define EXECUTE_COMPARISON_BINARY_OP(op)                                                   \
    do {                                                                                   \
        value b = vm_stack_peek(vm, 0);                                                    \
        value a = vm_stack_peek(vm, 1);                                                    \
        if (value_type(a) == VALUE_STRING || value_type(b) == VALUE_STRING) {              \
            vm_stack_pop(vm);                                                              \
            vm_stack_pop(vm);                                                              \
            bool result = strcmp(value_as_string(a)->data, value_as_string(b)->data) op 0; \
            vm_stack_push(vm, bool_value(result));                                         \
        } else {                                                                           \
            EXECUTE_BINARY_NUMBER_OP(op, bool_value);                                      \
        }                                                                                  \
    } while (false);

        case OP_LESS:
            EXECUTE_COMPARISON_BINARY_OP(<);
            break;

        case OP_LESS_EQUAL:
            EXECUTE_COMPARISON_BINARY_OP(<=);
            break;

        case OP_GREATER:
            EXECUTE_COMPARISON_BINARY_OP(>);
            break;

        case OP_GREATER_EQUAL:
            EXECUTE_COMPARISON_BINARY_OP(>=);
            break;
#undef EXECUTE_COMPARISON_BINARY_OP

        case OP_ADD: {
            value b = vm_stack_peek(vm, 0);
            value a = vm_stack_peek(vm, 1);
            if (value_type(a) == VALUE_STRING || value_type(b) == VALUE_STRING) {
                vm_stack_pop(vm);
                vm_stack_pop(vm);
                struct object_string *result =
                    vm_concat_strings(vm, value_as_string(a), value_as_string(b));
                vm_stack_push(vm, string_value(result));
            } else {
                EXECUTE_BINARY_NUMBER_OP(+, number_value);
            }
            break;
        }

        case OP_SUBTRACT:
            EXECUTE_BINARY_NUMBER_OP(-, number_value);
            break;

        case OP_MULTIPLY: {
            value b = vm_stack_peek(vm, 0);
            value a = vm_stack_peek(vm, 1);
            enum value_type a_type = value_type(a);
            enum value_type b_type = value_type(b);
            if ((a_type == VALUE_NUMBER && b_type == VALUE_STRING) ||
                (a_type == VALUE_STRING && b_type == VALUE_NUMBER)) {
                vm_stack_pop(vm);
                vm_stack_pop(vm);
                value n_operand = a;
                value string_operand = b;
                if (a_type == VALUE_STRING) {
                    n_operand = b;
                    string_operand = a;
                }
                double n_double = value_as_number(n_operand);
                int64_t n = trunc(n_double);
                if (n != n_double) {
                    vm_report_errorf(vm, source, "cannot multiply '%s' by non-integer '%s'",
                                     value_type_string(value_type(string_operand)),
                                     value_type_string(value_type(n_operand)));
                    goto out_error;
                }
                if (n < 0) {
                    vm_report_errorf(vm, source, "cannot multiply '%s' by negative '%s'",
                                     value_type_string(value_type(string_operand)),
                                     value_type_string(value_type(n_operand)));
                    goto out_error;
                }
                struct object_string *result =
                    vm_repeat_string(vm, value_as_string(string_operand), n);
                vm_stack_push(vm, string_value(result));
            } else {
                EXECUTE_BINARY_NUMBER_OP(*, number_value);
            }
            break;
        }

        case OP_DIVIDE: {
            value b = vm_stack_peek(vm, 0);
            if (value_as_number(b) == 0) {
                vm_report_errorf(vm, source, "cannot divide by 0");
                goto out_error;
            }
            EXECUTE_BINARY_NUMBER_OP(/, number_value);
            break;
        }
#undef EXECUTE_BINARY_NUMBER_OP

        case OP_MODULO: {
            value b = vm_stack_peek(vm, 0);
            value a = vm_stack_peek(vm, 1);
            if (value_as_number(b) == 0) {
                vm_report_errorf(vm, source, "cannot modulo by 0");
                goto out_error;
            }
            if (value_type(a) != VALUE_NUMBER || value_type(b) != VALUE_NUMBER) {
                vm_report_errorf(vm, source,
                                 "'%%' operator cannot be used with types '%s' and '%s'",
                                 value_type_string(value_type(a)),
                                 value_type_string(value_type(b)));
                goto out_error;
            }
            vm_stack_pop(vm);
            vm_stack_pop(vm);
            vm_stack_push(vm, number_value(fmod(value_as_number(a), value_as_number(b))));
            break;
        }

        case OP_NOT: {
            value value = vm_stack_pop(vm);
            bool result = !value_is_truthy(value);
            vm_stack_push(vm, bool_value(result));
            break;
        }

        case OP_NEGATE: {
            value value = vm_stack_peek(vm, 0);
            if (value_type(value) != VALUE_NUMBER) {
                vm_report_errorf(vm, source, "'-' operator cannot be used with type '%s'",
                                 value_type_string(value_type(value)));
                goto out_error;
            }
            vm_stack_pop(vm);
            vm_stack_push(vm, number_value(-value_as_number(value)));
            break;
        }

        case OP_PRINT: {
            value value = vm_stack_pop(vm);
            char *string = value_string(value);
            printf("%s\n", string);
            free(string);
            break;
        }

        case OP_JUMP: {
            jump_offset offset = frame_read_jump_offset(frame);
            frame->_ip += offset;
            break;
        }

        case OP_JUMP_BACK: {
            jump_offset offset = frame_read_jump_offset(frame);
            frame->_ip -= offset;
            break;
        }

        case OP_JUMP_IF_TRUE: {
            jump_offset offset = frame_read_jump_offset(frame);
            value top_value = vm_stack_peek(vm, 0);
            if (value_is_truthy(top_value))
                frame->_ip += offset;
            break;
        }

        case OP_JUMP_IF_FALSE: {
            jump_offset offset = frame_read_jump_offset(frame);
            value top_value = vm_stack_peek(vm, 0);
            if (!value_is_truthy(top_value))
                frame->_ip += offset;
            break;
        }

        case OP_CALL: {
            argument_count arg_count = frame_read_argument_count(frame);
            value callable = vm_stack_peek(vm, arg_count);

            struct object_native_function *native_function = NULL;
            struct object_function *function = NULL;
            argument_count arity;
            switch (value_type(callable)) {
            case VALUE_NATIVE_FUNCTION:
                native_function = value_as_native_function(callable);
                arity = native_function->arity;
                break;
            case VALUE_FUNCTION:
                function = value_as_function(callable);
                arity = function->arity;
                break;
            default:
                vm_report_errorf(vm, source, "'%s' value is not callable",
                                 value_type_string(value_type(callable)));
                goto out_error;
            }

            if (arg_count != arity) {
                const char *were_was = (arg_count) == 1 ? "was" : "were";
                const char *argument_suffix = arity == 1 ? "" : "s";
                struct object_string *name = function != NULL ? function->name :
                                                                native_function->name;
                vm_report_errorf(vm, source, "%s() accepts %d argument%s but %d %s given",
                                 name->data, arity, argument_suffix, arg_count, were_was);
                goto out_error;
            }

            if (native_function != NULL) {
                value *args = vm->_stack_top - arg_count;
                vm_stack_pop_n(vm, arg_count + 1); // Arguments + callable
                char *error_msg;
                value result = native_function->body(vm, args, &error_msg);
                if (values_are_equal(result, none_value)) {
                    vm_report_errorf(vm, source, "%s", error_msg);
                    free(error_msg);
                    goto out_error;
                }
                vm_stack_push(vm, result);
                break;
            }

            DYNAMIC_ARRAY_GROW(vm->_frames, vm->_frames_cap, vm->_frames_len + 1);
            size_t required_stack_depth =
                vm->_stack_top - vm->_stack + function->body.max_stack_depth;
            vm_stack_grow(vm, required_stack_depth);

            frame = &vm->_frames[vm->_frames_len++];
            frame->name = function->name;
            frame->_chunk = &function->body;
            frame->_ip = function->body.instructions;
            frame->_stack_offset = vm->_stack_top - vm->_stack - arg_count;
            break;
        }
#undef REPORT_ARITY_ERROR

        case OP_RETURN: {
            value return_value = vm_stack_pop(vm);
            vm->_frames_len--;
            if (vm->_frames_len == 0)
                return true;
            // OP_CALL left the callable and arguments on the stack and pointed stack_offset in the
            // new frame to the start of the arguments. So adding stack_offset takes us to the start
            // of the arguments and we need to take away 1 so that stack top points at the callable.
            vm->_stack_top = vm->_stack + frame->_stack_offset - 1;
            frame = &vm->_frames[vm->_frames_len - 1];
            vm_stack_push(vm, return_value);
            break;
        }
        }
    }

out_error:
    vm->_stack_top = vm->_stack;
    vm->_frames_len = 0;
    return false;
}

bool vm_interpret(vm *vm, const char *source)
{
    bool success = true;

    struct bytecode_chunk chunk;
    bytecode_chunk_init(&chunk);

    if (!compile(vm, source, &chunk)) {
        success = false;
        goto out_chunk_deinit;
    }

    success = vm_execute(vm, &chunk, source);

out_chunk_deinit:
    bytecode_chunk_deinit(&chunk);
    return success;
}
