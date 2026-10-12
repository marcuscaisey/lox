#include "map.h"

#include <assert.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "dynamic_array.h"
#include "errors.h"
#include "object.h"
#include "value.h"
#include "vm_memory.h"

// Entry is what gets stored in the each slot of the array. The meanings of `key` and `value` depend
// on their values.
//   1. If `key != value_none`, then the slot is populated. This means that `value` is the value
//      associated with `key`.
//   2. If `key == value_none` and `value == value_none`, then the slot is empty. This means that
//      there's no value associated with `key`.
//   3. If `key == value_none` and `value == value_nil`, then the entry is a tombstone. This means
//      that the value previously associated with `key` was deleted.
//
// Use the helpers defined below for working with entries instead of directly working with the
// representation describe above.
struct _entry {
    value key;
    value value;
};

// Marks the slot containing `entry` as empty.
static void mark_empty_slot(struct _entry *entry)
{
    entry->key = value_none;
    entry->value = value_none;
}

// Reports whether the slot containing `entry` is populated.
static bool slot_is_populated(struct _entry entry)
{
    return !values_are_equal(entry.key, value_none);
}

// Reports whether the slot containing `entry` is empty.
static bool slot_is_empty(struct _entry entry)
{
    return values_are_equal(entry.key, value_none) && values_are_equal(entry.value, value_none);
}

// Marks `entry` as a tombstone.
static void mark_tombstone(struct _entry *entry)
{
    entry->key = value_none;
    entry->value = value_nil;
}

// Reports whether `entry` is a tombstone.
static bool is_tombstone(struct _entry entry)
{
    return values_are_equal(entry.key, value_none) && values_are_equal(entry.value, value_nil);
}

void map_init(map *map)
{
    map->_entries = NULL;
    map->_count = 0;
    map->_cap = 0;
}

void map_free(vm *vm, map *map)
{
    vm_mfree(vm, map->_entries, DYNAMIC_ARRAY_SIZE(map->_entries, map->_cap));
    map_init(map);
}

// Returns the entry corresponding with `key`. `entries` must be non-empty.
static struct _entry *find_entry(struct _entry *entries, size_t cap, value key)
{
    size_t index = value_hash(key) % cap;
    struct _entry *tombstone = NULL;
    for (size_t i = 0; i < cap; i++) {
        struct _entry *entry = &entries[index];
        if (!slot_is_populated(*entry)) {
            if (slot_is_empty(*entry))
                return tombstone != NULL ? tombstone : entry;
            if (is_tombstone(*entry) && tombstone == NULL)
                tombstone = entry;
        } else if (values_are_equal(entry->key, key)) {
            return entry;
        }
        index = (index + 1) % cap;
    }
    // We end up here if we've iterated over all of the entries and not found either the entry we're
    // looking for, an empty slot, or a tombstone. Since we grow the array when the load factor gets
    // above a certain threshold, there should always be at least 1 slot free and we should never
    // get here.
    panicf("clox: map: no empty slot found\n");
}

struct object_string *map_get_string_key(map *map, const char *data, size_t len, hash_value hash)
{
    // This is basically the same as find_entry. The two should stay in sync.
    if (map->_cap == 0)
        return NULL;
    size_t index = hash % map->_cap;
    for (size_t i = 0; i < map->_cap; i++) {
        struct _entry *entry = &map->_entries[index];
        if (!slot_is_populated(*entry)) {
            if (slot_is_empty(*entry))
                return NULL;
        } else if (value_type(entry->key) == VALUE_STRING) {
            struct object_string *string = value_as_string(entry->key);
            if (value_hash(entry->key) == hash && string->len == len &&
                memcmp(string->data, data, len) == 0) {
                return string;
            }
        }
        index = (index + 1) % map->_cap;
    }
    // We end up here if we've iterated over all of the entries and not found either the entry we're
    // looking for, an empty slot, or a tombstone. Since we grow the array when the load factor gets
    // above a certain threshold, there should always be at least 1 slot free and we should never
    // get here.
    panicf("clox: map: no empty slot found\n");
}

// Allocates a new array with capacity `new_cap` and transfers all of the entries from the old one
// to it.
void map_grow(vm *vm, map *map, size_t new_cap)
{
    struct _entry *new_entries = vm_malloc(vm, DYNAMIC_ARRAY_SIZE(new_entries, new_cap));
    for (struct _entry *entry = new_entries; entry < new_entries + new_cap; entry++) {
        mark_empty_slot(entry);
    }

    map->_count = 0; // Reset since current value includes tombstones which won't be copied over
    for (struct _entry *entry = map->_entries; entry < map->_entries + map->_cap; entry++) {
        if (!slot_is_populated(*entry))
            continue;
        struct _entry *new_entry = find_entry(new_entries, new_cap, entry->key);
        new_entry->key = entry->key;
        new_entry->value = entry->value;
        map->_count++;
    }

    vm_mfree(vm, map->_entries, DYNAMIC_ARRAY_SIZE(map->_entries, map->_cap));

    map->_entries = new_entries;
    map->_cap = new_cap;
}

// Number of entries that space will be allocated for for the first time.
#define INITIAL_CAPACITY 8

// Load factor is the ratio between the map's count and capacity. When the load factor gets above
// this number, the array needs to be grown.
// Split into numerator and denominator so that they can be used in static assertions.
#define MAX_LOAD_FACTOR_NUM 3
#define MAX_LOAD_FACTOR_DENOM 4
#define MAX_LOAD_FACTOR ((double)MAX_LOAD_FACTOR_NUM / MAX_LOAD_FACTOR_DENOM)

// The factor that the map's capacity is grown by once the max load factor is reached.
// Split into numerator and denominator so that they can be used in static assertions.
#define GROWTH_FACTOR_NUM 2
#define GROWTH_FACTOR_DENOM 1
#define GROWTH_FACTOR ((double)GROWTH_FACTOR_NUM / GROWTH_FACTOR_DENOM)

bool map_set(vm *vm, map *map, value key, value value)
{
    if ((map->_count + 1) > map->_cap * MAX_LOAD_FACTOR) {
        size_t new_cap = map->_cap > 0 ? map->_cap * GROWTH_FACTOR : INITIAL_CAPACITY;
        map_grow(vm, map, new_cap);
    }
    struct _entry *entry = find_entry(map->_entries, map->_cap, key);
    bool is_new_key = !slot_is_populated(*entry);
    if (is_new_key && slot_is_empty(*entry))
        map->_count++;
    entry->key = key;
    entry->value = value;
    return is_new_key;
}

bool map_delete(map *map, value key)
{
    if (map->_cap == 0)
        return false;
    struct _entry *entry = find_entry(map->_entries, map->_cap, key);
    bool found = slot_is_populated(*entry);
    if (found) {
        mark_tombstone(entry);
    }
    return found;
}

bool map_get(map *map, value key, value *out)
{
    if (map->_cap == 0)
        return false;
    struct _entry *entry = find_entry(map->_entries, map->_cap, key);
    bool found = slot_is_populated(*entry);
    if (found && out != NULL)
        *out = entry->value;
    return found;
}

// We need a few things to hold for operations on the map to function correctly:
//   1. There must be a slot available for every entry.
//   2. The growth factor needs to be large enough so that a new entry can be added after growing the
//      array.
//   3. After growing the array and adding a new entry, the new load factor must be less than the
//      maximum.
//
// We can derive assertions so that the above 3 statements will hold by considering the
// relationships between the following quantities:
//   N = count
//   C = capacity
//   L = max load factor
//   G = growth factor
//   I = initial capacity
//
// For 1:
//   If there's a slot available for every entry then we have N <= C <=> N/C <= 1
//   N/C is the load factor, so we need L <= 1
_Static_assert(MAX_LOAD_FACTOR_NUM <= MAX_LOAD_FACTOR_DENOM, "MAX_LOAD_FACTOR must be <= 1");
//
// For 2:
//   There's space for a new entry if N + 1 <= C
//   After growing the array, the capacity is now GC
//   So we need N + 1 <= GC <=> N/C + 1/C <= G
//   N <= C, so in the worst case we'll have 1 + 1/C <= G
//   1 + 1/C decreases as C increases, so it's enough that 1 + 1/I <= G <=> I + 1 <= GI
_Static_assert(INITIAL_CAPACITY *GROWTH_FACTOR_DENOM + GROWTH_FACTOR_DENOM <=
                   GROWTH_FACTOR_NUM * INITIAL_CAPACITY,
               "INITIAL_CAPACTIY + 1 must be <= GROWTH_FACTOR * INITIAL_CAPACITY");
//
// For 3:
//   After growing the array, the load factor is (N + 1)/CG
//   So we need (N + 1)/CG <= L
//   N <= C, so in the worst case we'll have (C + 1)/CG <= L <=> 1/G + 1/CG <= L
//   1/G + 1/CG decreases as C increases, so it's enough that 1/G + 1/IG <= L <=> I + 1 <= LGI
_Static_assert(INITIAL_CAPACITY * GROWTH_FACTOR_DENOM * MAX_LOAD_FACTOR_DENOM +
                       GROWTH_FACTOR_DENOM * MAX_LOAD_FACTOR_DENOM <=
                   MAX_LOAD_FACTOR_NUM * GROWTH_FACTOR_NUM * INITIAL_CAPACITY,
               "INITIAL_CAPACITY + 1 must be <= MAX_LOAD_FACTOR * GROWTH_FACTOR * INITIAL_CAPACITY");
