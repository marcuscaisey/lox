#ifndef CLOX_VM_INTERNAL_H
#define CLOX_VM_INTERNAL_H

#include <stdbool.h>
#include <stddef.h>

#include "bytecode.h"
#include "value.h"
#include "vm_fwd.h"

// Allocates `size` bytes of VM managed memory and returns a pointer to the allocated memory.
// The process exits on allocation failure.
// Like `xmalloc()`, except the memory is managed by the VM.
void *vm_malloc(vm *vm, size_t size);

// Grows or shrinks the allocation of `current_size` bytes of VM managed memory pointed to by `p` to
// `target_size` bytes and returns a pointer to the reallocated memory.
// If `p` is `NULL`, `vm_reallocate(p, current_size, target_size)` is equivalent to
// `vm_allocate(target_size)`.
// The process exits on allocation failure.
void *vm_realloc(vm *vm, void *p, size_t current_size, size_t target_size);

// Frees the allocation of `size` bytes of VM managed memory pointed to by `p`.
void vm_free(vm *vm, void *p, size_t size);

// Represents a global variable.
// Must be created with `vm_add_global()` and deinitialised with `global_deinit()` after use.
// Internal type, do not use.
struct global {
    char *name;
    size_t name_len;
    value value; // `none_value` if the variable has not been declared yet
    bool is_final;
};

// Initialises `global` as an undeclared global variable by making a copy of the first `len` bytes
// of `name`.
void global_init(struct global *global, const char *name, size_t len);

// Frees the memory associated with `global`.
void global_deinit(struct global *global);

// Adds a global variable to `vm's` set if it doesn't exist and returns its index.
// The first `len` bytes of `name` are copied if a global is added.
global_index vm_add_global(vm *vm, const char *name, size_t len);

#endif
