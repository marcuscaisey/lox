#ifndef CLOX_OBJECT_H
#define CLOX_OBJECT_H

#include <stddef.h>
#include <stdint.h>

#include "value.h"

typedef struct vm vm;

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
// Must be created with one of the `object_string_*()` functions and freed with `object_destroy()`.
// The `object_string_*()` functions ensure that every call which creates a string with the same
// data will return the same string object, so pointers to created strings can be compared to
// determine equality.
struct object_string {
    struct object _object; // Internal field, do not use.
    // Pointer to the string data.
    // This is not a null-terminated string, so `len` must be used to read it.
    const char *data;
    size_t len; // Number of characters in `data`
    uint32_t hash; // Hash of the string data
};

// Allocates and initialises a new string by making a copy of `data`.
struct object_string *object_string_copy(vm *vm, const char *data, size_t len);

// Allocates and initialises a new string by taking ownership of `data`.
struct object_string *object_string_take(vm *vm, char *data, size_t len);

value value_string(struct object_string *string);
struct object_string *value_as_string(value value);

#endif
