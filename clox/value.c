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
    array->_capacity = 0;
}

void value_array_write(struct value_array *array, value value)
{
    if (array->len + 1 > array->_capacity) {
        size_t current_size = array->_capacity * sizeof(*array->values);
        array->_capacity = array->_capacity < 8 ? 8 : array->_capacity * 2;
        size_t target_size = array->_capacity * sizeof(*array->values);
        array->values = reallocate(array->values, current_size, target_size);
    }
    array->values[array->len++] = value;
}

void value_array_free(struct value_array *array)
{
    deallocate(array->values, array->_capacity * sizeof(*array->values));
    value_array_init(array);
}
