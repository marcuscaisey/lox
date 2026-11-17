#include "object.h"

#include <stddef.h>
#include <string.h>
#ifdef DEBUG_TRACE_GC
#include <stdio.h>
#include <stdlib.h>
#endif

#include "bytecode.h"
#include "map.h"
#include "strings.h"
#include "value.h"
#include "vm.h"
#include "vm_internal.h"

const char *object_type_string(enum object_type type)
{
    switch (type) {
    case OBJECT_STRING:
        return "string";
    case OBJECT_FUNCTION:
        return "function";
    case OBJECT_CLOSURE:
        return "closure";
    case OBJECT_UPVALUE:
        return "upvalue";
    case OBJECT_NATIVE_FUNCTION:
        return "native function";
    }
}

void object_free(vm *vm, struct object *object)
{
#ifdef DEBUG_TRACE_GC
    char *string = object_string(vm, object);
    fprintf(stderr, "GC: freeing %s '%s' (%p)\n", object_type_string(object->type), string,
            (void *)object);
    free(string);
#endif
    switch (object->type) {
    case OBJECT_STRING: {
        struct object_string *string = (struct object_string *)object;
        vm_free(vm, string->data, string->len + 1);
        vm_free(vm, string, sizeof(struct object_string));
        break;
    }
    case OBJECT_FUNCTION: {
        struct object_function *function = (struct object_function *)object;
        vm_free(vm, function->name, strlen(function->name) + 1);
        bytecode_chunk_deinit(vm, &function->body);
        vm_free(vm, function, sizeof(struct object_function));
        break;
    }
    case OBJECT_CLOSURE: {
        struct object_closure *closure = (struct object_closure *)object;
        vm_free(vm, closure->upvalues, closure->upvalue_count * sizeof(*closure->upvalues));
        vm_free(vm, closure, sizeof(struct object_closure));
        break;
    }
    case OBJECT_UPVALUE: {
        struct object_upvalue *upvalue = (struct object_upvalue *)object;
        if (upvalue->_final_local_name)
            vm_free(vm, upvalue->_final_local_name, strlen(upvalue->_final_local_name) + 1);
        vm_free(vm, upvalue, sizeof(struct object_upvalue));
        break;
    }
    case OBJECT_NATIVE_FUNCTION: {
        struct object_native_function *function = (struct object_native_function *)object;
        vm_free(vm, function->name, strlen(function->name) + 1);
        vm_free(vm, function, sizeof(struct object_native_function));
        break;
    }
    }
}

enum value_type object_value_type(const struct object *object)
{
    // All object_type members are equal to their equivalent value_type
    return (enum value_type)object->type;
}

static hash_value hash_string(const char *data, size_t len)
{
    // algorithm fnv-1a is
    //     hash := FNV_offset_basis
    //     for each byte_of_data to be hashed do
    //         hash := hash XOR byte_of_data
    //         hash := hash × FNV_prime
    //     return hash
    hash_value hash = 2166136261u;
    for (const char *p = data; p < data + len; p++) {
        hash ^= (unsigned char)*p;
        hash *= 16777619;
    }
    return hash;
}

// Allocates space for the struct with `name` corresponding to the `enum _object_type` `type` and
// initialises its object field.
#define OBJECT_NEW(vm, name, type) (struct name *)object_new(vm, sizeof(struct name), (type))

// Allocates `size` bytes of memory and initialises it with an object of the given `type`.
// Use the `OBJECT_NEW()` macro instead.
static struct object *object_new(vm *vm, size_t size, enum object_type type)
{
    struct object *object = vm_xmalloc(vm, size);
    object->next = vm->_objects;
    object->is_reachable = false;
    vm->_objects = object;
    object->type = type;
    return object;
}

#ifdef DEBUG_TRACE_GC
#define LOG_OBJECT_ALLOCATION(obj)                                                                 \
    do {                                                                                    \
        char *obj_string = object_string(vm, (struct object *)obj);                      \
        fprintf(stderr, "GC: allocated %s '%s' (%p)\n",                                     \
                object_type_string(((struct object *)obj)->type), obj_string, (void *)obj); \
        free(obj_string);                                                                   \
    } while (false)
#else
#define LOG_OBJECT_ALLOCATION(obj)
#endif

// Allocates a new string, initialises it, and interns it in the VM.
static struct object_string *object_string_new(vm *vm, char *data, size_t len, hash_value hash)
{
    struct object_string *string = OBJECT_NEW(vm, object_string, OBJECT_STRING);
    string->data = data;
    string->len = len;
    string->_hash = hash;
    map_set(vm, &vm->_strings, string_value(string), nil_value);
    LOG_OBJECT_ALLOCATION(string);
    return string;
}

struct object_string *object_string_take(vm *vm, char *data, size_t len)
{
    hash_value hash = hash_string(data, len);
    struct object_string *interned_string = map_get_string_key(&vm->_strings, data, len, hash);
    if (interned_string != NULL) {
        vm_free(vm, data, len);
        return interned_string;
    }
    return object_string_new(vm, data, len, hash);
}

struct object_string *object_string_copy(vm *vm, const char *data, size_t len)
{
    hash_value hash = hash_string(data, len);
    struct object_string *interned_string = map_get_string_key(&vm->_strings, data, len, hash);
    if (interned_string != NULL)
        return interned_string;
    char *data_copy = vm_xmalloc(vm, len + 1);
    memcpy(data_copy, data, len);
    data_copy[len] = '\0';
    return object_string_new(vm, data_copy, len, hash);
}

value string_value(struct object_string *string)
{
    return object_value((struct object *)string);
}

struct object_string *value_as_string(value value)
{
    return (struct object_string *)value_as_object(value);
}

struct object_function *object_function_new(vm *vm, const char *name, size_t name_len,
                                            argument_count arity, struct bytecode_chunk body)
{
    // Copy name before creating the object to avoid triggering a collection of the object
    char *name_copy = vm_xstrndup(vm, name, name_len);
    struct object_function *function = OBJECT_NEW(vm, object_function, OBJECT_FUNCTION);
    function->name = name_copy;
    function->arity = arity;
    function->body = body;
    LOG_OBJECT_ALLOCATION(function);
    return function;
}

value function_value(struct object_function *function)
{
    return object_value((struct object *)function);
}

struct object_function *value_as_function(value value)
{
    return (struct object_function *)value_as_object(value);
}

struct object_closure *object_closure_new(vm *vm, struct object_function *function,
                                          struct object_upvalue **upvalues,
                                          upvalue_count upvalue_count)
{
    struct object_closure *closure = OBJECT_NEW(vm, object_closure, OBJECT_CLOSURE);
    closure->function = function;
    closure->upvalues = upvalues;
    closure->upvalue_count = upvalue_count;
    LOG_OBJECT_ALLOCATION(closure);
    return closure;
}

value closure_value(struct object_closure *closure)
{
    return object_value((struct object *)closure);
}

struct object_closure *value_as_closure(value value)
{
    return (struct object_closure *)value_as_object(value);
}

struct object_upvalue *object_upvalue_new_open(vm *vm, size_t vm_stack_index)
{
    struct object_upvalue *upvalue = OBJECT_NEW(vm, object_upvalue, OBJECT_UPVALUE);
    upvalue->next = NULL;
    upvalue->vm_stack_index = vm_stack_index;
    upvalue->is_open = true;
    upvalue->value = none_value;
    upvalue->_final_local_name = NULL;
    LOG_OBJECT_ALLOCATION(upvalue);
    return upvalue;
}

void object_upvalue_close(vm *vm, struct object_upvalue *upvalue)
{
    upvalue->is_open = false;
    upvalue->value = vm->_stack[upvalue->vm_stack_index];
    const char *final_local_name = vm->_final_local_names[upvalue->vm_stack_index];
    upvalue->_final_local_name = final_local_name != NULL ? vm_xstrdup(vm, final_local_name) : NULL;
}

value object_upvalue_read(const vm *vm, struct object_upvalue *upvalue)
{
    if (upvalue->is_open)
        return vm->_stack[upvalue->vm_stack_index];
    else
        return upvalue->value;
}

void object_upvalue_write(vm *vm, struct object_upvalue *upvalue, value value)
{
    if (upvalue->is_open)
        vm->_stack[upvalue->vm_stack_index] = value;
    else
        upvalue->value = value;
}

const char *object_upvalue_final_local_name(vm *vm, struct object_upvalue *upvalue)
{
    return upvalue->is_open ? vm->_final_local_names[upvalue->vm_stack_index] :
                              upvalue->_final_local_name;
}

struct object_native_function *object_native_function_new(vm *vm, const char *name, size_t name_len,
                                                          argument_count arity,
                                                          native_function_body body)
{
    // Copy name before creating the object to avoid triggering a collection of the object
    char *name_copy = vm_xstrndup(vm, name, name_len);
    struct object_native_function *function =
        OBJECT_NEW(vm, object_native_function, OBJECT_NATIVE_FUNCTION);
    function->name = name_copy;
    function->arity = arity;
    function->body = body;
    LOG_OBJECT_ALLOCATION(function);
    return function;
}

value native_function_value(struct object_native_function *function)
{
    return object_value((struct object *)function);
}

struct object_native_function *value_as_native_function(value value)
{
    return (struct object_native_function *)value_as_object(value);
}

char *object_string(const struct vm *vm, struct object *object)
{
    switch (object->type) {
    case OBJECT_STRING: {
        struct object_string *string = (struct object_string *)object;
        return xstrdup(string->data);
    }
    case OBJECT_FUNCTION: {
        struct object_function *function = (struct object_function *)object;
        char *out;
        xasprintf(&out, "[function %s]", function->name);
        return out;
    }
    case OBJECT_CLOSURE: {
        struct object_closure *closure = (struct object_closure *)object;
        return object_string(vm, (struct object *)(closure->function));
    }
    case OBJECT_UPVALUE: {
        struct object_upvalue *upvalue = (struct object_upvalue *)object;
        value value = upvalue->is_open ? vm->_stack[upvalue->vm_stack_index] : upvalue->value;
        return value_string(value);
    }
    case OBJECT_NATIVE_FUNCTION: {
        struct object_native_function *function = (struct object_native_function *)object;
        char *out;
        xasprintf(&out, "[built-in function %s]", function->name);
        return out;
    }
    }
}
