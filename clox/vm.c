#include "vm.h"

#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>

#include "bytecode.h"
#include "compiler.h"
#include "debug.h"
#include "value.h"

// Updates the stack to be empty
static void vm_stack_reset(struct vm *vm)
{
    vm->_stack_top = vm->_stack;
}

static void vm_stack_push(struct vm *vm, value value)
{
    *vm->_stack_top++ = value;
}

static value vm_stack_pop(struct vm *vm)
{
    return *(--vm->_stack_top);
}

void vm_init(struct vm *vm)
{
    vm_stack_reset(vm);
}

void vm_free(struct vm *vm)
{
    (void)vm;
}

// Reads the u8 pointed to by `ip` and increments `ip` past it.
static uint8_t vm_read_u8(struct vm *vm)
{
    return *vm->_ip++;
}

// Reads the u24 pointed to by `ip` and increments `ip` past it.
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
    if (vm->_stack_top - vm->_stack > 0) {
        printf("          ");
        for (const value *p = vm->_stack; p < vm->_stack_top; p++) {
            printf("[ ");
            value_print(*p);
            printf(" ]");
        }
        printf("\n");
    }
    int offset = vm->_ip - vm->_chunk->instructions;
    disassemble_instruction(*vm->_chunk, offset);
}
#endif

// Executes the chunk of instructions stored in `_chunk`, starting from the instruction pointer
// `_ip`, and reports whether execution was successful.
static bool vm_execute(struct vm *vm)
{
    while (true) {
#ifdef DEBUG
        vm_print_trace_info(vm);
#endif

        enum opcode opcode = vm_read_u8(vm);
        switch (opcode) {
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
        case OP_ADD: {
            value b = vm_stack_pop(vm);
            value a = vm_stack_pop(vm);
            vm_stack_push(vm, a + b);
            break;
        }
        case OP_SUBTRACT: {
            value b = vm_stack_pop(vm);
            value a = vm_stack_pop(vm);
            vm_stack_push(vm, a - b);
            break;
        }
        case OP_MULTIPLY: {
            value b = vm_stack_pop(vm);
            value a = vm_stack_pop(vm);
            vm_stack_push(vm, a * b);
            break;
        }
        case OP_DIVIDE: {
            value b = vm_stack_pop(vm);
            value a = vm_stack_pop(vm);
            vm_stack_push(vm, a / b);
            break;
        }
        case OP_NEGATE: {
            value value = vm_stack_pop(vm);
            vm_stack_push(vm, -value);
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

    if (!compile(source, &chunk)) {
        success = false;
        goto out_chunk_free;
    }

    // TODO: maybe these don't need to be struct members
    vm->_chunk = &chunk;
    vm->_ip = chunk.instructions;
    success = vm_execute(vm);

out_chunk_free:
    bytecode_chunk_free(&chunk);
    return success;
}
