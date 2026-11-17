#include "vm_internal.h"

#include <stddef.h>
#ifdef DEBUG_TRACE_GC
#include <stdio.h>
#endif
#include <stdlib.h>
#include <string.h>

#include "bytecode.h"
#include "dynamic_array.h"
#include "map.h"
#include "memory.h"
#include "object.h"
#include "strings.h"
#include "value.h"
#include "vm.h"

void global_init(struct _global *global, const char *name, size_t len)
{
    global->name = xstrndup(name, len);
    global->name_len = len;
    global->value = none_value;
    global->is_final = false;
}

void global_deinit(struct _global *global)
{
    free(global->name);
}

size_t vm_add_global(vm *vm, const char *name, size_t len)
{
    for (size_t i = 0; i < vm->_globals_len; i++) {
        struct _global global = vm->_globals[i];
        if (len == global.name_len && strncmp(name, global.name, len) == 0)
            return i;
    }
    DYNAMIC_ARRAY_GROW(vm->_globals, vm->_globals_cap, vm->_globals_len + 1);
    size_t index = vm->_globals_len;
    struct _global *global = &vm->_globals[vm->_globals_len++];
    global_init(global, name, len);
    return index;
}

// Marks `object` as reachable and adds it to the grey stack if it was previously unmarked (white).
static void vm_mark_object(vm *vm, struct object *object)
{
    if (object->is_reachable)
        return;

    object->is_reachable = true;
#ifdef DEBUG_TRACE_GC
    char *string = object_string(vm, object);
    fprintf(stderr, "GC: marked %s '%s' (%p)\n", object_type_string(object->type), string,
            (void *)object);
    free(string);
#endif

    // Some object types don't reference any other object types, so there's no need to add them to
    // the grey stack.
    switch (object->type) {
    case OBJECT_STRING:
    case OBJECT_NATIVE_FUNCTION:
        return;
    default:
        break;
    }

    DYNAMIC_ARRAY_GROW(vm->_grey_objects, vm->_grey_objects_cap, vm->_grey_objects_len + 1);
    vm->_grey_objects[vm->_grey_objects_len++] = object;
}

// Marks (see `vm_mark_object()`) the objects referenced by `chunk`.
static void vm_mark_chunk(vm *vm, const struct bytecode_chunk *chunk)
{
    for (size_t i = 0; i < chunk->constants_len; i++) {
        value value = chunk->constants[i];
        if (value_is_object(value))
            vm_mark_object(vm, value_as_object(value));
    }
}

// Marks (see `vm_mark_object()`) the objects referenced by `object`.
static void vm_mark_references(vm *vm, struct object *object)
{
#ifdef DEBUG_TRACE_GC
    char *string = object_string(vm, object);
    fprintf(stderr, "GC: marking references of %s %s (%p)\n", object_type_string(object->type),
            string, (void *)object);
    free(string);
#endif
    switch (object->type) {
    case OBJECT_FUNCTION: {
        struct object_function *function = (struct object_function *)object;
        vm_mark_chunk(vm, &function->body);
        break;
    }
    case OBJECT_CLOSURE: {
        struct object_closure *closure = (struct object_closure *)object;
        vm_mark_object(vm, (struct object *)closure->function);
        for (upvalue_count i = 0; i < closure->upvalue_count; i++)
            vm_mark_object(vm, (struct object *)closure->upvalues[i]);
        break;
    }
    case OBJECT_UPVALUE: {
        struct object_upvalue *upvalue = (struct object_upvalue *)object;
        value value = upvalue->is_open ? vm->_stack[upvalue->vm_stack_index] : upvalue->value;
        if (value_is_object(value))
            vm_mark_object(vm, value_as_object(value));
        break;
    }
    case OBJECT_STRING:
    case OBJECT_NATIVE_FUNCTION:
        break;
    }
}

// Multiple that the total number of bytes of VM managed memory has to increase to after a
// collection cycle before the next one will run. I.e. if it's 3, then the allocated bytes has to
// increase to 3x what it was after the last collection.
static int gc_allocated_bytes_growth_factor = 2;

void vm_collect_garbage(vm *vm)
{
    // This function implements tracing garbage collection using tri-colour marking as described at
    // https://en.wikipedia.org/wiki/Tracing_garbage_collection#Tri-color_marking
    if (!vm->_is_gc_enabled)
        return;

#ifdef DEBUG_TRACE_GC
    fprintf(stderr, "GC: starting cycle\n");
    size_t prev_allocated_bytes = vm->_allocated_bytes;
#endif

    // Mark objects reachable from each of the VM's roots and move them to the grey set
    for (size_t i = 0; i < vm->_frames_len; i++) {
        struct frame frame = vm->_frames[i];
        if (frame.function != NULL)
            vm_mark_object(vm, (struct object *)frame.function);
        vm_mark_chunk(vm, frame._chunk);
    }
    for (value *value = vm->_stack; value < vm->_stack_top; value++)
        if (value_is_object(*value))
            vm_mark_object(vm, value_as_object(*value));
    for (size_t i = 0; i < vm->_globals_len; i++) {
        struct _global global = vm->_globals[i];
        if (value_is_object(global.value))
            vm_mark_object(vm, value_as_object(global.value));
    }
    for (struct object_upvalue *upvalue = vm->_open_upvalues; upvalue != NULL;
         upvalue = upvalue->next)
        vm_mark_object(vm, (struct object *)upvalue);

    // Trace the references of the grey objects and move them to the black set (objects are
    // implicitly in the black set if they're reachable, traced, and not in the grey set)
    while (vm->_grey_objects_len > 0) {
        struct object *object = vm->_grey_objects[--vm->_grey_objects_len];
        vm_mark_references(vm, object);
    }

    // Remove interned strings from the map which are about to be freed
    MAP_FOR_EACH_KEY(&vm->_strings, key, {
        struct object *key_object = value_as_object(key);
        if (!key_object->is_reachable)
            map_delete(&vm->_strings, key);
    });

    // Iterate over all objects, freeing the ones still in the white set (not reachable), and
    // unmarking the ones in the black set (reachable) so that they're ready for the next collection
    struct object **next_ptr = &vm->_objects;
    while (*next_ptr != NULL) {
        if ((*next_ptr)->is_reachable) {
            (*next_ptr)->is_reachable = false;
            next_ptr = &(*next_ptr)->next;
        } else {
            struct object *object = *next_ptr;
            *next_ptr = (*next_ptr)->next;
            object_free(vm, object);
        }
    }

    vm->_next_gc_bytes = vm->_allocated_bytes * gc_allocated_bytes_growth_factor;

#ifdef DEBUG_TRACE_GC
    fprintf(stderr, "GC: finished cycle, ");
    size_t collected_bytes = prev_allocated_bytes - vm->_allocated_bytes;
    fprintf(stderr, "collected %zu bytes", collected_bytes);
    if (collected_bytes > 0)
        fprintf(stderr, " (from %zu to %zu)", prev_allocated_bytes, vm->_allocated_bytes);
#ifndef DEBUG_STRESS_GC
    fprintf(stderr, ", next collection is at %zu\n", vm->_next_gc_bytes);
#endif
    fprintf(stderr, "\n");
#endif
}

void *vm_xmalloc(vm *vm, size_t size)
{
#ifndef DEBUG_STRESS_GC
    if (vm->_allocated_bytes + size > vm->_next_gc_bytes)
#endif
        vm_collect_garbage(vm);
    vm->_allocated_bytes += size;
    return xmalloc(size);
}

void *vm_xrealloc(vm *vm, void *p, size_t current_size, size_t target_size)
{
    ptrdiff_t size_delta = target_size - current_size;
#ifndef DEBUG_STRESS_GC
    if (vm->_allocated_bytes + size_delta > vm->_next_gc_bytes)
#endif
        vm_collect_garbage(vm);
    vm->_allocated_bytes += size_delta;
    return xrealloc(p, target_size);
}

void vm_free(vm *vm, void *p, size_t size)
{
    vm->_allocated_bytes -= size;
    free(p);
}

char *vm_xstrdup(vm *vm, const char *s)
{
    return vm_xstrndup(vm, s, strlen(s));
}

char *vm_xstrndup(vm *vm, const char *s, size_t n)
{
    size_t len = 0;
    while (len < n && s[len] != '\0')
        len++;
    char *copy = vm_xmalloc(vm, len + 1);
    memcpy(copy, s, len);
    copy[len] = '\0';
    return copy;
}
