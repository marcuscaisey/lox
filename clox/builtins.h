#ifndef CLOX_BUILTINS_H
#define CLOX_BUILTINS_H

#include <stddef.h>

#include "bytecode.h"
#include "object.h"

// Represents a builtin Lox function.
struct builtin_function {
    const char *name; // Null-terminated string
    argument_count arity; // Number of parameters that the function expects
    native_function_body body;
};

// All of the builtin Lox functions.
extern const struct builtin_function builtin_functions[];

// Number of builtin Lox functions.
extern const size_t builtin_functions_len;

#endif
