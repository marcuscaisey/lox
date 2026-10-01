#ifndef CLOX_COMPILER_H
#define CLOX_COMPILER_H

#include <stdbool.h>
#include "bytecode.h"

// Writes the bytecode for the Lox `source` into `out` and reports whether compilation was
// successful. If complication was unsuccessful, errors are printed to stderr.
bool compile(const char *source, struct bytecode_chunk *out);

#endif
