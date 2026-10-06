// The functions in this module manage memory owned by the garbage collector. All allocations made
// by the VM must go through this module so that they can be garbage collected.

#ifndef CLOX_MEMORY_H
#define CLOX_MEMORY_H

#include <stddef.h>

// Works like `malloc()`, except it exits the program on error.
void *xmalloc(size_t size);

// Works like `realloc()`, except it exits the program on error.
void *xrealloc(void *p, size_t size);

// TODO: explicitly attach these to the vm
// Allocates `size` bytes of memory and returns a pointer to the allocated memory.
// The process exits on allocation failure.
void *allocate(size_t size);

// Grows or shrinks the allocation of `current_size` bytes pointed to by `p` to `target_size` bytes
// and returns a pointer to the reallocated memory.
// If `p` is `NULL`, `reallocate(p, current_size, target_size)` is equivalent to
// `allocate(target_size)`.
// The process exits on allocation failure.
void *reallocate(void *p, size_t current_size, size_t target_size);

// Frees the allocation of `size` bytes pointed to by `p`.
void deallocate(void *p, size_t size);

#endif
