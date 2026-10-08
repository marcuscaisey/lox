#include "object.h"

#include <stdio.h>
#include <string.h>

#include "value.h"
#include "vm_internal.h"

void object_destroy(vm *vm, struct object *object)
{
    switch (object->_type) {
    case _OBJECT_STRING: {
        struct object_string *string = (struct object_string *)object;
        vm_mfree(vm, string, sizeof(struct object_string) + string->len);
    }
    }
}

enum value_type object_value_type(const struct object *object)
{
    // All _object_type members are equal to their equivalent value_type
    return (enum value_type)object->_type;
}

struct object_string *object_string_alloc(vm *vm, size_t len)
{
    struct object_string *string = vm_malloc(vm, sizeof(struct object_string) + len);
    string->_object.next = vm->_objects;
    vm->_objects = &string->_object;
    string->_object._type = _OBJECT_STRING;
    string->len = len;
    return string;
}

struct object_string *object_string_create(vm *vm, const char *data, size_t len)
{
    struct object_string *string = object_string_alloc(vm, len);
    memcpy(string->data, data, len);
    return string;
}

value value_string(struct object_string *string)
{
    return value_object((struct object *)string);
}

struct object_string *value_as_string(value value)
{
    return (struct object_string *)value_as_object(value);
}
