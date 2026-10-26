#ifndef CLOX_MEMORY_H
#define CLOX_MEMORY_H

#include <stddef.h>

// Works like `malloc()`, except it exits the program on error.
void *xmalloc(size_t size);

// Works like `realloc()`, except it exits the program on error.
void *xrealloc(void *p, size_t size);

#endif
