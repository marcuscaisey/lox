#include "value.h"

#include <stddef.h>
#include <stdio.h>

#include "dynamic_array.h"
#include "memory.h"

void value_print(value value)
{
    printf("%g", value);
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
    deallocate(array->values, DYNAMIC_ARRAY_SIZE(array->values, array->_cap));
    value_array_init(array);
}
