#ifndef CLOX_VALUE_H
#define CLOX_VALUE_H

#include <stdbool.h>

// Type of a Lox value.
enum value_type {
    TYPE_NUMBER,
    TYPE_BOOL,
    TYPE_NIL,
};

// Returns a string representation of `type`.
const char *value_type_string(enum value_type type);

// Represents a Lox value.
struct value {
    enum value_type type;
    union {
        double number;
        bool bool_;
    };
};

// Constructs a number value with value `n`.
struct value value_number(double n);

// Constructs a bool value with value `b`.
struct value value_bool(bool b);

// Nil value
const struct value value_nil;

// Reports whether `value` is falsey.
bool value_is_falsey(struct value value);

// Reports whether `a` and `b` are equal.
bool values_are_equal(struct value a, struct value b);

// Prints a Lox syntax representation of `value` with no trailing newline.
void value_print(struct value value);

// Array of `values`. Elements can be accessed directly through `values` but must only be appended
// via `value_array_write()`.
// Must be initialised with `value_array_init()` before use and freed with `value_array_free()`
// after use.
struct value_array {
    struct value *values;
    int len; // Number of elements in `values`
    // Internal fields below, do not use.
    int _cap; // Number of elements that space has been allocated for in `values`
};

// Initialises `array` for use.
void value_array_init(struct value_array *array);

// Frees the memory associated with `array`.
void value_array_free(struct value_array *array);

// Writes `value` into `array`.
void value_array_write(struct value_array *array, struct value value);

#endif
