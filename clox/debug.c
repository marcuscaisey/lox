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

int disassemble_instruction(const struct bytecode_chunk *chunk, int offset)
{
    printf("%04d ", offset);
    int line = bytecode_chunk_offset_line(chunk, offset);
    if (offset > 0 && line == bytecode_chunk_offset_line(chunk, offset - 1))
        printf("   | ");
    else
        printf("%4d ", line);
    enum opcode opcode = chunk->instructions[offset];
    switch (opcode) {
    case OPCODE_CONSTANT: {
        uint8_t index = chunk->instructions[offset + 1];
        int width = sizeof("OPCODE_CONSTANT_LONG") - 1;
        printf("%-*s %4d '", width, "OPCODE_CONSTANT", index);
        value_print(chunk->constants.values[index]);
        printf("'\n");
        return offset + 2;
    }
    case OPCODE_CONSTANT_LONG: {
        int index = (chunk->instructions[offset + 1] << 16) +
                    (chunk->instructions[offset + 2] << 8) + (chunk->instructions[offset + 3]);
        printf("%s %4d '", "OPCODE_CONSTANT_LONG", index);
        value_print(chunk->constants.values[index]);
        printf("'\n");
        return offset + 4;
    }
    case OPCODE_NEGATE: {
        printf("%s\n", "OPCODE_NEGATE");
        return offset + 1;
        break;
    }
    case OPCODE_RETURN:
        printf("%s\n", "OPCODE_RETURN");
        return offset + 1;
    default:
        printf("Unknown opcode %d\n", opcode);
        return offset + 1;
    }
}
