#include "vm.h"

#include <stddef.h>
#include <stdint.h>
#include <stdio.h>

#include "bytecode.h"
#include "debug.h"
#include "value.h"

#define DEBUG_TRACE_EXECUTION
// #undef DEBUG_TRACE_EXECUTION

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
    disassemble_instruction(vm->_chunk, offset);
}

// Executes the chunk of instructions stored in `_chunk`, starting from the instruction pointer
// `_ip`.
static enum interpret_result vm_execute(struct vm *vm)
{
    for (;;) {
#ifdef DEBUG_TRACE_EXECUTION
        vm_print_trace_info(vm);
#endif
        enum opcode opcode = vm_read_u8(vm);
        switch (opcode) {
        case OPCODE_CONSTANT: {
            uint8_t index = vm_read_u8(vm);
            value value = vm->_chunk->constants.values[index];
            vm_stack_push(vm, value);
            break;
        }
        case OPCODE_CONSTANT_LONG: {
            uint8_t index = vm_read_u24(vm);
            value value = vm->_chunk->constants.values[index];
            vm_stack_push(vm, value);
            break;
        }
        case OPCODE_RETURN: {
            value value = vm_stack_pop(vm);
            value_print(value);
            printf("\n");
            return INTERPRET_RESULT_OK;
        }
        }
    }
}

enum interpret_result vm_interpret(struct vm *vm, const struct bytecode_chunk *chunk)
{
    // TODO: maybe these don't need to be struct members
    vm->_chunk = chunk;
    vm->_ip = chunk->instructions;

    return vm_execute(vm);
}
