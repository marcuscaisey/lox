#include "value.h"

#include <stdio.h>

#include "dynamic_array.h"
#include "memory.h"

const char *value_type_string(enum value_type type)
{
    switch (type) {
    case TYPE_NUMBER:
        return "number";
    case TYPE_BOOL:
        return "bool";
    case TYPE_NIL:
        return "nil";
    }
}

struct value value_number(double n)
{
    return (struct value){ .type = TYPE_NUMBER, .number = n };
}

struct value value_bool(bool b)
{
    return (struct value){ .type = TYPE_BOOL, .bool_ = b };
}

const struct value value_nil = (struct value){ .type = TYPE_NIL };

bool value_is_falsey(struct value value)
{
    return (value.type == TYPE_BOOL && !value.bool_) || value.type == TYPE_NIL;
}

bool values_are_equal(struct value a, struct value b)
{
    if (a.type != b.type)
        return false;
    switch (a.type) {
    case TYPE_NUMBER:
        return a.number == b.number;
    case TYPE_BOOL:
        return a.bool_ == b.bool_;
    case TYPE_NIL:
        return true;
    }
}

void value_print(struct value value)
{
    switch (value.type) {
    case TYPE_NUMBER:
        printf("%g", value.number);
        return;
    case TYPE_BOOL:
        printf(value.bool_ ? "true" : "false");
        return;
    case TYPE_NIL:
        printf("nil");
        break;
    }
}

void value_array_init(struct value_array *array)
{
    array->values = NULL;
    array->len = 0;
    array->_cap = 0;
}

void value_array_write(struct value_array *array, struct value value)
{
    DYNAMIC_ARRAY_GROW(array->values, array->_cap, array->len + 1);
    array->values[array->len++] = value;
}

void value_array_free(struct value_array *array)
{
    deallocate(array->values, DYNAMIC_ARRAY_SIZE(array->values, array->_cap));
    value_array_init(array);
}
