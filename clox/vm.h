#ifndef VM_H
#define VM_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#include "line_number.h"
#include "map.h"
#include "value.h"
#include "vm_fwd.h" // IWYU pragma: export

// Runtime representation of a frame as defined in bytecode.h.
struct frame {
    // Function called if this frame was created for a function call, otherwise `NULL`
    struct object_function *function;
    // Internal fields below, do not use.
    const struct bytecode_chunk *_chunk; // Bytecode chunk being executed
    uint8_t *_ip; // Pointer to the next byte to be read from the instructions (instruction pointer)
    size_t _stack_offset; // Index in the VM's value stack where this frame's value stack begins
    // Array of upvalues associated with the closure called if this frame was created for a closure
    // call, otherwise `NULL`
    struct object_upvalue **_upvalues;
};

// Returns the line associated with the last instruction read in this frame.
line_number frame_last_instruction_line(const struct frame *frame);

// A virtual machine which interprets Lox code.
// Must be initialised with `vm_init()` before use and deinitialised with `vm_deinit()` after use.
struct vm {
    // Internal fields below, do not use.
    struct object *_objects; // Head of the linked list of objects
    size_t _allocated_bytes; // Total number of bytes of VM managed memory allocated
    // Total number of bytes of VM managed memory that have to be allocated before the next GC cycle
    // is triggered
    size_t _next_gc_bytes;
    bool _is_gc_enabled;
    // Stack of reachable objects which haven't had their references traced
    struct object **_grey_objects;
    size_t _grey_objects_len; // Number of grey objects
    size_t _grey_objects_cap; // Numbre of grey objects space has been allocated for
    struct frame *_frames; // Stack of frames for active bytecode executions
    size_t _frames_len; // Number of active frames
    size_t _frames_cap; // Number of actives frames space has been allocated for
    // Value stack shared by all active frames.
    // Each frame uses a contiguous region of the stack, starting at the index recorded in its
    // `stack_offset` field.
    value *_stack;
    size_t _stack_cap; // Number of elements space has been allocated for in the value stack
    value *_stack_top; // Points to where the next element will be pushed on the value stack
    map _strings; // Interned strings
    // Array of global variables.
    // Must only be appended to via `vm_add_global()`.
    struct _global *_globals;
    size_t _globals_len; // Number of global variables
    size_t _globals_cap; // Number of global variables space has been allocated for
    // Head of the linked list of open upvalues.
    // The list is sorted in descending VM stack index order.
    struct object_upvalue *_open_upvalues;
    // Array where each element stores whether the local variable (if any) at the same index in the
    // value stack is a final variable. If it is, then the element is the variable name. Otherwise,
    // the element is `NULL`.
    // The size of this array mirrors the value stack.
    const char **_final_local_names;
    int64_t _interpret_calls; // Number of times `vm_interpret()` has been called.
};

// Initialises `vm` for use.
void vm_init(vm *vm);

// Frees the memory associated with `vm`.
void vm_deinit(vm *vm);

// Interprets the given Lox `source` and reports whether interpretation was successful.
// This function can be called multiple times with different sources and the state will be
// maintained between calls.
bool vm_interpret(vm *vm, const char *source);

#endif
