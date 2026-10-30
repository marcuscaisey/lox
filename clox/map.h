#ifndef CLOX_MAP_H
#define CLOX_MAP_H

#include <stdbool.h>
#include <stddef.h>

#include "value.h"
#include "vm_fwd.h"

// Map from `value` keys to `value` values.
// Must be initialised with `map_init()` before use and deinitialised with `map_deinit()` after use.
typedef struct {
    // Internal fields below, do not use.
    // Map is implemented as an open addressing hash table.
    // See https://en.wikipedia.org/wiki/Hash_table#Open_addressing
    struct _entry *_entries;
    size_t _count; // Number of values and tombstones in the map
    size_t _cap; // Number of elements that space has been allocated for in `_entries`
    // Internal field, do not use.
    // Whether the VM manages dynamic memory allocation for this map.
    bool _is_vm_managed;
} map;

// Initialises `map` for use.
// `is_vm_managed` determines whether the VM manages dynamic memory allocation for `map`. If a map
// is not VM managed, then `NULL` can be passed for the `vm *` parameter to all `map_*()` functions.
void map_init(map *map, bool is_vm_managed);

// Frees the memory associated with `map`.
void map_deinit(vm *vm, map *map);

// Associates `value` with `key` and reports whether this was a new key.
bool map_set(vm *vm, map *map, value key, value value);

// Looks up the value associated with `key` and sets `out` to it if it's found.
// Reports whether the value was found.
// If `out == NULL` then it's ignored.
bool map_get(map *map, value key, value *out);

// Returns the string key with the provided fields or `NULL` if no key is found.
struct object_string *map_get_string_key(map *map, const char *data, size_t len, hash_value hash);

// Deletes the value associated with `key` if it's found.
// Reports whether the value was found.
bool map_delete(map *map, value key);

#endif
