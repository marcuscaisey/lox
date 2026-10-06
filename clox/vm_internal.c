#include "vm_internal.h"

#include <stdlib.h>

#include "memory.h"

void *vm_malloc(struct vm *vm, size_t size)
{
    (void)vm;
    return xmalloc(size);
}

void *vm_realloc(struct vm *vm, void *p, size_t current_size, size_t target_size)
{
    (void)vm;
    (void)current_size;
    return xrealloc(p, target_size);
}

void vm_mfree(struct vm *vm, void *p, size_t size)
{
    (void)vm;
    (void)size;
    free(p);
}
