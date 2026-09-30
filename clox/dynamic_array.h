#ifndef CLOX_DYNAMIC_ARRAY_H
#define CLOX_DYNAMIC_ARRAY_H

#include <stdbool.h>

// Ensures that there is enough space in a dynamically allocated array so that it can hold at least
// `n` elements by reallocating more space if necessary. If the array already has enough space, then
// this is a no-op.
// `data` is a modifiable lvalue pointing to the start of the array. After this macro, this will
// point to the start of an array with the same elements as `data` and enough space for at least `n`
// elements.
// `size` is a modifiable lvalue which is the current number of elements that space has been
// allocated for. If more space is allocated, this is updated to the new number of elements that
// space has been allocated for.
#define DYNAMIC_ARRAY_GROW(data, size, n)                             \
    do {                                                              \
        if ((n) > (size)) {                                           \
            size_t current_size = DYNAMIC_ARRAY_SIZE((data), (size)); \
            (size) = (size) < 8 ? 8 : (size) * 2;                     \
            while ((size) < (n))                                      \
                (size) *= 2;                                          \
            size_t target_size = DYNAMIC_ARRAY_SIZE((data), (size));  \
            (data) = reallocate((data), current_size, target_size);   \
        }                                                             \
    } while (false)

// Returns the number of bytes occupied by the dynamically allocated array pointed to by `data` with
// space allocated for `size` elements.
#define DYNAMIC_ARRAY_SIZE(data, size) ((size) * sizeof(*(data)))

#endif
