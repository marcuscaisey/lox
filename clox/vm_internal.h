#ifndef CLOX_VM_INTERNAL_H
#define CLOX_VM_INTERNAL_H

#include "vm.h"

// Allocates `size` bytes of VM managed memory and returns a pointer to the allocated memory.
// The process exits on allocation failure.
// Like `xmalloc()`, except the memory is managed by the VM.
void *vm_malloc(struct vm *vm, size_t size);

// Grows or shrinks the allocation of `current_size` bytes of VM managed memory pointed to by `p` to
// `target_size` bytes and returns a pointer to the reallocated memory.
// If `p` is `NULL`, `vm_reallocate(p, current_size, target_size)` is equivalent to
// `vm_allocate(target_size)`.
// The process exits on allocation failure.
void *vm_realloc(struct vm *vm, void *p, size_t current_size, size_t target_size);

// Frees the allocation of `size` bytes of VM managed memory pointed to by `p`.
void vm_mfree(struct vm *vm, void *p, size_t size);

#endif
