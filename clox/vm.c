#include "vm.h"

#include <stdarg.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "bytecode.h"
#include "compiler.h"
#include "debug.h"
#include "errors.h"
#include "map.h"
#include "object.h"
#include "strings.h"
#include "value.h"
#include "vm_memory.h"

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

// Returns the element `distance` places from the top of the stack.
static value vm_stack_peek(vm *vm, int distance)
{
    return vm->_stack_top[-1 - distance];
}

// Empties the stack so that the next element added with `vm_stack_push()` will be the first one.
static void vm_stack_reset(vm *vm)
{
    vm->_stack_top = vm->_stack;
}

void vm_init(vm *vm)
{
    vm_stack_reset(vm);
    vm->_objects = NULL;
    map_init(&vm->_strings);
    map_init(&vm->_globals);
    map_init(&vm->_global_final_vars);
}

// Frees the objects managed by the VM and memory associated with them.
void vm_destroy_objects(vm *vm)
{
    struct object *object = vm->_objects;
    while (object != NULL) {
        struct object *next = object->next;
        object_destroy(vm, object);
        object = next;
    }
}

void vm_free(vm *vm)
{
    vm_destroy_objects(vm);
    map_free(vm, &vm->_strings);
    map_free(vm, &vm->_globals);
    map_free(vm, &vm->_global_final_vars);
}

// Reads the u8 at the instruction pointer and moves the instruction pointer past it.
static uint8_t vm_read_u8(vm *vm)
{
    return *vm->_ip++;
}

// Reads the u24 at the instruction pointer and moves the instruction pointer past it.
static uint32_t vm_read_u24(vm *vm)
{
    return (vm_read_u8(vm) << 16) + (vm_read_u8(vm) << 8) + (vm_read_u8(vm));
}

// Reads the constant index at the instruction pointer and moves the instruction pointer past it.
static constant_index vm_read_constant_index(vm *vm)
{
    uint8_t index = vm_read_u8(vm);
    if (index == 0xFF)
        return vm_read_u24(vm);
    return index;
}

#ifdef DEBUG_TRACE_EXECUTION
// Prints the contents of the value stack and the current instruction.
static void vm_print_trace_info(const vm *vm)
{
    // Lines up the start of the stack with the start of the instruction in the
    // disassemble_instruction output
    printf("          ");
    for (const value *p = vm->_stack; p < vm->_stack_top; p++) {
        printf("[ ");
        value_print(*p);
        printf(" ]");
    }
    printf("\n");
    int offset = vm->_ip - vm->_chunk->instructions;
    disassemble_instruction(vm->_chunk, offset);
}
#endif

// Prints an error relating to the bytecode offset `offset` with a formatted message.
static __attribute__((format(printf, 3, 4))) void vm_report_errorf(const vm *vm, int offset,
                                                                   const char *format, ...)
{
    char *msg;
    va_list args;
    va_start(args, format);
    int size = vasprintf(&msg, format, args);
    if (size < 0) {
        panicf("clox: compiler: encoding error formatting \"%s\"\n", format);
    }
    va_end(args);
    int line = vm->_chunk->offset_lines[offset];
    print_invalid_line_error(msg, vm->_source, line);
    free(msg);
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

// Executes the chunk of instructions stored in `_chunk`, starting from the instruction pointer
// `_ip`, and reports whether execution was successful.
static bool vm_execute(vm *vm)
{
    while (true) {
#ifdef DEBUG_TRACE_EXECUTION
        vm_print_trace_info(vm);
#endif

        int instruction_offset = vm->_ip - vm->_chunk->instructions;
        enum opcode instruction = vm_read_u8(vm);
        switch (instruction) {
        case OP_CONSTANT: {
            constant_index index = vm_read_constant_index(vm);
            value value = vm->_chunk->constants.values[index];
            vm_stack_push(vm, value);
            break;
        }
        case OP_NIL:
            vm_stack_push(vm, value_nil);
            break;
        case OP_TRUE:
            vm_stack_push(vm, value_bool(true));
            break;
        case OP_FALSE:
            vm_stack_push(vm, value_bool(false));
            break;
        case OP_POP:
            vm_stack_pop(vm);
            break;
        case OP_DEFINE_GLOBAL:
        case OP_DEFINE_GLOBAL_FINAL: {
            constant_index name_index = vm_read_constant_index(vm);
            value name = vm->_chunk->constants.values[name_index];
            // map_set could trigger an allocation causing garbage collection, so don't pop the
            // initialiser until we've set the variable
            value initialiser = vm_stack_peek(vm, 0);
            map_set(vm, &vm->_globals, name, initialiser);
            vm_stack_pop(vm);
            if (instruction == OP_DEFINE_GLOBAL_FINAL)
                map_set(vm, &vm->_global_final_vars, name, value_nil);
            break;
        }
        case OP_GET_GLOBAL: {
            constant_index name_index = vm_read_constant_index(vm);
            value name = vm->_chunk->constants.values[name_index];
            value value;
            if (!map_get(&vm->_globals, name, &value)) {
                vm_report_errorf(vm, instruction_offset, "'%s' has not been declared",
                                 value_as_string(name)->data);
                return false;
            }
            vm_stack_push(vm, value);
            break;
        }
        case OP_SET_GLOBAL: {
            constant_index name_index = vm_read_constant_index(vm);
            value name = vm->_chunk->constants.values[name_index];
            if (map_get(&vm->_global_final_vars, name, NULL)) {
                vm_report_errorf(vm, instruction_offset, "final variable '%s' can't be reassigned",
                                 value_as_string(name)->data);
                return false;
            }
            value value = vm_stack_peek(vm, 0);
            bool is_new_name = map_set(vm, &vm->_globals, name, value);
            if (is_new_name) {
                map_delete(&vm->_globals, name);
                vm_report_errorf(vm, instruction_offset, "'%s' has not been declared",
                                 value_as_string(name)->data);
                return false;
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
            constant_index name_index = vm_read_constant_index(vm);
            vm->_local_final_vars[top_stack_index] = name_index;
            break;
        }
        case OP_GET_LOCAL: {
            stack_index stack_index = vm_read_u8(vm);
            value value = vm->_stack[stack_index];
            vm_stack_push(vm, value);
            break;
        }
        case OP_SET_LOCAL: {
            stack_index stack_index = vm_read_u8(vm);
            int64_t name_index = vm->_local_final_vars[stack_index];
            if (name_index <= MAX_CONSTANT_INDEX) {
                value name = vm->_chunk->constants.values[name_index];
                vm_report_errorf(vm, instruction_offset, "final variable '%s' can't be reassigned",
                                 value_as_string(name)->data);
                return false;
            }
            value value = vm_stack_peek(vm, 0);
            vm->_stack[stack_index] = value;
            break;
        }
        case OP_EQUAL: {
            value a = vm_stack_pop(vm);
            value b = vm_stack_pop(vm);
            bool result = values_are_equal(a, b);
            vm_stack_push(vm, value_bool(result));
            break;
        }
        case OP_NOT_EQUAL: {
            value a = vm_stack_pop(vm);
            value b = vm_stack_pop(vm);
            bool result = !values_are_equal(a, b);
            vm_stack_push(vm, value_bool(result));
            break;
        }
#define EXECUTE_BINARY_NUMBER_OP(op, construct_result)                                     \
    do {                                                                                   \
        value b = vm_stack_peek(vm, 0);                                                    \
        value a = vm_stack_peek(vm, 1);                                                    \
        if (value_type(a) != VALUE_NUMBER || value_type(b) != VALUE_NUMBER) {              \
            vm_report_errorf(vm, instruction_offset,                                       \
                             "'%s' operator cannot be used with types '%s' and '%s'", #op, \
                             value_type_string(a), value_type_string(b));                  \
            return false;                                                                  \
        }                                                                                  \
        vm_stack_pop(vm);                                                                  \
        vm_stack_pop(vm);                                                                  \
        vm_stack_push(vm, construct_result(value_as_number(a) op value_as_number(b)));     \
    } while (false)
        case OP_LESS:
            EXECUTE_BINARY_NUMBER_OP(<, value_bool);
            break;
        case OP_LESS_EQUAL:
            EXECUTE_BINARY_NUMBER_OP(<=, value_bool);
            break;
        case OP_GREATER:
            EXECUTE_BINARY_NUMBER_OP(>, value_bool);
            break;
        case OP_GREATER_EQUAL:
            EXECUTE_BINARY_NUMBER_OP(>=, value_bool);
            break;
        case OP_ADD: {
            value b = vm_stack_peek(vm, 0);
            value a = vm_stack_peek(vm, 1);
            if (value_type(a) == VALUE_STRING || value_type(b) == VALUE_STRING) {
                vm_stack_pop(vm);
                vm_stack_pop(vm);
                struct object_string *result =
                    vm_concat_strings(vm, value_as_string(a), value_as_string(b));
                vm_stack_push(vm, value_string(result));
            } else {
                EXECUTE_BINARY_NUMBER_OP(+, value_number);
            }
            break;
        }
        case OP_SUBTRACT:
            EXECUTE_BINARY_NUMBER_OP(-, value_number);
            break;
        case OP_MULTIPLY:
            EXECUTE_BINARY_NUMBER_OP(*, value_number);
            break;
        case OP_DIVIDE: {
            value b = vm_stack_peek(vm, 0);
            if (value_as_number(b) == 0) {
                vm_report_errorf(vm, instruction_offset, "cannot divide by 0");
                return false;
            }
            EXECUTE_BINARY_NUMBER_OP(/, value_number);
            break;
        }
#undef EXECUTE_BINARY_NUMBER_OP
        case OP_NOT: {
            value value = vm_stack_pop(vm);
            bool result = value_is_falsey(value);
            vm_stack_push(vm, value_bool(result));
            break;
        }
        case OP_NEGATE: {
            value value = vm_stack_peek(vm, 0);
            if (value_type(value) != VALUE_NUMBER) {
                vm_report_errorf(vm, instruction_offset,
                                 "'-' operator cannot be used with type '%s'",
                                 value_type_string(value));
                return false;
            }
            vm_stack_pop(vm);
            vm_stack_push(vm, value_number(-value_as_number(value)));
            break;
        }
        case OP_PRINT: {
            value value = vm_stack_pop(vm);
            value_print(value);
            printf("\n");
            break;
        }
        case OP_RETURN: {
            return true;
        }
        }
    }
}

bool vm_interpret(vm *vm, const char *source)
{
    bool success = true;

    struct bytecode_chunk chunk;
    bytecode_chunk_init(&chunk);

    if (!compile(vm, source, &chunk)) {
        success = false;
        goto out_chunk_free;
    }

    // TODO: maybe these don't need to be struct members
    vm->_source = source;
    vm->_chunk = &chunk;
    vm->_ip = chunk.instructions;
    success = vm_execute(vm);

out_chunk_free:
    bytecode_chunk_free(&chunk);
    return success;
}
