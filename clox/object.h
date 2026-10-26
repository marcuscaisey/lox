#ifndef CLOX_OBJECT_H
#define CLOX_OBJECT_H

#include <stddef.h>
#include <stdint.h>

#include "bytecode.h"
#include "value.h"

typedef struct vm vm;

// Objects represent dynamically allocated Lox values whose lifetime is managed by the VM.
// This type contains the data which is common to all object types.
// Values which are represented by objects can be converted to this type using `value_as_object()`.
//
// Implementation detail: the struct corresponding to each object type embeds this type as its first
// field so that a pointer to it can be casted to a `struct object *` and back.
struct object {
    struct object *next; // Next object managed by the VM, or `NULL` if this is the last one
    // Internal fields below, do not use.
    // Members must be equal to their corresponding value_type member
    enum _object_type {
        _OBJECT_FUNCTION = VALUE_FUNCTION,
        _OBJECT_STRING = VALUE_STRING,
    } _type; // Indicates which object type this object is embedded in
};

// Frees `object` and the memory associated with it.
void object_free(vm *vm, struct object *object);

// Returns the type of `object`.
enum value_type object_value_type(const struct object *object);

// Represents a Lox string.
// Must be created with one of the `object_string_*()` functions and freed with `object_free()`.
// The `object_string_*()` functions ensure that every call which creates a string with the same
// data will return the same string object, so pointers to created strings can be compared to
// determine equality.
struct object_string {
    struct object _object; // Internal field, do not use.
    const char *data; // Pointer to the null-terminated string data
    size_t len; // Length of the string
    hash_value _hash; // Internal field, do not use. Hash of the string data.
};

// Allocates and initialises a new string by making a copy of the first `len` bytes of `data`.
struct object_string *object_string_copy(vm *vm, const char *data, size_t len);

// Allocates and initialises a new string by taking ownership of `data` which must be
// null-terminated.
struct object_string *object_string_take(vm *vm, char *data, size_t len);

value value_string(struct object_string *string);
struct object_string *value_as_string(value value);

// Represents a Lox function.
// Must be created with `object_function_new()` and freed with `object_free()`.
struct object_function {
    struct object _object; // Internal field, do not use.
    struct object_string *name;
    struct bytecode_chunk body; // Instructions to execute the function's body
    argument_count arity; // Number of parameters that the function expects
};

// Allocates and initialises a new function, making a copy of the first `name_len` bytes of `name`
// and taking ownership of `body`.
struct object_function *object_function_new(vm *vm, const char *name, size_t name_len,
                                            struct bytecode_chunk body, argument_count arity);

value value_function(struct object_function *function);
struct object_function *value_as_function(value value);

#endif
