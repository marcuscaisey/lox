#include "builtins.h"

#include <ctype.h>
#include <errno.h>
#include <math.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#include "object.h"
#include "strings.h"
#include "value.h"
#include "vm_memory.h"

value builtin_clock(vm *vm, value *args, char **error_msg)
{
    (void)vm;
    (void)args;
    struct timespec tp;
    if (clock_gettime(CLOCK_MONOTONIC, &tp) < 0) {
        xasprintf(error_msg, "failed to retrieve time: %s", strerror(errno));
        return none_value;
    }
    double result = tp.tv_sec + (tp.tv_nsec / 1e9);
    return number_value(result);
}

value builtin_sleep(vm *vm, value *args, char **error_msg)
{
    (void)vm;
    value duration_value = args[0];
    if (value_type(duration_value) != VALUE_NUMBER) {
        xasprintf(error_msg, "expected sleep argument to be a '%s', got '%s'",
                  value_type_string(VALUE_NUMBER), value_type_string(value_type(duration_value)));
        return none_value;
    }
    double duration = value_as_number(duration_value);
    if (duration < 0) {
        char *duration_string = value_string(duration_value);
        xasprintf(error_msg, "expected sleep argument (%s) to be non-negative", duration_string);
        free(duration_string);
        return none_value;
    }
    if (duration == 0)
        return nil_value;
    struct timespec tp;
    tp.tv_sec = trunc(duration);
    tp.tv_nsec = (duration - tp.tv_sec) * 1e9;
    nanosleep(&tp, NULL);
    return nil_value;
}

value builtin_type(vm *vm, value *args, char **error_msg)
{
    (void)error_msg;
    value value = args[0];
    const char *type_string = value_type_string(value_type(value));
    struct object_string *result = object_string_copy(vm, type_string, strlen(type_string));
    return string_value(result);
}

value builtin_parseNumber(vm *vm, value *args, char **error_msg)
{
    (void)vm;
    value str_value = args[0];
    if (value_type(str_value) != VALUE_STRING) {
        xasprintf(error_msg, "expected parseNumber argument to be a '%s', got '%s'",
                  value_type_string(VALUE_STRING), value_type_string(value_type(str_value)));
        return none_value;
    }
    const char *str = value_as_string(str_value)->data;
    char *end = NULL;
    double result = strtod(str, &end);
    if (end != str) {
        for (char *c = end; *c != '\0'; c++)
            if (!isspace(*c))
                goto out_error;
        return number_value(result);
    }
out_error:
    {
        char *str_string = value_string(str_value);
        xasprintf(error_msg, "\"%s\" could not be parsed as a '%s'", str_string,
                  value_type_string(VALUE_NUMBER));
        free(str_string);
        return none_value;
    }
}

value builtin_string(vm *vm, value *args, char **error_msg)
{
    (void)vm;
    (void)error_msg;
    value value = args[0];
    char *data = value_string(value);
    struct object_string *result = object_string_take(vm, data, strlen(data));
    return string_value(result);
}

value builtin_error(vm *vm, value *args, char **error_msg)
{
    (void)vm;
    value msg_value = args[0];
    *error_msg = value_string(msg_value);
    return none_value;
}

value builtin_printerr(vm *vm, value *args, char **error_msg)
{
    (void)vm;
    (void)error_msg;
    value msg_value = args[0];
    char *msg = value_string(msg_value);
    fprintf(stderr, "%s\n", msg);
    free(msg);
    return nil_value;
}

value builtin_exit(vm *vm, value *args, char **error_msg)
{
    (void)vm;
    value code_value = args[0];
    if (value_type(code_value) != VALUE_NUMBER) {
        xasprintf(error_msg, "expected exit argument to be a '%s', got '%s'",
                  value_type_string(VALUE_NUMBER), value_type_string(value_type(code_value)));
        return none_value;
    }
    double code_double = value_as_number(code_value);
    int code = trunc(code_double);
    if (code != code_double) {
        char *code_string = value_string(code_value);
        xasprintf(error_msg, "expected exit argument (%s) to be an integer", code_string);
        free(code_string);
        return none_value;
    }
    exit(code);
}

const struct builtin_function builtin_functions[] = {
    { "clock", 0, builtin_clock },       { "sleep", 1, builtin_sleep },
    { "type", 1, builtin_type },         { "parseNumber", 1, builtin_parseNumber },
    { "string", 1, builtin_string },     { "error", 1, builtin_error },
    { "printerr", 1, builtin_printerr }, { "exit", 1, builtin_exit },
};

const size_t builtin_functions_len = sizeof(builtin_functions) / sizeof(struct builtin_function);
