#ifndef CLOX_DYNAMIC_ARRAY_H
#define CLOX_DYNAMIC_ARRAY_H

#include <stdbool.h>
#include <stddef.h>

#include "memory.h" // IWYU pragma: keep // xrealloc used by DYNAMIC_ARRAY_GROW
#include "vm_memory.h" // IWYU pragma: keep // vm_realloc used by DYNAMIC_ARRAY_GROW

// Number of elements that space will be allocated for for the first time.
#define DYNAMIC_ARRAY_INITIAL_CAPACITY 8

// The factor that the array's capacity is grown by once its capacity is reached.
#define DYNAMIC_ARRAY_GROW_FACTOR 2

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
            if ((cap) < DYNAMIC_ARRAY_INITIAL_CAPACITY)      \
                (cap) = DYNAMIC_ARRAY_INITIAL_CAPACITY;      \
            else                                             \
                while ((cap) < (n))                          \
                    (cap) *= DYNAMIC_ARRAY_GROW_FACTOR;      \
            size_t size = DYNAMIC_ARRAY_SIZE((data), (cap)); \
            (data) = xrealloc((data), size);                 \
        }                                                    \
    } while (false)

// Works like `DYNAMIC_ARRAY_GROW()`, except the memory is allocated by `vm`.
#define VM_DYNAMIC_ARRAY_GROW(vm, data, cap, n)                           \
    do {                                                                  \
        if ((n) > (cap)) {                                                \
            size_t current_size = DYNAMIC_ARRAY_SIZE((data), (cap));      \
            (cap) = (cap) < 8 ? 8 : (cap) * 2;                            \
            while ((cap) < (n))                                           \
                (cap) *= 2;                                               \
            size_t target_size = DYNAMIC_ARRAY_SIZE((data), (cap));       \
            (data) = vm_realloc((vm), (data), current_size, target_size); \
        }                                                                 \
    } while (false)

// Returns the number of bytes occupied by the dynamically allocated array pointed to by `data` with
// space allocated for `cap` elements.
#define DYNAMIC_ARRAY_SIZE(data, cap) ((cap) * sizeof(*(data)))

#endif
