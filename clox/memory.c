#include "memory.h"

#include <errno.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "errors.h"

void *xmalloc(size_t size)
{
    void *p = malloc(size);
    if (p == NULL)
        exitf("clox: %s\n", strerror(errno));
    return p;
}

void *xrealloc(void *p, size_t size)
{
    p = realloc(p, size);
    if (p == NULL)
        exitf("clox: %s\n", strerror(errno));
    return p;
}
