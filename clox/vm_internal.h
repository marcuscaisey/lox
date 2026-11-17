#ifndef CLOX_VM_INTERNAL_H
#define CLOX_VM_INTERNAL_H

#include <stdbool.h>
#include <stddef.h>

#include "value.h"
#include "vm_fwd.h" // IWYU pragma: export

// Works like `xmalloc()`, except it allocates VM managed memory.
void *vm_xmalloc(vm *vm, size_t size);

// Works like `xrealloc()`, execpt it allocates VM managed memory.
void *vm_xrealloc(vm *vm, void *p, size_t current_size, size_t target_size);

// Frees the allocation of `size` bytes of VM managed memory pointed to by `p`.
void vm_free(vm *vm, void *p, size_t size);

// Works like `xstrdup()`, except it allocates VM managed memory.
char *vm_xstrdup(vm *vm, const char *s);

// Works like `vm_xstrndup()`, execpt it allocates VM managed memory.
char *vm_xstrndup(vm *vm, const char *s, size_t n);

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
size_t vm_add_global(vm *vm, const char *name, size_t len);

#endif
