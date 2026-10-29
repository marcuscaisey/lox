#include "debug.h"

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "bytecode.h"
#include "line_number.h"
#include "value.h"

void disassemble(const struct bytecode_chunk *chunk, const char *name)
{
    printf("== %s ==\n", name);
    for (bytecode_offset offset = 0; offset < chunk->len;)
        offset = disassemble_instruction(chunk, offset);
}

int disassemble_instruction(const struct bytecode_chunk *chunk, int offset)
{
    printf("%04d ", offset);
    line_number line = chunk->offset_lines[offset];
    if (offset > 0 && line == chunk->offset_lines[offset - 1])
        printf("   | ");
    else
        printf("%4d ", line);

    // Width of longest name of instruction which accepts an operand
    int name_width = strlen(instruction_name(OP_DEFINE_GLOBAL_FINAL));
    int operand_width = 4;

    enum opcode opcode = chunk->instructions[offset];
    const char *instruction = instruction_name(opcode);
    switch (opcode) {
    case OP_CONSTANT:
    case OP_DEFINE_GLOBAL:
    case OP_DEFINE_GLOBAL_FINAL:
    case OP_GET_GLOBAL:
    case OP_SET_GLOBAL:
    case OP_DEFINE_LOCAL_FINAL: {
        constant_index index = chunk->instructions[offset + 1];
        int index_bytes = 1;
        if (index == 0xFF) {
            index = (chunk->instructions[offset + 2] << 16) |
                    (chunk->instructions[offset + 3] << 8) | (chunk->instructions[offset + 4]);
            index_bytes = 4;
        }
        char *name = value_string(chunk->constants[index]);
        printf("%-*s %*d '%s'\n", name_width, instruction, operand_width, index, name);
        free(name);
        return offset + 1 + index_bytes;
    }
    case OP_POP_N: {
        pop_count count = chunk->instructions[offset + 1];
        int count_bytes = 1;
        if (count == 0xFF) {
            count = (chunk->instructions[offset + 2] << 16) |
                    (chunk->instructions[offset + 3] << 8) | (chunk->instructions[offset + 4]);
            count_bytes = 4;
        }
        printf("%-*s %*d\n", name_width, instruction, operand_width, count);
        return offset + 1 + count_bytes;
    }
    case OP_GET_LOCAL:
    case OP_SET_LOCAL: {
        stack_index index = chunk->instructions[offset + 1];
        int index_bytes = 1;
        if (index == 0xFF) {
            index = (chunk->instructions[offset + 2] << 16) |
                    (chunk->instructions[offset + 3] << 8) | (chunk->instructions[offset + 4]);
            index_bytes = 4;
        }
        printf("%-*s %*d\n", name_width, instruction, operand_width, index);
        return offset + 1 + index_bytes;
    }
    case OP_JUMP:
    case OP_JUMP_BACK:
    case OP_JUMP_IF_TRUE:
    case OP_JUMP_IF_FALSE: {
        jump_offset jump_offset = (chunk->instructions[offset + 1] << 8) |
                                  (chunk->instructions[offset + 2]);
        printf("%-*s %*d\n", name_width, instruction, operand_width, jump_offset);
        return offset + 3;
    }
    case OP_CALL: {
        argument_count count = chunk->instructions[offset + 1];
        int count_bytes = 1;
        if (count == 0xFF) {
            count = (chunk->instructions[offset + 2] << 16) |
                    (chunk->instructions[offset + 3] << 8) | (chunk->instructions[offset + 4]);
            count_bytes = 4;
        }
        printf("%-*s %*d\n", name_width, instruction, operand_width, count);
        return offset + 1 + count_bytes;
    }
    case OP_NIL:
    case OP_TRUE:
    case OP_FALSE:
    case OP_POP:
    case OP_DUP:
    case OP_DEFINE_LOCAL:
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
    case OP_MODULO:
    case OP_NOT:
    case OP_NEGATE:
    case OP_PRINT:
    case OP_RETURN:
        printf("%s\n", instruction);
        return offset + 1;
    }
    printf("Unknown opcode %d\n", opcode);
    return offset + 1;
}
