#include "value.h"

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "dynamic_array.h"
#include "object.h"

const value value_none = (value){ ._type = _VALUE_NONE };

value value_number(double number)
{
    return (value){ ._type = _VALUE_NUMBER, ._number = number };
}

value value_bool(bool bool_)
{
    return (value){ ._type = _VALUE_BOOL, ._bool = bool_ };
}

const value value_nil = (value){ ._type = _VALUE_NIL };

value value_object(struct object *object)
{
    return (value){ ._type = _VALUE_OBJECT, ._object = object };
}

bool value_is_object(value value)
{
    return value._type == _VALUE_OBJECT;
}

double value_as_number(value value)
{
    return value._number;
}

bool value_as_bool(value value)
{
    return value._bool;
}

struct object *value_as_object(value value)
{
    return value._object;
}

enum value_type value_type(value value)
{
    if (value._type == _VALUE_OBJECT)
        return object_value_type(value_as_object(value));
    // All _value_type members apart from _VALUE_OBJECT are equal their equivalent value_type
    return (enum value_type)value._type;
}

const char *value_type_string(value value)
{
    switch (value_type(value)) {
    case VALUE_NONE:
        return "NONE";
    case VALUE_NUMBER:
        return "number";
    case VALUE_BOOL:
        return "bool";
    case VALUE_NIL:
        return "nil";
    case VALUE_STRING:
        return "string";
    }
}

bool values_are_equal(value a, value b)
{
    if (value_type(a) != value_type(b))
        return false;
    switch (value_type(a)) {
    case VALUE_NUMBER:
        return value_as_number(a) == value_as_number(b);
    case VALUE_BOOL:
        return value_as_bool(a) == value_as_bool(b);
    case VALUE_NONE:
    case VALUE_NIL:
        return true;
    case VALUE_STRING:
        return value_as_object(a) == value_as_object(b);
    }
}

// Fixed hash values chosen so that they're distinct
#define HASH_VALUE_NONE 0
#define HASH_VALUE_ZERO 1
#define HASH_VALUE_TRUE 2
#define HASH_VALUE_FALSE 3
#define HASH_VALUE_NIL 4

hash_value hash_number(double value)
{
    // Lox (as C does) compares -0 and 0 equal we assign the same hash value to both
    if (value == 0)
        return HASH_VALUE_ZERO;
    // algorithm fnv-1a is
    //     hash := FNV_offset_basis
    //     for each byte_of_data to be hashed do
    //         hash := hash XOR byte_of_data
    //         hash := hash × FNV_prime
    //     return hash
    hash_value hash = 2166136261u;
    uint64_t bits;
    memcpy(&bits, &value, sizeof(bits));
    for (int i = 7; i >= 0; i--) {
        unsigned char byte = (bits >> (i * 8)) & 0xFF;
        hash ^= byte;
        hash *= 16777619;
    }
    return hash;
}

hash_value value_hash(value value)
{
    switch (value_type(value)) {
    case VALUE_NONE:
        return HASH_VALUE_NONE;
    case VALUE_NUMBER:
        return hash_number(value_as_number(value));
    case VALUE_BOOL:
        return value_as_bool(value) ? HASH_VALUE_TRUE : HASH_VALUE_FALSE;
    case VALUE_NIL:
        return HASH_VALUE_NIL;
    case VALUE_STRING: {
        struct object_string *string = value_as_string(value);
        return string->_hash;
    }
    }
}

bool value_is_falsey(value value)
{
    return (value_type(value) == VALUE_BOOL && !value_as_bool(value)) ||
           value_type(value) == VALUE_NIL;
}

void value_print(value value)
{
    switch (value_type(value)) {
    case VALUE_NONE:
        printf("NONE");
        break;
    case VALUE_NUMBER:
        printf("%g", value_as_number(value));
        break;
    case VALUE_BOOL:
        printf(value_as_bool(value) ? "true" : "false");
        break;
    case VALUE_NIL:
        printf("nil");
        break;
    case VALUE_STRING: {
        struct object_string *string = value_as_string(value);
        printf("%s", string->data);
        break;
    }
    }
}

void value_array_init(struct value_array *array)
{
    array->values = NULL;
    array->len = 0;
    array->_cap = 0;
}

void value_array_write(struct value_array *array, value value)
{
    DYNAMIC_ARRAY_GROW(array->values, array->_cap, array->len + 1);
    array->values[array->len++] = value;
}

void value_array_free(struct value_array *array)
{
    free(array->values);
    value_array_init(array);
}
