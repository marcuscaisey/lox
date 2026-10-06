#ifndef CLOX_VALUE_H
#define CLOX_VALUE_H

#include <stdbool.h>
#include <stddef.h>

struct object;

// Type of a Lox value.
enum value_type {
    VALUE_NUMBER,
    VALUE_BOOL,
    VALUE_NIL,
    VALUE_STRING,
};

// Returns a string representation of `type`.
const char *value_type_string(enum value_type type);

// Represents a Lox value of any type.
//
// For each Lox value type `T`, use the following families of functions from value.h and object.h to
// work with `T` values:
//   - `value_type(value)` returns the type `T`.
//   - `value_T(t_value)` constructs a `T` value from `t_value`. Use the constant `value_nil`
//     instead where a nil value is required.
//   - `value_as_T(value)` returns the `T` value represented by `value`. This is not safe to call
//     unless you've verified (possibly with `value_type()`) that `value` is actually a `T` value.
//
// Some Lox values are represented by objects (see object.h). Use `value_object()`,
// `value_is_object()`, and `value_as_object()` to work with this representation.
typedef struct {
    // Internal fields below, do not use.
    // Members must be equal to their corresponding value_type member apart from _VALUE_OBJECT
    enum _value_type {
        _VALUE_NUMBER = VALUE_NUMBER,
        _VALUE_BOOL = VALUE_BOOL,
        _VALUE_NIL = VALUE_NIL,
        _VALUE_OBJECT,
    } _type; ///< Indicates which of the union fields holds the actual value unless
    ///< `_type == _VALUE_NIL`, in which case there is no associated value.
    union {
        double _number; // Valid when `_type == _VALUE_NUMBER`.
        bool _bool; // Valid when `_type == _VALUE_BOOL`.
        struct object *_object; // Valid when `_type == _VALUE_OBJECT`.
    };
} value;

// Returns the type of `value`.
enum value_type value_type(value value);

value value_number(double number);
value value_bool(bool bool_);
const value value_nil;

double value_as_number(value value);
bool value_as_bool(value value);

value value_object(struct object *object);
bool value_is_object(value value);
struct object *value_as_object(value value);

// Reports whether `a` and `b` are equal.
bool values_are_equal(value a, value b);

// Reports whether `value` is falsey.
bool value_is_falsey(value value);

// Prints `value` in the format of the Lox `print` statement with no trailing newline.
void value_print(value value);

// Array of `values`. Elements can be accessed directly through `values` but must only be appended
// via `value_array_write()`.
// Must be initialised with `value_array_init()` before use and freed with `value_array_free()`
// after use.
struct value_array {
    value *values;
    size_t len; // Number of elements in `values`
    // Internal fields below, do not use.
    size_t _cap; // Number of elements that space has been allocated for in `values`
};

// Initialises `array` for use.
void value_array_init(struct value_array *array);

// Frees the memory associated with `array`.
void value_array_free(struct value_array *array);

// Writes `value` into `array`.
void value_array_write(struct value_array *array, value value);

#endif
