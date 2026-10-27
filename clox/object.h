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
        _OBJECT_STRING = VALUE_STRING,
        _OBJECT_FUNCTION = VALUE_FUNCTION,
        _OBJECT_NATIVE_FUNCTION = VALUE_NATIVE_FUNCTION,
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

value string_value(struct object_string *string);
struct object_string *value_as_string(value value);

// Represents a Lox function.
// Must be created with `object_function_new()` and freed with `object_free()`.
struct object_function {
    struct object _object; // Internal field, do not use.
    struct object_string *name;
    argument_count arity; // Number of parameters that the function expects
    struct bytecode_chunk body; // Instructions to execute the function's body
};

// Allocates and initialises a new function, making a copy of the first `name_len` bytes of `name`
// and taking ownership of `body`.
struct object_function *object_function_new(vm *vm, const char *name, size_t name_len,
                                            argument_count arity, struct bytecode_chunk body);

value function_value(struct object_function *function);
struct object_function *value_as_function(value value);

// Type of function that implements the body of a native Lox function.
// `args` is an array containing the function arguments. The caller is responsible for ensuring that
// the correct number of arguments are passed.
// If no error occurs, the function's result is returned to the Lox caller unchanged. Lox functions
// implicitly return `nil` so `nil_value` must be returned if there is no result.
// If an error occurs, the function must return `none_value` and write an error message to
// `*error_msg`. The caller is responsible for freeing the error message with `free()` when it's no
// longer needed.
typedef value (*native_function_body)(vm *vm, value *args, char **error_msg);

// Represents a native Lox function.
// Must be created with `object_native_function_new()` and freed with `object_free()`.
struct object_native_function {
    struct object _object; // Internal field, do not use.
    struct object_string *name;
    native_function_body body;
    argument_count arity; // Number of parameters that the function expects
};

// Allocates and initialises a new native function, making a copy of the first `name_len` bytes of
// `name`.
struct object_native_function *object_native_function_new(vm *vm, const char *name, size_t name_len,
                                                   argument_count arity, native_function_body body);

value native_function_value(struct object_native_function *function);
struct object_native_function *value_as_native_function(value value);

#endif
