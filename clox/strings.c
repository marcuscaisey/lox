#include "strings.h"

#include <errno.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "errors.h"
#include "memory.h"

int vasprintf(char **out, const char *format, va_list args)
{
    va_list args_copy;
    va_copy(args_copy, args);
    int len = vsnprintf(NULL, 0, format, args_copy);
    va_end(args_copy);
    if (len < 0)
        return len;

    size_t size = (size_t)len + 1;
    *out = xmalloc(size);

    va_copy(args_copy, args);
    len = vsnprintf(*out, size, format, args_copy);
    va_end(args_copy);
    if (len < 0)
        free(out);

    return len;
}

int asprintf(char **out, const char *format, ...)
{
    va_list args;
    va_start(args, format);
    int len = vasprintf(out, format, args);
    va_end(args);
    return len;
}

void xvasprintf(char **out, const char *format, va_list args)
{
    if (vasprintf(out, format, args) < 0)
        fprintf(stderr, "clox: error formatting \"%s\"\n", format);
}

void xasprintf(char **out, const char *format, ...)
{
    va_list args;
    va_start(args, format);
    xvasprintf(out, format, args);
    va_end(args);
}

char *xstrdup(const char *s)
{
    char *result = strdup(s);
    if (result == NULL)
        exitf("clox: %s\n", strerror(errno));
    return result;
}
