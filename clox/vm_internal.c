#include "vm_internal.h"

#include <stdlib.h>
#include <string.h>

#include "bytecode.h"
#include "dynamic_array.h"
#include "memory.h"
#include "strings.h"
#include "value.h"
#include "vm.h"

void global_init(struct global *global, const char *name, size_t len)
{
    global->name = xstrndup(name, len);
    global->name_len = len;
    global->value = none_value;
    global->is_final = false;
}

void global_deinit(struct global *global)
{
    free(global->name);
}

global_index vm_add_global(vm *vm, const char *name, size_t len)
{
    for (size_t i = 0; i < vm->_globals_len; i++) {
        struct global global = vm->_globals[i];
        if (len == global.name_len && strncmp(name, global.name, len) == 0)
            return i;
    }
    DYNAMIC_ARRAY_GROW(vm->_globals, vm->_globals_cap, vm->_globals_len + 1);
    global_index index = vm->_globals_len;
    struct global *global = &vm->_globals[vm->_globals_len++];
    global_init(global, name, len);
    return index;
}

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

void vm_free(vm *vm, void *p, size_t size)
{
    (void)vm;
    (void)size;
    free(p);
}
