// The functions in this module manage memory owned by the garbage collector. All allocations made
// by the VM must go through this module so that they can be garbage collected.

#ifndef CLOX_MEMORY_H
#define CLOX_MEMORY_H

#include <stddef.h>

// Works like `malloc()`, except it exits the program on error.
void *xmalloc(size_t size);

// Works like `realloc()`, except it exits the program on error.
void *xrealloc(void *p, size_t size);

#endif
