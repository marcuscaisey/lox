#include "bytecode.h"

#include <assert.h>
#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>

#include "dynamic_array.h"
#include "value.h"

const char *instruction_name(enum opcode opcode)
{
    switch (opcode) {
    case OP_CONSTANT:
        return "CONSTANT";
    case OP_NIL:
        return "NIL";
    case OP_TRUE:
        return "TRUE";
    case OP_FALSE:
        return "FALSE";
    case OP_POP:
        return "POP";
    case OP_DEFINE_GLOBAL:
        return "DEFINE_GLOBAL";
    case OP_DEFINE_GLOBAL_FINAL:
        return "DEFINE_GLOBAL_FINAL";
    case OP_GET_GLOBAL:
        return "GET_GLOBAL";
    case OP_SET_GLOBAL:
        return "SET_GLOBAL";
    case OP_DEFINE_LOCAL:
        return "DEFINE_LOCAL";
    case OP_DEFINE_LOCAL_FINAL:
        return "DEFINE_LOCAL_FINAL";
    case OP_GET_LOCAL:
        return "GET_LOCAL";
    case OP_SET_LOCAL:
        return "SET_LOCAL";
    case OP_EQUAL:
        return "EQUAL";
    case OP_NOT_EQUAL:
        return "NOT_EQUAL";
    case OP_LESS:
        return "LESS";
    case OP_LESS_EQUAL:
        return "LESS_EQUAL";
    case OP_GREATER:
        return "GREATER";
    case OP_GREATER_EQUAL:
        return "GREATER_EQUAL";
    case OP_ADD:
        return "ADD";
    case OP_SUBTRACT:
        return "SUBTRACT";
    case OP_MULTIPLY:
        return "MULTIPLY";
    case OP_DIVIDE:
        return "DIVIDE";
    case OP_NOT:
        return "NOT";
    case OP_NEGATE:
        return "NEGATE";
    case OP_PRINT:
        return "PRINT";
    case OP_RETURN:
        return "RETURN";
    }
}

void bytecode_chunk_init(struct bytecode_chunk *chunk)
{
    chunk->len = 0;
    chunk->_cap = 0;
    chunk->instructions = NULL;
    chunk->offset_lines = NULL;
    value_array_init(&chunk->constants);
}

void bytecode_chunk_free(struct bytecode_chunk *chunk)
{
    free(chunk->instructions);
    value_array_free(&chunk->constants);
    bytecode_chunk_init(chunk);
}

// TODO: feels like we should just have a single function which takes in an opcode and operands, or
// multiple functions for writing instructions with each possible number of operands?

void bytecode_chunk_write(struct bytecode_chunk *chunk, uint8_t byte, int line)
{
    size_t original_cap = chunk->_cap;
    DYNAMIC_ARRAY_GROW(chunk->instructions, chunk->_cap, chunk->len + 1);
    // DYNAMIC_ARRAY_GROW might modify _cap so we need to pass in the original value to ensure that
    // we'll grow _offset_lines as well.
    DYNAMIC_ARRAY_GROW(chunk->offset_lines, original_cap, chunk->len + 1);
    int offset = chunk->len;
    chunk->instructions[offset] = byte;
    chunk->offset_lines[offset] = line;
    chunk->len++;
}

uint32_t bytecode_chunk_add_constant(struct bytecode_chunk *chunk, value value)
{
    uint32_t index = chunk->constants.len;
    assert(index <= (1 << 24) - 1); // The largest constant index operand type is u24
    value_array_write(&chunk->constants, value);
    return index;
}
