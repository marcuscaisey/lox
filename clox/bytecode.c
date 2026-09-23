#include "bytecode.h"

#include <assert.h>
#include <limits.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#include "memory.h"
#include "value.h"

static void offset_lines_init(struct _bytecode_offset_lines *lines)
{
    lines->_data = NULL;
    lines->_length = 0;
    lines->_capacity = 0;
    lines->_max_offset = -1;
}

static void offset_lines_free(struct _bytecode_offset_lines *lines)
{
    deallocate(lines->_data, lines->_capacity * sizeof(lines->_data));
    offset_lines_init(lines);
}

static void offset_lines_insert(struct _bytecode_offset_lines *lines, int offset, int line)
{
    if (offset > lines->_max_offset)
        lines->_max_offset = offset;
    if (lines->_length > 0) {
        int last_offset = lines->_data[lines->_length - 2];
        int last_line = lines->_data[lines->_length - 1];
        // These are just the assumptions stated in the comment on _bytecode_offset_lines
        assert(offset > last_offset);
        assert(line >= last_line);
        if (line == last_line)
            return;
    }
    if (lines->_length + 2 > lines->_capacity) {
        size_t current_size = sizeof(lines->_data) * lines->_capacity;
        lines->_capacity = lines->_capacity < 8 ? 8 : lines->_capacity * 2;
        size_t target_size = sizeof(lines->_data) * lines->_capacity;
        lines->_data = reallocate(lines->_data, current_size, target_size);
    }
    lines->_data[lines->_length++] = offset;
    lines->_data[lines->_length++] = line;
}

// Returns the line corresponding with `offset` or -1 if not found.
int offset_lines_get(const struct _bytecode_offset_lines *lines, int offset)
{
    if (offset < 0 || offset > lines->_max_offset)
        return -1;
    // The line corresponding with offset i is the line from the last {offset, line} pair with
    // offset <= i. To find this, we do a binary search to find the first pair with offset > i and
    // then look at the preceding pair which will then have offset <= i.
    int low = 0; // Lower bound of search range
    int high = lines->_length; // Upper bound of search range
    while (low != high) {
        int midpoint = low + (high - low) / 2;
        midpoint -= midpoint % 2; // Ensure this stays even so that it points to an offset
        if (lines->_data[midpoint] > offset)
            // This might be the answer so keep in search range
            high = midpoint;
        else
            // Everything to the left including this is definitely not the answer so exclude from
            // the search range. Increment by 2 to skip the line part of the {offset, line}
            // pair.
            low = midpoint + 2;
    }
    // {..., offset_i, line_i, ..., offset_j, line, low, line_k, ...}
    int line = lines->_data[low - 1];
    return line;
}

void bytecode_chunk_init(struct bytecode_chunk *chunk)
{
    chunk->length = 0;
    chunk->_capacity = 0;
    chunk->instructions = NULL;
    value_array_init(&chunk->constants);
    offset_lines_init(&chunk->_offset_lines);
}

void bytecode_chunk_free(struct bytecode_chunk *chunk)
{
    deallocate(chunk->instructions, chunk->_capacity * sizeof(chunk->instructions));
    value_array_free(&chunk->constants);
    offset_lines_free(&chunk->_offset_lines);
    bytecode_chunk_init(chunk);
}

// TODO: feels like we should just have a single function which takes in an opcode and operands, or
// multiple functions for writing instructions with each possible number of operands?

void bytecode_chunk_write(struct bytecode_chunk *chunk, uint8_t byte, int line)
{
    if (chunk->length + 1 > chunk->_capacity) {
        size_t current_size = sizeof(chunk->instructions) * chunk->_capacity;
        chunk->_capacity = chunk->_capacity < 8 ? 8 : chunk->_capacity * 2;
        size_t target_size = sizeof(chunk->instructions) * chunk->_capacity;
        chunk->instructions = reallocate(chunk->instructions, current_size, target_size);
    }
    int offset = chunk->length;
    offset_lines_insert(&chunk->_offset_lines, offset, line);
    chunk->instructions[chunk->length++] = byte;
}

void bytecode_chunk_write_constant(struct bytecode_chunk *chunk, value value, int line)
{
    int constant_index = chunk->constants.length;
    value_array_write(&chunk->constants, value);
    int max_constant_index = (1 << (sizeof(*chunk->instructions) * 8)) - 1;
    if (constant_index <= max_constant_index) {
        bytecode_chunk_write(chunk, OPCODE_CONSTANT, line);
        bytecode_chunk_write(chunk, constant_index, line);
    } else {
        bytecode_chunk_write(chunk, OPCODE_CONSTANT_LONG, line);
        bytecode_chunk_write(chunk, (constant_index >> 16) & 0xff, line);
        bytecode_chunk_write(chunk, (constant_index >> 8) & 0xff, line);
        bytecode_chunk_write(chunk, (constant_index >> 0) & 0xff, line);
    }
}

int bytecode_chunk_offset_line(const struct bytecode_chunk *chunk, int offset)
{
    return offset_lines_get(&chunk->_offset_lines, offset);
}
