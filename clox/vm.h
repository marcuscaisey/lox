#ifndef VM_H
#define VM_H

#include <stdint.h>
#include "bytecode.h"
#include "value.h"

#define STACK_MAX 256

// A virtual machine which interprets Lox code.
// Must be initialised with `vm_init` before usage and freed with `vm_free` after usage.
struct vm {
    // Internal fields below, do not use.
    const struct bytecode_chunk *_chunk; // Chunk currently being executed
    uint8_t *_ip; // Instruction pointer pointing to the instruction to be executed next
    value _stack[STACK_MAX];
    value *_stack_top; // Points to where the next element will be pushed
};

// Initialises `vm` for use.
void vm_init(struct vm *vm);

// Frees the memory associated with `vm`.
void vm_free(struct vm *vm);

// TODO: In the future, return a success status and populate an error struct similar to LoxError

// Values returned by `vm_interpret`.
enum interpret_result {
    INTERPRET_RESULT_OK,
    INTERPRET_RESULT_COMPILE_ERROR,
    INTERPRET_RESULT_RUNTIME_ERROR,
};

// Executes the given chunk of bytecode.
enum interpret_result vm_interpret(struct vm *vm, const struct bytecode_chunk *chunk);

#endif
