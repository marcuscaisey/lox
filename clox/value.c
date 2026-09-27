#include "value.h"

#include <stddef.h>
#include <stdio.h>

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
    if (array->len + 1 > array->_cap) {
        size_t current_size = array->_cap * sizeof(*array->values);
        array->_cap = array->_cap < 8 ? 8 : array->_cap * 2;
        size_t target_size = array->_cap * sizeof(*array->values);
        array->values = reallocate(array->values, current_size, target_size);
    }
    array->values[array->len++] = value;
}

void value_array_free(struct value_array *array)
{
    deallocate(array->values, array->_cap * sizeof(*array->values));
    value_array_init(array);
}
