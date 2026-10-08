#ifndef CLOX_OBJECT_H
#define CLOX_OBJECT_H

#include <stddef.h>

#include "value.h"
#include "vm.h"

// Objects represent dynamically allocated Lox values whose lifetime is managed by the VM.
// This type contains the data which is common to all object types.
// Values which are represented by objects can be converted to this type using `value_as_object()`.
struct object {
    struct object *next; // Next object managed by the VM, or `NULL` if this is the last one
    // Internal fields below, do not use.
    // Members must be equal to their corresponding value_type member
    enum _object_type {
        _OBJECT_STRING = VALUE_STRING,
    } _type; // Indicates which object type this object is embedded in
};

// Frees `object` and the memory associated with it.
void object_destroy(vm *vm, struct object *object);

// Returns the type of `object`.
enum value_type object_value_type(const struct object *object);

// Represents a Lox string.
// Must be created with one of the `object_string_*()` functions.
struct object_string {
    struct object _object; // Internal field, do not use.
    size_t len; // Number of characters in `data`
    // Array of string data.
    // This is not a null-terminated string, so `len` must be used to read it.
    char data[];
};

// Allocates and initialises a new string by copying the first `len` characters of `data`.
struct object_string *object_string_create(vm *vm, const char *data, size_t len);

// Allocates a new string, leaving `data` unitialised.
struct object_string *object_string_alloc(vm *vm, size_t len);

value value_string(struct object_string *string);
struct object_string *value_as_string(value value);

#endif
