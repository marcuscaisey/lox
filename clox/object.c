#include "object.h"

#include <stddef.h>
#include <string.h>

#include "bytecode.h"
#include "map.h"
#include "value.h"
#include "vm.h"
#include "vm_internal.h"

void object_free(vm *vm, struct object *object)
{
    switch (object->_type) {
    case _OBJECT_STRING: {
        struct object_string *string = (struct object_string *)object;
        vm_free(vm, string->data, string->len);
        vm_free(vm, string, sizeof(struct object_string));
        break;
    }
    case _OBJECT_FUNCTION: {
        struct object_function *function = (struct object_function *)object;
        vm_free(vm, function->name, strlen(function->name) + 1);
        bytecode_chunk_deinit(vm, &function->body);
        vm_free(vm, function, sizeof(struct object_function));
        break;
    }
    case _OBJECT_CLOSURE: {
        struct object_closure *closure = (struct object_closure *)object;
        vm_free(vm, closure->upvalues, closure->upvalue_count * sizeof(*closure->upvalues));
        vm_free(vm, closure, sizeof(struct object_closure));
        break;
    }
    case _OBJECT_UPVALUE: {
        struct object_upvalue *upvalue = (struct object_upvalue *)object;
        if (upvalue->final_local_name != NULL)
            vm_free(vm, upvalue->final_local_name, strlen(upvalue->final_local_name) + 1);
        vm_free(vm, upvalue, sizeof(struct object_upvalue));
        break;
    }
    case _OBJECT_NATIVE_FUNCTION: {
        struct object_native_function *function = (struct object_native_function *)object;
        vm_free(vm, function->name, strlen(function->name) + 1);
        vm_free(vm, function, sizeof(struct object_native_function));
        break;
    }
    }
}

enum value_type object_value_type(const struct object *object)
{
    // All _object_type members are equal to their equivalent value_type
    return (enum value_type)object->_type;
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
static struct object *object_new(vm *vm, size_t size, enum _object_type type)
{
    struct object *object = vm_xmalloc(vm, size);
    object->next = vm->_objects;
    vm->_objects = object;
    object->_type = type;
    return object;
}

// Allocates a new string, initialises it, and interns it in the VM.
static struct object_string *object_string_new(vm *vm, char *data, size_t len, hash_value hash)
{
    struct object_string *string = OBJECT_NEW(vm, object_string, _OBJECT_STRING);
    string->data = data;
    string->len = len;
    string->_hash = hash;
    map_set(vm, &vm->_strings, string_value(string), nil_value);
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
    struct object_function *function = OBJECT_NEW(vm, object_function, _OBJECT_FUNCTION);
    function->name = vm_xstrndup(vm, name, name_len);
    function->arity = arity;
    function->body = body;
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
    struct object_closure *closure = OBJECT_NEW(vm, object_closure, _OBJECT_CLOSURE);
    closure->function = function;
    closure->upvalues = upvalues;
    closure->upvalue_count = upvalue_count;
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

struct object_upvalue *object_upvalue_new_open(vm *vm, size_t vm_stack_index,
                                               const char *final_local_name)
{
    struct object_upvalue *upvalue = OBJECT_NEW(vm, object_upvalue, _OBJECT_UPVALUE);
    upvalue->next = NULL;
    upvalue->vm_stack_index = vm_stack_index;
    upvalue->_value = none_value;
    upvalue->_is_open = true;
    upvalue->final_local_name = final_local_name != NULL ? vm_xstrdup(vm, final_local_name) : NULL;
    return upvalue;
}

void object_upvalue_close(const vm *vm, struct object_upvalue *upvalue)
{
    upvalue->_value = vm->_stack[upvalue->vm_stack_index];
    upvalue->_is_open = false;
}

value object_upvalue_read(const vm *vm, struct object_upvalue *upvalue)
{
    if (upvalue->_is_open)
        return vm->_stack[upvalue->vm_stack_index];
    else
        return upvalue->_value;
}

void object_upvalue_write(vm *vm, struct object_upvalue *upvalue, value value)
{
    if (upvalue->_is_open)
        vm->_stack[upvalue->vm_stack_index] = value;
    else
        upvalue->_value = value;
}

struct object_native_function *object_native_function_new(vm *vm, const char *name, size_t name_len,
                                                          argument_count arity,
                                                          native_function_body body)
{
    struct object_native_function *function =
        OBJECT_NEW(vm, object_native_function, _OBJECT_NATIVE_FUNCTION);
    function->name = vm_xstrndup(vm, name, name_len);
    function->arity = arity;
    function->body = body;
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
