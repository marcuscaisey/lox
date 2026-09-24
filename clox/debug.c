#include "debug.h"

#include <stdint.h>
#include <stdio.h>
#include <string.h>

#include "bytecode.h"

void disassemble(const struct bytecode_chunk *chunk, const char *name)
{
    printf("== %s ==\n", name);
    for (int offset = 0; offset < chunk->length;)
        offset = disassemble_instruction(chunk, offset);
}

static const char *opcode_name(enum opcode opcode)
{
    switch (opcode) {
    case OPCODE_CONSTANT:
        return "CONSTANT";
    case OPCODE_CONSTANT_LONG:
        return "CONSTANT_LONG";
    case OPCODE_ADD:
        return "ADD";
    case OPCODE_SUBTRACT:
        return "SUBTRACT";
    case OPCODE_MULTIPLY:
        return "MULTIPLY";
    case OPCODE_DIVIDE:
        return "DIVIDE";
    case OPCODE_NEGATE:
        return "NEGATE";
    case OPCODE_RETURN:
        return "RETURN";
    }
}

int disassemble_instruction(const struct bytecode_chunk *chunk, int offset)
{
    printf("%04d ", offset);
    int line = bytecode_chunk_offset_line(chunk, offset);
    if (offset > 0 && line == bytecode_chunk_offset_line(chunk, offset - 1))
        printf("   | ");
    else
        printf("%4d ", line);
    enum opcode opcode = chunk->instructions[offset];
    const char *name = opcode_name(opcode);
    switch (opcode) {
    case OPCODE_CONSTANT: {
        uint8_t index = chunk->instructions[offset + 1];
        int width = strlen(opcode_name(OPCODE_CONSTANT_LONG));
        printf("%-*s %4d '", width, name, index);
        value_print(chunk->constants.values[index]);
        printf("'\n");
        return offset + 2;
    }
    case OPCODE_CONSTANT_LONG: {
        int index = (chunk->instructions[offset + 1] << 16) +
                    (chunk->instructions[offset + 2] << 8) + (chunk->instructions[offset + 3]);
        printf("%s %4d '", name, index);
        value_print(chunk->constants.values[index]);
        printf("'\n");
        return offset + 4;
    }
    case OPCODE_ADD:
    case OPCODE_SUBTRACT:
    case OPCODE_MULTIPLY:
    case OPCODE_DIVIDE:
    case OPCODE_NEGATE:
    case OPCODE_RETURN:
        printf("%s\n", name);
        return offset + 1;
    default:
        printf("Unknown opcode %d\n", opcode);
        return offset + 1;
    }
}
