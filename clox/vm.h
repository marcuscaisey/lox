#ifndef VM_H
#define VM_H

#include <stdbool.h>
#include <stdint.h>

#include "bytecode.h"
#include "value.h"

#define STACK_MAX 256

// A virtual machine which interprets Lox code.
// Must be initialised with `vm_init()` before use and freed with `vm_free()` after use.
struct vm {
    // Internal fields below, do not use.
    const struct bytecode_chunk *_chunk; // Chunk currently being executed
    uint8_t *_ip; // Instruction pointer pointing to the instruction to be executed next
    value _stack[STACK_MAX]; // Value stack for instructions to use
    value *_stack_top; // Points to where the next element will be pushed on the value stack
};

// Initialises `vm` for use.
void vm_init(struct vm *vm);

// Frees the memory associated with `vm`.
void vm_free(struct vm *vm);

// Interprets the given Lox `source` and reports whether interpretation was successful.
// This function can be called multiple times with different sources and the state will be
// maintained between calls.
bool vm_interpret(struct vm *vm, const char *source);

#endif
