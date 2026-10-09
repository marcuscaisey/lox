#include "vm_memory.h"

#include <stdlib.h>

#include "memory.h"

void *vm_malloc(vm *vm, size_t size)
{
    (void)vm;
    return xmalloc(size);
}

void *vm_realloc(vm *vm, void *p, size_t current_size, size_t target_size)
{
    (void)vm;
    (void)current_size;
    return xrealloc(p, target_size);
}

void vm_mfree(vm *vm, void *p, size_t size)
{
    (void)vm;
    (void)size;
    free(p);
}
