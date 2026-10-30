#ifndef CLOX_STRINGS_H
#define CLOX_STRINGS_H

#include <stdarg.h>

// Works like `sprintf()`, except a new string is allocated for the output and stored in `*out`.
// The caller is responsible for freeing the string pointed to by `*out` with `free()` when it's no
// longer needed.
// Returns the length of the allocated string if successful, otherwise a negative number if there
// was an error formatting the string.
__attribute__((format(printf, 2, 3))) int asprintf(char **out, const char *format, ...);

// Works like `asprintf()`, except it exits the program if there's a formatting error.
__attribute__((format(printf, 2, 3))) void xasprintf(char **out, const char *format, ...);

// Works like `sprintf()`, except it accepts a `va_list` instead of variadic arguments.
__attribute__((format(printf, 2, 0))) int vasprintf(char **out, const char *format, va_list args);

// Works like `vasprintf()`, except it exits the program if there's a formatting error.
__attribute__((format(printf, 2, 0))) void xvasprintf(char **out, const char *format, va_list args);

// Works like `strdup()`, except it exits the program on error.
char *xstrdup(const char *s1);

#endif
