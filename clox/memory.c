#include "memory.h"

#include <errno.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

void *xmalloc(size_t size)
{
    void *p = malloc(size);
    if (p == NULL) {
        fprintf(stderr, "clox: %s\n", strerror(errno));
        exit(1);
    }
    return p;
}

void *xrealloc(void *p, size_t size)
{
    p = realloc(p, size);
    if (p == NULL) {
        fprintf(stderr, "clox: %s\n", strerror(errno));
        exit(1);
    }
    return p;
}
