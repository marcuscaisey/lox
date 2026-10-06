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
#include "object.h"
#include "strings.h"
#include "value.h"
#include "vm_internal.h"

// Adds an element to the top of the stack.
static void vm_stack_push(struct vm *vm, value value)
{
    *vm->_stack_top++ = value;
}

// Removes the element from the top of the stack and returns it.
static value vm_stack_pop(struct vm *vm)
{
    return *(--vm->_stack_top);
}

// Returns the element `distance` places from the top of the stack.
static value vm_stack_peek(struct vm *vm, int distance)
{
    return vm->_stack_top[-1 - distance];
}

// Empties the stack so that the next element added with `vm_stack_push()` will be the first one.
static void vm_stack_reset(struct vm *vm)
{
    vm->_stack_top = vm->_stack;
}

void vm_init(struct vm *vm)
{
    vm_stack_reset(vm);
    vm->_objects = NULL;
}

// Frees the objects managed by the VM and memory associated with them.
void vm_destroy_objects(struct vm *vm)
{
    struct object *object = vm->_objects;
    while (object != NULL) {
        struct object *next = object->next;
        object_destroy(vm, object);
        object = next;
    }
}

void vm_free(struct vm *vm)
{
    vm_destroy_objects(vm);
}

// Reads the u8 at the instruction pointer and moves the instruction pointer past it.
static uint8_t vm_read_u8(struct vm *vm)
{
    return *vm->_ip++;
}

// Reads the u24 at the instruction pointer and moves the instruction pointer past it.
static uint8_t vm_read_u24(struct vm *vm)
{
    return (vm_read_u8(vm) << 16) + (vm_read_u8(vm) << 8) + (vm_read_u8(vm));
}

#ifdef DEBUG
// Prints the contents of the value stack and the current instruction.
static void vm_print_trace_info(const struct vm *vm)
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
    disassemble_instruction(*vm->_chunk, offset);
}
#endif

// Prints an error relating to the bytecode offset `offset` with a formatted message.
static __attribute__((format(printf, 3, 4))) void vm_report_errorf(struct vm vm, int offset,
                                                                   const char *format, ...)
{
    char *msg;
    va_list args;
    va_start(args, format);
    int size = vasprintf(&msg, format, args);
    if (size < 0) {
        fprintf(stderr, "compiler: encoding error formatting \"%s\"\n", format);
        abort();
    }
    va_end(args);
    int line = vm._chunk->offset_lines[offset];
    print_invalid_line_error(msg, vm._source, line);
    free(msg);
}

// Returns a new string created by concatenating `a` and `b`.
struct object_string *vm_concat_strings(struct vm *vm, struct object_string *a,
                                        struct object_string *b)
{
    size_t len = a->len + b->len;
    char *data = vm_malloc(vm, len);
    memcpy(data, a->data, a->len);
    memcpy(data + a->len, b->data, b->len);
    return object_string_take(vm, data, len);
}

// Executes the chunk of instructions stored in `_chunk`, starting from the instruction pointer
// `_ip`, and reports whether execution was successful.
static bool vm_execute(struct vm *vm)
{
    while (true) {
#ifdef DEBUG
        vm_print_trace_info(vm);
#endif

        int instruction_offset = vm->_ip - vm->_chunk->instructions;
        enum opcode instruction = vm_read_u8(vm);
        switch (instruction) {
        case OP_CONSTANT: {
            uint8_t index = vm_read_u8(vm);
            value value = vm->_chunk->constants.values[index];
            vm_stack_push(vm, value);
            break;
        }
        case OP_CONSTANT_LONG: {
            uint8_t index = vm_read_u24(vm);
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
#define EXECUTE_BINARY_NUMBER_OP(op, construct_result)                                            \
    do {                                                                                          \
        value b = vm_stack_peek(vm, 0);                                                           \
        value a = vm_stack_peek(vm, 1);                                                           \
        if (value_type(a) != VALUE_NUMBER || value_type(b) != VALUE_NUMBER) {                     \
            vm_report_errorf(*vm, instruction_offset,                                             \
                             "'%s' operator cannot be used with types '%s' and '%s'", #op,        \
                             value_type_string(value_type(a)), value_type_string(value_type(b))); \
            return false;                                                                         \
        }                                                                                         \
        vm_stack_pop(vm);                                                                         \
        vm_stack_pop(vm);                                                                         \
        vm_stack_push(vm, construct_result(value_as_number(a) op value_as_number(b)));            \
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
        case OP_DIVIDE:
            EXECUTE_BINARY_NUMBER_OP(/, value_number);
            break;
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
                vm_report_errorf(*vm, instruction_offset,
                                 "'-' operator cannot be used with type '%s'",
                                 value_type_string(value_type(value)));
                return false;
            }
            vm_stack_pop(vm);
            vm_stack_push(vm, value_number(-value_as_number(value)));
            break;
        }
        case OP_RETURN: {
            value value = vm_stack_pop(vm);
            value_print(value);
            printf("\n");
            return true;
        }
        }
    }
}

bool vm_interpret(struct vm *vm, const char *source)
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
