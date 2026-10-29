#ifndef CLOX_VALUE_H
#define CLOX_VALUE_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

// Type of a Lox value.
enum value_type {
    // This doesn't corresond to a Lox value type. See `none_value` for what this represents.
    VALUE_NONE,
    VALUE_NUMBER,
    VALUE_BOOL,
    VALUE_NIL,
    VALUE_STRING,
    VALUE_FUNCTION,
    VALUE_NATIVE_FUNCTION,
};

// Returns a string representation of `type`.
const char *value_type_string(enum value_type type);

// Represents a Lox value of any type.
//
// For each Lox value type `T`, use the following families of functions from value.h and object.h to
// work with `T` values:
//   - `value_type(value)` returns the type `T`.
//   - `T_value(t_value)` constructs a `T` value from `t_value`. Use the constant `nil_value`
//     instead where a `nil` value is required.
//   - `value_as_T(value)` returns the `T` value represented by `value`. This is not safe to call
//     unless you've verified (possibly with `value_type()`) that `value` is actually a `T` value.
//
// Some Lox values are represented by objects (see object.h). Use `object_value()`,
// `value_is_object()`, and `value_as_object()` to work with this representation.
typedef struct {
    // Internal fields below, do not use.
    // Members must be equal to their corresponding value_type member apart from _VALUE_OBJECT
    enum _value_type {
        _VALUE_NONE = VALUE_NONE,
        _VALUE_NUMBER = VALUE_NUMBER,
        _VALUE_BOOL = VALUE_BOOL,
        _VALUE_NIL = VALUE_NIL,
        _VALUE_OBJECT,
    } _type; ///< Indicates which of the union fields holds the actual value unless
    ///< it's `_VALUE_EMPTY` or `_VALUE_NIL`, in which case there is no associated value.
    union {
        double _number; // Valid when `_type == _VALUE_NUMBER`.
        bool _bool; // Valid when `_type == _VALUE_BOOL`.
        struct object *_object; // Valid when `_type == _VALUE_OBJECT`.
    };
} value;

// Sentinel which represents a `value` with no associated value, similar to `NULL` for a pointer
// type. The type of this `value` is `VALUE_NONE`.
// This is distinct from `nil_value` which represents a Lox `nil`.
extern const value none_value;

value number_value(double number);
value bool_value(bool bool_);
extern const value nil_value;

double value_as_number(value value);
bool value_as_bool(value value);

value object_value(struct object *object);
bool value_is_object(value value);
struct object *value_as_object(value value);

// Returns the type of `value`.
enum value_type value_type(value value);

// Reports whether `a` and `b` are equal.
bool values_are_equal(value a, value b);

// Represents the hash value of a `value`.
typedef uint32_t hash_value;

// Returns the hash of `value`.
hash_value value_hash(value value);

// Reports whether `value` is truthy.
bool value_is_truthy(value value);

// Returns a string representation of `value`.
// The caller is responsible for freeing the returned string with `free()` when it's no longer
// needed.
char *value_string(value value);

#endif
