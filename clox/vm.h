#ifndef VM_H
#define VM_H

#include <stdbool.h>
#include <stdint.h>

#include "bytecode.h"
#include "map.h"
#include "value.h"

#define STACK_MAX 256

// A virtual machine which interprets Lox code.
// Must be initialised with `vm_init()` before use and freed with `vm_free()` after use.
typedef struct vm {
    // Internal fields below, do not use.
    const char *_source; // Source being interpreted
    const struct bytecode_chunk *_chunk; // Chunk currently being executed
    uint8_t *_ip; // Instruction pointer pointing to the instruction to be executed next
    value _stack[STACK_MAX]; // Value stack for instructions to use
    value *_stack_top; // Points to where the next element will be pushed on the value stack
    struct object *_objects; // Head of the linked list of managed objects
    map _strings; // Interned strings
    map _globals; // Global variables
    map _global_final_vars; // Set of the global final variables
    // Each element stores whether the local variable (if any) at the same index in the value stack
    // is a final variable. If it is, then the element is the index of the constant containing the
    // variable name. Otherwise, the element is MAX_CONSTANT_INDEX+1.
    constant_index _local_final_vars[STACK_MAX];
} vm;

// Initialises `vm` for use.
void vm_init(vm *vm);

// Frees the memory associated with `vm`.
void vm_free(vm *vm);

// Interprets the given Lox `source` and reports whether interpretation was successful.
// This function can be called multiple times with different sources and the state will be
// maintained between calls.
bool vm_interpret(vm *vm, const char *source);

#endif
