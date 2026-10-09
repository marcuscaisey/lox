#include "object.h"

#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>

#include "map.h"
#include "value.h"
#include "vm_memory.h"

void object_destroy(vm *vm, struct object *object)
{
    switch (object->_type) {
    case _OBJECT_STRING: {
        struct object_string *string = (struct object_string *)object;
        vm_mfree(vm, (void *)string->data, string->len);
        vm_mfree(vm, string, sizeof(struct object_string));
    }
    }
}

enum value_type object_value_type(const struct object *object)
{
    // All _object_type members are equal to their equivalent value_type
    return (enum value_type)object->_type;
}

uint32_t hash_string(const char *data, size_t len)
{
    // algorithm fnv-1a is
    //     hash := FNV_offset_basis
    //     for each byte_of_data to be hashed do
    //         hash := hash XOR byte_of_data
    //         hash := hash × FNV_prime
    //     return hash
    uint32_t hash = 2166136261u;
    for (const char *p = data; p < data + len; p++) {
        hash ^= (unsigned char)*p;
        hash *= 16777619;
    }
    return hash;
}

// Allocates and initialises a new string, adds it to the VM's set of managed objects, and interns
// it in the VM.
struct object_string *object_string_create(vm *vm, const char *data, size_t len, uint32_t hash)
{
    struct object_string *string = vm_malloc(vm, sizeof(struct object_string));
    string->_object.next = vm->_objects;
    vm->_objects = &string->_object;
    string->_object._type = _OBJECT_STRING;
    string->data = data;
    string->len = len;
    string->hash = hash;
    map_set(vm, &vm->_strings, string, value_nil);
    return string;
}

struct object_string *object_string_take(vm *vm, char *data, size_t len)
{
    uint32_t hash = hash_string(data, len);
    struct object_string *interned_string = map_get_key(&vm->_strings, data, len, hash);
    if (interned_string != NULL) {
        vm_mfree(vm, data, len);
        return interned_string;
    }
    return object_string_create(vm, data, len, hash);
}

struct object_string *object_string_copy(vm *vm, const char *data, size_t len)
{
    uint32_t hash = hash_string(data, len);
    struct object_string *interned_string = map_get_key(&vm->_strings, data, len, hash);
    if (interned_string != NULL)
        return interned_string;
    char *data_copy = vm_malloc(vm, len);
    memcpy(data_copy, data, len);
    return object_string_create(vm, data_copy, len, hash);
}

value value_string(struct object_string *string)
{
    return value_object((struct object *)string);
}

struct object_string *value_as_string(value value)
{
    return (struct object_string *)value_as_object(value);
}
