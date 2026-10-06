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

void *allocate(size_t size)
{
    return xmalloc(size);
}

void *reallocate(void *p, size_t current_size, size_t target_size)
{
    // NOTE: We could keep track of allocation sizes if passing in the current size becomes awkward
    // for the caller.
    (void)current_size;
    return xrealloc(p, target_size);
}

void deallocate(void *p, size_t size)
{
    (void)size;
    free(p);
}
