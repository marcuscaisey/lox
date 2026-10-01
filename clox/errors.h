#ifndef CLOX_ERRORS_H
#define CLOX_ERRORS_H

// Prints an error message to stderr highlighting an invalid range of characters in `source`.
// `start` points to the first character in the range.
// `end` points to the character just after the range.
void print_invalid_range_error(const char *msg, const char *source, const char *start,
                               const char *end);

#endif
