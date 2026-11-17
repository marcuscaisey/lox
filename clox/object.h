#ifndef CLOX_OBJECT_H
#define CLOX_OBJECT_H

#include <stdbool.h>
#include <stddef.h>

#include "bytecode.h"
#include "value.h"
#include "vm_fwd.h"

// Type of an object.
enum object_type {
    OBJECT_STRING = VALUE_STRING,
    OBJECT_FUNCTION = VALUE_FUNCTION,
    OBJECT_CLOSURE = VALUE_CLOSURE,
    OBJECT_UPVALUE = -1,
    OBJECT_NATIVE_FUNCTION = VALUE_NATIVE_FUNCTION,
};

// Returns a string representation of `type`.
const char *object_type_string(enum object_type type);

// Objects are dynamically allocated values whose lifetime is managed by the VM.
// This type contains the data which is common to all object types. The struct corresponding to each
// object type embeds this type as its first field so that a pointer to it can be cast to a `struct
// object *` and back.
// Values which are represented by objects can be converted to this type using `value_as_object()`.
struct object {
    struct object *next; // Next object managed by the VM, or `NULL` if this is the last one
    bool is_reachable; // Whether this object is reachable from a VM root
    enum object_type type; // Type of the object
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
    struct object object;
    char *data; // Null-terminated string contents
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
    struct object object;
    char *name;
    argument_count arity; // Number of parameters that the function expects
    struct bytecode_chunk body; // Instructions to execute the function's body
};

// Allocates and initialises a new function, making a copy of the first `name_len` bytes of `name`
// and taking ownership of `body`.
struct object_function *object_function_new(vm *vm, const char *name, size_t name_len,
                                            argument_count arity, struct bytecode_chunk body);

value function_value(struct object_function *function);
struct object_function *value_as_function(value value);
struct object *function_object(struct object_function *function);

// Represents a Lox closure.
// Must be created with `object_closure_new()` and freed with `object_free()`.
struct object_closure {
    struct object object;
    struct object_function *function;
    struct object_upvalue **upvalues; // Array of upvalues
    upvalue_count upvalue_count; // Number of upvalues
};

// Allocates and initialises a new closure, taking ownership of the array of `upvalue_count`
// upvalues pointed to by `upvalues`.
struct object_closure *object_closure_new(vm *vm, struct object_function *function,
                                          struct object_upvalue **upvalues,
                                          upvalue_count upvalue_count);

value closure_value(struct object_closure *closure);
struct object_closure *value_as_closure(value value);

// Represents an upvalue as defined in bytecode.h.
struct object_upvalue {
    struct object object;
    struct object_upvalue *next; // Next upvalue managed by the VM, or `NULL` if this is the last one
    size_t vm_stack_index; // Index in the VM's value stack of the upvalue's value
    bool is_open;
    value value; // Upvalue's value if it's closed, otherwise `none_value`
    // Internal field, do not use.
    // Name of the variable if this upvalue represents a final variable and the upvalue is closed,
    // otherwise `NULL`.
    char *_final_local_name;
};

// Allocates and initialises a new open upvalue which aliases the value at `vm_stack_index` in the
// VM's value stack.
struct object_upvalue *object_upvalue_new_open(vm *vm, size_t vm_stack_index);

// Closes `upvalue`.
void object_upvalue_close(vm *vm, struct object_upvalue *upvalue);

// Returns the value of `upvalue`.
value object_upvalue_read(const vm *vm, struct object_upvalue *upvalue);

// Sets the value of `upvalue` to `value`.
void object_upvalue_write(vm *vm, struct object_upvalue *upvalue, value value);

// Returns the name of the variable if this upvalue represents a final variable closed, otherwise
// `NULL`.
const char *object_upvalue_final_local_name(vm *vm, struct object_upvalue *upvalue);

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
    struct object object;
    char *name;
    native_function_body body;
    argument_count arity; // Number of parameters that the function expects
};

// Allocates and initialises a new native function, making a copy of the first `name_len` bytes of
// `name`.
struct object_native_function *object_native_function_new(vm *vm, const char *name, size_t name_len,
                                                          argument_count arity,
                                                          native_function_body body);

value native_function_value(struct object_native_function *function);
struct object_native_function *value_as_native_function(value value);

// Returns a string representation of `object`.
// The caller is responsible for freeing the returned string with `free()` when it's no longer
// needed.
// The `vm` parameter is unused unless `object` is an upvalue.
char *object_string(const vm *vm, struct object *object);

#endif
