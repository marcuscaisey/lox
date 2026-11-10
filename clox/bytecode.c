#include "bytecode.h"

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#include "dynamic_array.h"
#include "line_number.h"
#include "memory.h"
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
    case OP_GET_UPVALUE:
        return "GET_UPVALUE";
    case OP_SET_UPVALUE:
        return "SET_UPVALUE";
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
    case OP_CLOSURE:
        return "CLOSURE";
    case OP_CLOSE_UPVALUE:
        return "CLOSE_UPVALUE";
    case OP_CLOSE_UPVALUE_N:
        return "CLOSE_UPVALUE_N";
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
    chunk->final_local_names = NULL;
    chunk->final_local_names_len = 0;
    chunk->_final_local_names_cap = 0;
    chunk->max_stack_depth = 0;
    chunk->_is_vm_managed = vm_managed;
}

void bytecode_chunk_deinit(vm *vm, struct bytecode_chunk *chunk)
{
    for (size_t i = 0; i < chunk->final_local_names_len; i++) {
        char *name = chunk->final_local_names[i];
        if (chunk->_is_vm_managed)
            vm_free(vm, name, strlen(name) + 1);
        else
            free(name);
    }
    if (chunk->_is_vm_managed) {
        vm_free(vm, chunk->final_local_names,
                DYNAMIC_ARRAY_SIZE(chunk->final_local_names, chunk->_final_local_names_cap));
        vm_free(vm, chunk->instructions, DYNAMIC_ARRAY_SIZE(chunk->instructions, chunk->_cap));
        vm_free(vm, chunk->offset_lines, DYNAMIC_ARRAY_SIZE(chunk->offset_lines, chunk->_cap));
        vm_free(vm, chunk->constants, DYNAMIC_ARRAY_SIZE(chunk->constants, chunk->_constants_cap));
    } else {
        free(chunk->final_local_names);
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

final_local_name_index bytecode_chunk_add_final_local_name(vm *vm, struct bytecode_chunk *chunk,
                                                          const char *name, size_t len)
{
    final_local_name_index index = chunk->final_local_names_len;
    size_t target_len = chunk->final_local_names_len + 1;
    char *name_copy;
    if (chunk->_is_vm_managed) {
        VM_DYNAMIC_ARRAY_GROW(vm, chunk->final_local_names, chunk->_final_local_names_cap, target_len);
        name_copy = vm_malloc(vm, len + 1);
    } else {
        DYNAMIC_ARRAY_GROW(chunk->final_local_names, chunk->_final_local_names_cap, target_len);
        name_copy = xmalloc(len + 1);
    }
    memcpy(name_copy, name, len);
    name_copy[len] = '\0';
    chunk->final_local_names[chunk->final_local_names_len++] = name_copy;
    return index;
}
