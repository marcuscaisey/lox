#ifndef CLOX_DYNAMIC_ARRAY_H
#define CLOX_DYNAMIC_ARRAY_H

#include <stdbool.h>
#include <stddef.h>

#include "memory.h" // IWYU pragma: keep // xrealloc used by DYNAMIC_ARRAY_GROW

// Ensures that there is enough space in a dynamically allocated array so that it can hold at least
// `n` elements by reallocating more space if necessary. If the array already has enough space, then
// this is a no-op.
// `data` is a modifiable lvalue pointing to the start of the array. After this macro, this will
// point to the start of an array with the same elements as `data` and enough space for at least `n`
// elements.
// `cap` is a modifiable lvalue which is the current number of elements that space has been
// allocated for. If more space is allocated, this is updated to the new number of elements that
// space has been allocated for.
#define DYNAMIC_ARRAY_GROW(data, cap, n)                     \
    do {                                                     \
        if ((n) > (cap)) {                                   \
            (cap) = (cap) < 8 ? 8 : (cap) * 2;               \
            while ((cap) < (n))                              \
                (cap) *= 2;                                  \
            size_t size = DYNAMIC_ARRAY_SIZE((data), (cap)); \
            (data) = xrealloc((data), size);                 \
        }                                                    \
    } while (false)

// Returns the number of bytes occupied by the dynamically allocated array pointed to by `data` with
// space allocated for `cap` elements.
#define DYNAMIC_ARRAY_SIZE(data, cap) ((cap) * sizeof(*(data)))

#endif
