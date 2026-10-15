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
    case OP_JUMP:
        return "JUMP";
    case OP_JUMP_IF_FALSE:
        return "JUMP_IF_FALSE";
    case OP_RETURN:
        return "RETURN";
    }
}

int instruction_stack_depth_effect(enum opcode opcode)
{
    switch (opcode) {
    case OP_CONSTANT:
    case OP_NIL:
    case OP_TRUE:
    case OP_FALSE:
    case OP_GET_GLOBAL:
    case OP_GET_LOCAL:
        return 1;
    case OP_POP:
    case OP_DEFINE_GLOBAL:
    case OP_DEFINE_GLOBAL_FINAL:
    case OP_EQUAL:
    case OP_NOT_EQUAL:
    case OP_LESS:
    case OP_LESS_EQUAL:
    case OP_GREATER:
    case OP_GREATER_EQUAL:
    case OP_ADD:
    case OP_SUBTRACT:
    case OP_MULTIPLY:
    case OP_DIVIDE:
    case OP_PRINT:
        return -1;
    case OP_SET_GLOBAL:
    case OP_DEFINE_LOCAL:
    case OP_DEFINE_LOCAL_FINAL:
    case OP_SET_LOCAL:
    case OP_NOT:
    case OP_NEGATE:
    case OP_JUMP:
    case OP_JUMP_IF_FALSE:
    case OP_RETURN: // TODO
        return 0;
    }
}

void bytecode_chunk_init(struct bytecode_chunk *chunk)
{
    chunk->len = 0;
    chunk->_cap = 0;
    chunk->instructions = NULL;
    chunk->offset_lines = NULL;
    value_array_init(&chunk->constants);
    chunk->max_stack_depth = 0;
}

void bytecode_chunk_free(struct bytecode_chunk *chunk)
{
    free(chunk->instructions);
    value_array_free(&chunk->constants);
    bytecode_chunk_init(chunk);
}

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

constant_index bytecode_chunk_add_constant(struct bytecode_chunk *chunk, value value)
{
    constant_index index = chunk->constants.len;
    value_array_write(&chunk->constants, value);
    return index;
}
