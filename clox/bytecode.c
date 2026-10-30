#include "bytecode.h"

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>

#include "dynamic_array.h"
#include "line_number.h"
#include "value.h"
#include "vm_memory.h"

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
    case OP_POP_N:
        return "POP_N";
    case OP_DUP:
        return "DUP";
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
    case OP_MODULO:
        return "MODULO";
    case OP_NOT:
        return "NOT";
    case OP_NEGATE:
        return "NEGATE";
    case OP_PRINT:
        return "PRINT";
    case OP_JUMP:
        return "JUMP";
    case OP_JUMP_BACK:
        return "JUMP_BACK";
    case OP_JUMP_IF_TRUE:
        return "JUMP_IF_TRUE";
    case OP_JUMP_IF_FALSE:
        return "JUMP_IF_FALSE";
    case OP_CALL:
        return "CALL";
    case OP_RETURN:
        return "RETURN";
    }
}

void bytecode_chunk_init(struct bytecode_chunk *chunk, bool vm_managed)
{
    chunk->len = 0;
    chunk->_cap = 0;
    chunk->instructions = NULL;
    chunk->offset_lines = NULL;
    chunk->constants = NULL;
    chunk->constants_len = 0;
    chunk->_constants_cap = 0;
    chunk->max_stack_depth = 0;
    chunk->_is_vm_managed = vm_managed;
}

void bytecode_chunk_deinit(vm *vm, struct bytecode_chunk *chunk)
{
    if (chunk->_is_vm_managed) {
        vm_free(vm, chunk->instructions, DYNAMIC_ARRAY_SIZE(chunk->instructions, chunk->_cap));
        vm_free(vm, chunk->offset_lines, DYNAMIC_ARRAY_SIZE(chunk->offset_lines, chunk->_cap));
        vm_free(vm, chunk->constants, DYNAMIC_ARRAY_SIZE(chunk->constants, chunk->_constants_cap));
    } else {
        free(chunk->instructions);
        free(chunk->offset_lines);
        free(chunk->constants);
    }
    bytecode_chunk_init(chunk, chunk->_is_vm_managed);
}

void bytecode_chunk_write(vm *vm, struct bytecode_chunk *chunk, uint8_t byte, line_number line)
{
    size_t original_cap = chunk->_cap;
    size_t target_len = chunk->len + 1;
    if (chunk->_is_vm_managed) {
        VM_DYNAMIC_ARRAY_GROW(vm, chunk->instructions, chunk->_cap, target_len);
        if (chunk->_cap > original_cap) {
            chunk->offset_lines = vm_realloc(vm, chunk->offset_lines,
                                             DYNAMIC_ARRAY_SIZE(chunk->offset_lines, original_cap),
                                             DYNAMIC_ARRAY_SIZE(chunk->offset_lines, chunk->_cap));
        }
    } else {
        DYNAMIC_ARRAY_GROW(chunk->instructions, chunk->_cap, target_len);
        if (chunk->_cap > original_cap) {
            chunk->offset_lines =
                realloc(chunk->offset_lines, DYNAMIC_ARRAY_SIZE(chunk->offset_lines, chunk->_cap));
        }
    }
    int offset = chunk->len;
    chunk->instructions[offset] = byte;
    chunk->offset_lines[offset] = line;
    chunk->len++;
}

constant_index bytecode_chunk_add_constant(vm *vm, struct bytecode_chunk *chunk, value value)
{
    constant_index index = chunk->constants_len;
    size_t target_len = chunk->constants_len + 1;
    if (chunk->_is_vm_managed)
        VM_DYNAMIC_ARRAY_GROW(vm, chunk->constants, chunk->_constants_cap, target_len);
    else
        DYNAMIC_ARRAY_GROW(chunk->constants, chunk->_constants_cap, target_len);
    chunk->constants[chunk->constants_len++] = value;
    return index;
}
