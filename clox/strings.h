#ifndef CLOX_STRINGS_H
#define CLOX_STRINGS_H

#include <stdarg.h>

// Works like `vsprintf()`, except a new string is allocated for the output and stored in `*out`.
// The caller is responsible for freeing the string pointed to by `*out` with `free()` when it's no
// longer needed.
// Returns the size of the allocated string if successful, otherwise a negative number.
int vasprintf(char **out, const char *format, va_list ap);

#endif
