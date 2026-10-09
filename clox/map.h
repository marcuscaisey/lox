#ifndef CLOX_MAP_H
#define CLOX_MAP_H

#include <stddef.h>

#include "object.h"
#include "value.h"

// Map from `struct object_string *` keys to `value` values.
// Must be initialised with `map_init()` before use and freed with `map_free()` after use.
typedef struct {
    // Internal fields below, do not use.
    // Map is implemented as an open addressing hash table.
    // See https://en.wikipedia.org/wiki/Hash_table#Open_addressing
    struct _entry *_entries;
    size_t _count; // Number of values and tombstones in the map
    size_t _cap; // Number of elements that space has been allocated for in `_entries`
} map;

// Initialises `map` for use.
void map_init(map *map);

// Frees the memory associated with `map`.
void map_free(vm *vm, map *map);

// Associates `value` with `key` and reports whether this was a new key.
bool map_set(vm *vm, map *map, struct object_string *key, value value);

// Looks up the value associated with `key` and sets `out` to it if it's found.
// Reports whether the value was found.
bool map_get(map *map, struct object_string *key, value *out);

// Returns the key with data matching the provided parameters or `NULL` if no key is found.
struct object_string *map_get_key(map *map, const char *data, size_t len, uint32_t hash);

// Deletes the value associated with `key` if it's found.
// Reports whether the value was found.
bool map_delete(map *map, struct object_string *key);

#endif
