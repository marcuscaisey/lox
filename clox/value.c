#include "value.h"

#include <stdio.h>
#include <stdlib.h>

#include "dynamic_array.h"

const char *value_type_string(value value)
{
    switch (value._type) {
    case _VALUE_NUMBER:
        return "number";
    case _VALUE_BOOL:
        return "bool";
    case _VALUE_NIL:
        return "nil";
    }
}

value value_number(double n)
{
    return (value){ ._type = _VALUE_NUMBER, ._number = n };
}

value value_bool(bool b)
{
    return (value){ ._type = _VALUE_BOOL, ._bool = b };
}

const value value_nil = (value){ ._type = _VALUE_NIL };

bool value_is_number(value v)
{
    return v._type == _VALUE_NUMBER;
}

bool value_is_bool(value v)
{
    return v._type == _VALUE_BOOL;
}

bool value_is_nil(value v)
{
    return v._type == _VALUE_NIL;
}

double value_as_number(value v)
{
    return v._number;
}

bool value_as_bool(value v)
{
    return v._bool;
}

bool value_is_falsey(value value)
{
    return (value._type == _VALUE_BOOL && !value._bool) || value._type == _VALUE_NIL;
}

bool values_are_equal(value a, value b)
{
    if (a._type != b._type)
        return false;
    switch (a._type) {
    case _VALUE_NIL:
        return true;
    case _VALUE_BOOL:
        return a._bool == b._bool;
    case _VALUE_NUMBER:
        return a._number == b._number;
    }
}

void value_print(value value)
{
    switch (value._type) {
    case _VALUE_NIL:
        printf("nil");
        break;
    case _VALUE_BOOL:
        printf(value._bool ? "true" : "false");
        return;
    case _VALUE_NUMBER:
        printf("%g", value._number);
        return;
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
