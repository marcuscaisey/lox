#ifndef clox_value_h
#define clox_value_h

// Represents a Lox value.
typedef double value;

// Prints a Lox syntax representation of `value` with no trailing newline.
void value_print(value value);

// Array of `values`. Elements can be accessed directly through `values` but must only be appended
// via `value_array_write`.
// Must be initialised with `value_array_init` before usage and freed with `value_array_free` after
// usage.
struct value_array {
    value *values;
    int length; // Number of elements in `values`
    // Internal fields below, do not use.
    int _capacity; // Number of elements that space has been allocated for in `values`
};

// Initialises `array` for use.
void value_array_init(struct value_array *array);

// Frees the memory associated with `array`.
void value_array_free(struct value_array *array);

// Writes `value` into `array`.
void value_array_write(struct value_array *array, value value);

#endif
