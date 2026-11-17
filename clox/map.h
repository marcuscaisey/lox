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

// Expands to a loop over the keys of the map pointed to by the result of the expression `map_ptr`.
// The loop body is provided as the third argument to the macro. The key is referenced in the loop
// body using `key_var_name`.
//
// For example:
//     MAP_FOR_EACH_KEY(&vm->_strings, key, {
//         struct object *key_object = value_as_object(key);
//         if (!key_object->is_reachable)
//             map_delete(&vm->_strings, key);
//     });
#define MAP_FOR_EACH_KEY(map_ptr, key_var_name, ...)                                         \
    do {                                                                                     \
        map *_map_for_each_key_map_ptr = (map_ptr);                                          \
        for (size_t _map_for_each_key_i = 0;                                                 \
             _map_for_each_key_i < _map_for_each_key_map_ptr->_cap; _map_for_each_key_i++) { \
            struct _entry _map_for_each_key_entry =                                          \
                _map_for_each_key_map_ptr->_entries[_map_for_each_key_i];                    \
            if (!_map_slot_is_populated(_map_for_each_key_entry))                            \
                continue;                                                                    \
            value key_var_name = _map_for_each_key_entry.key;                                \
            __VA_ARGS__                                                                      \
        }                                                                                    \
    } while (false)

// Entry is what gets stored in the each slot of the array. The meanings of `key` and `value` depend
// on their values.
//   1. If `key != none_value`, then the slot is populated. This means that `value` is the value
//      associated with `key`.
//   2. If `key == none_value` and `value == none_value`, then the slot is empty. This means that
//      there's no value associated with `key`.
//   3. If `key == none_value` and `value == nil_value`, then the entry is a tombstone. This means
//      that the value previously associated with `key` was deleted.
//
// Internal type, do not use.
struct _entry {
    value key;
    value value;
};

// Reports whether the slot containing `entry` is populated.
// Internal function, do not use.
bool _map_slot_is_populated(struct _entry entry);

#endif
