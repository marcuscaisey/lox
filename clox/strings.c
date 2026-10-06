#include "strings.h"

#include <stdarg.h>
#include <stdio.h>

#include "memory.h"

int vasprintf(char **out, const char *format, va_list args)
{
    va_list args_copy;
    va_copy(args_copy, args);
    int len = vsnprintf(NULL, 0, format, args);
    va_end(args_copy);
    if (len < 0)
        return len;
    size_t size = (size_t)len + 1;
    *out = xmalloc(size);
    va_copy(args_copy, args);
    vsnprintf(*out, size, format, args);
    va_end(args_copy);
    return size;
}
