#include "bytecode.h"

#include <assert.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>

#include "dynamic_array.h"
#include "line_number.h"
#include "value.h"
#include "vm_internal.h"

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
    if (chunk->_is_vm_managed)
        VM_DYNAMIC_ARRAY_GROW(vm, chunk->instructions, chunk->_cap, target_len);
    else
        DYNAMIC_ARRAY_GROW(chunk->instructions, chunk->_cap, target_len);
    if (chunk->_cap > original_cap) {
        size_t target_size = DYNAMIC_ARRAY_SIZE(chunk->offset_lines, chunk->_cap);
        if (chunk->_is_vm_managed) {
            size_t current_size = DYNAMIC_ARRAY_SIZE(chunk->offset_lines, original_cap);
            chunk->offset_lines = vm_realloc(vm, chunk->offset_lines, current_size, target_size);
        } else {
            chunk->offset_lines = realloc(chunk->offset_lines, target_size);
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

// Writes `value` as a big-endian u16 to `out`.
static void write_u16(uint8_t *out, uint16_t value)
{
    static const int32_t max_u16 = (1 << 16) - 1;
    assert(value >= 0 && value <= max_u16);
    out[0] = (value >> 8) & 0xFF;
    out[1] = value & 0xFF;
}

// Writes `value` associated with `line` to the chunk's instructions as a big-endian u16.
static void bytecode_chunk_write_u16(vm *vm, struct bytecode_chunk *chunk, int32_t value,
                                     line_number line)
{
    uint8_t bytes[2];
    write_u16(bytes, value);
    bytecode_chunk_write(vm, chunk, bytes[0], line);
    bytecode_chunk_write(vm, chunk, bytes[1], line);
}

static const int32_t max_u24 = (1 << 24) - 1;

// Writes `value` associated with line to the chunk's instructions as a big-endian u24.
static void bytecode_chunk_write_u24(vm *vm, struct bytecode_chunk *chunk, int32_t value,
                                     line_number line)
{
    assert(value >= 0 && value <= max_u24);
    bytecode_chunk_write(vm, chunk, (value >> 16) & 0xFF, line);
    bytecode_chunk_write(vm, chunk, (value >> 8) & 0xFF, line);
    bytecode_chunk_write(vm, chunk, value & 0xFF, line);
}

// Writes `value` associated with `line` to the chunk's instructions as a u8 if it's < 0xFF,
// otherwise as 0xFF followed by `value` as a big-endian u24.
static void bytecode_chunk_write_compact_u24(vm *vm, struct bytecode_chunk *chunk, int32_t value,
                                             line_number line)
{
    if (value < 0xFF) {
        assert(value >= 0);
        bytecode_chunk_write(vm, chunk, value, line);
    } else {
        bytecode_chunk_write(vm, chunk, 0xFF, line);
        bytecode_chunk_write_u24(vm, chunk, value, line);
    }
}

void bytecode_chunk_write_opcode(vm *vm, struct bytecode_chunk *chunk, enum opcode opcode,
                                 line_number line)
{
    bytecode_chunk_write(vm, chunk, opcode, line);
}

void bytecode_chunk_write_constant_index(vm *vm, struct bytecode_chunk *chunk, constant_index index,
                                         line_number line)
{
    bytecode_chunk_write_compact_u24(vm, chunk, index, line);
}

static const pop_count max_pop_count = max_u24;

void bytecode_write_pop_count(vm *vm, struct bytecode_chunk *chunk, pop_count count,
                              line_number line)
{
    bytecode_chunk_write_compact_u24(vm, chunk, count, line);
}

void bytecode_chunk_write_global_index(vm *vm, struct bytecode_chunk *chunk, global_index index,
                                       line_number line)
{
    assert(index >= 0 && index <= 0xFFFFFF);
    bytecode_chunk_write_compact_u24(vm, chunk, index, line);
}

// 1 less than `max_pop_count` so that it's possible to pop every value from the stack with one
// POP_N instruction.
static const stack_index max_stack_index = max_pop_count - 1;

void bytecode_chunk_write_stack_index(vm *vm, struct bytecode_chunk *chunk, stack_index index,
                                      line_number line)
{
    assert(index <= max_stack_index);
    bytecode_chunk_write_compact_u24(vm, chunk, index, line);
}

const size_t jump_offset_size = 2;

void bytecode_chunk_write_jump_offset(vm *vm, struct bytecode_chunk *chunk, jump_offset offset,
                                      line_number line)
{
    // Must update `jump_offset_size` if this is updated to write a differently sized value.
    bytecode_chunk_write_u16(vm, chunk, offset, line);
}

void bytecode_chunk_set_jump_offset(struct bytecode_chunk *chunk, bytecode_offset target,
                                    jump_offset offset)
{
    write_u16(chunk->instructions + target + 1, offset);
}

void bytecode_chunk_write_argument_count(vm *vm, struct bytecode_chunk *chunk, argument_count count,
                                         line_number line)
{
    bytecode_chunk_write_compact_u24(vm, chunk, count, line);
}

// Reads the u8 from the sequence of bytes pointed to by `*p` and moves `*p` past it.
static uint8_t bytecode_read_u8(uint8_t **ip)
{
    return *(*ip)++;
}

// Reads the big-endian u16 from the sequence of bytes pointed to by `*p` and moves `*p` past it.
static uint16_t bytecode_read_u16(uint8_t **ip)
{
    *ip += 2;
    return ((*ip)[-2] << 8) | ((*ip)[-1]);
}

// Reads the big-endian u24 from the sequence of bytes pointed to by `*p` and moves `*p` past it.
static uint32_t bytecode_read_u24(uint8_t **ip)
{
    *ip += 3;
    return ((*ip)[-3] << 16) | ((*ip)[-2] << 8) | ((*ip)[-1]);
}

// Reads the value encoded by `bytecode_chunk_write_compact_u24()` from the sequence of bytes
// pointed to by `*p` and moves `*p` past it.
static uint32_t bytecode_read_compact_u24(uint8_t **ip)
{
    uint8_t index = bytecode_read_u8(ip);
    if (index < 0xFF)
        return index;
    return bytecode_read_u24(ip);
}

enum opcode bytecode_read_opcode(uint8_t **ip)
{
    return bytecode_read_u8(ip);
}

constant_index bytecode_read_constant_index(uint8_t **ip)
{
    return bytecode_read_compact_u24(ip);
}

pop_count bytecode_read_pop_count(uint8_t **ip)
{
    return bytecode_read_compact_u24(ip);
}

global_index bytecode_read_global_index(uint8_t **ip)
{
    return bytecode_read_compact_u24(ip);
}

stack_index bytecode_read_stack_index(uint8_t **ip)
{
    return bytecode_read_compact_u24(ip);
}

jump_offset bytecode_read_jump_offset(uint8_t **ip)
{
    return bytecode_read_u16(ip);
}

argument_count bytecode_read_argument_count(uint8_t **ip)
{
    return bytecode_read_compact_u24(ip);
}
