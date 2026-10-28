#ifndef VM_H
#define VM_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#include "bytecode.h"
#include "line_number.h"
#include "map.h"
#include "object.h"
#include "value.h"

// Runtime representation of a frame as defined in bytecode.h.
struct frame {
    // Name of the callable if this frame was created to execute a callable, otherwise `NULL`
    struct object_string *name;
    // Internal fields below, do not use.
    const struct bytecode_chunk *_chunk; // Bytecode chunk being executed
    uint8_t *_ip; // Pointer to the next instruction to be executed (instruction pointer)
    size_t _stack_offset; // Index in the VM's value stack where this frame's value stack begins
};

// Returns the line associated with the last instruction read in this frame.
line_number frame_last_instruction_line(const struct frame *frame);

// A virtual machine which interprets Lox code.
// Must be initialised with `vm_init()` before use and deinitialised with `vm_deinit()` after use.
typedef struct vm {
    // Internal fields below, do not use.
    struct frame *_frames; // Stack of frames for active bytecode executions
    size_t _frames_len; // Number of active frames
    size_t _frames_cap; // Number of actives frames space has been allocated for
    // Value stack shared by all active frames.
    // Each frame uses a contiguous region of the stack, starting at the index recorded in its
    // `stack_offset` field.
    value *_stack;
    size_t _stack_cap; // Number of elements space has been allocated for in the value stack
    value *_stack_top; // Points to where the next element will be pushed on the value stack
    struct object *_objects; // Head of the linked list of managed objects
    map _strings; // Interned strings
    map _globals; // Global variables
    map _global_final_vars; // Set of the global final variables
    // Array where each element stores whether the local variable (if any) at the same index in the
    // value stack is a final variable. If it is, then the element is the index of the constant
    // containing the variable name. Otherwise, the element is MAX_CONSTANT_INDEX + 1.
    // The size of this array mirrors the value stack.
    constant_index *_local_final_vars;
} vm;

// Ensures that when MAX_CONSTANT_INDEX + 1 is a valid value in `vm._local_final_vars`.
_Static_assert((constant_index)-1 > 0, "constant_index must be unsigned");
_Static_assert(MAX_CONSTANT_INDEX < (constant_index)-1,
               "MAX_CONSTANT_INDEX must be < max(constant_index)");

// Initialises `vm` for use.
void vm_init(vm *vm);

// Frees the memory associated with `vm`.
void vm_deinit(vm *vm);

// Interprets the given Lox `source` and reports whether interpretation was successful.
// This function can be called multiple times with different sources and the state will be
// maintained between calls.
bool vm_interpret(vm *vm, const char *source);

#endif
