#ifndef CLOX_VALUE_H
#define CLOX_VALUE_H

#include <stdbool.h>
#include <stddef.h>

// Represents a Lox value of any type.
// The physical representation is an implementation detail. Use the following families of functions
// for working with `value`:
//   - `value_T()` constructs a Lox `T` value. Use the constant `value_nil` where a nil value is
//     required instead.
//   - `value_is_T(v)` reports whether `v` is a Lox `T` value.
//   - `value_as_T(v)` returns the Lox `T` value represented by `v`. This is not safe to call unless
//     you've verified (possibly with `value_is_T()`) that `v` is actually a Lox `T` value.
typedef struct {
    // Internal fields below, do not use.
    enum _value_type {
        _VALUE_NUMBER,
        _VALUE_BOOL,
        _VALUE_NIL,
    } _type;
    union {
        double _number; // Valid when `type == VALUE_NUMBER`.
        bool _bool; // Valid when `type == VALUE_BOOL`.
    };
} value;

// Returns a string representation of the type of `value`.
const char *value_type_string(value value);

value value_number(double n);
value value_bool(bool b);
const value value_nil;

bool value_is_number(value v);
bool value_is_bool(value v);
bool value_is_nil(value v);

double value_as_number(value v);
bool value_as_bool(value v);

// Reports whether `value` is falsey.
bool value_is_falsey(value value);

// Reports whether `a` and `b` are equal.
bool values_are_equal(value a, value b);

// Prints a Lox syntax representation of `value` with no trailing newline.
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
