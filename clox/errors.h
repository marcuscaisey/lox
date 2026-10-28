#ifndef CLOX_ERRORS_H
#define CLOX_ERRORS_H

#include <stddef.h>

#include "line_number.h"
#include "vm.h"

// Prints an error message to stderr highlighting an invalid range of characters in `source`.
// `start` points to the first character in the range.
// `end` points to the character just after the range.
void print_invalid_range_error(const char *msg, const char *source, const char *start,
                               const char *end);

// Prints an error message to stderr highlighting the invalid `line` in `source`.
void print_invalid_line_error(const char *msg, const char *source, line_number line);

// Prints a stack trace to stderr displaying the lines from `source` related to the `n` provided
// frames. `frames` should be sorted from least to most recent frame.
void print_stack_trace(const struct frame *frames, size_t n, const char *source);

// Exits the program with an error message.
__attribute__((format(printf, 1, 2), noreturn)) void exitf(const char *format, ...);

#endif
