#include "debug.h"

#include <stdint.h>
#include <stdio.h>
#include <string.h>

#include "bytecode.h"
#include "value.h"

void disassemble(struct bytecode_chunk chunk, const char *name)
{
    printf("== %s ==\n", name);
    for (int offset = 0; offset < chunk.len;)
        offset = disassemble_instruction(chunk, offset);
}

int disassemble_instruction(struct bytecode_chunk chunk, int offset)
{
    printf("%04d ", offset);
    int line = bytecode_chunk_offset_line(chunk, offset);
    if (offset > 0 && line == bytecode_chunk_offset_line(chunk, offset - 1))
        printf("   | ");
    else
        printf("%4d ", line);
    enum opcode opcode = chunk.instructions[offset];
    const char *name = instruction_name(opcode);
    switch (opcode) {
    case OP_CONSTANT: {
        uint8_t index = chunk.instructions[offset + 1];
        int width = strlen(instruction_name(OP_CONSTANT_LONG));
        printf("%-*s %4d '", width, name, index);
        value_print(chunk.constants.values[index]);
        printf("'\n");
        return offset + 2;
    }
    case OP_CONSTANT_LONG: {
        int index = (chunk.instructions[offset + 1] << 16) + (chunk.instructions[offset + 2] << 8) +
                    (chunk.instructions[offset + 3]);
        printf("%s %4d '", name, index);
        value_print(chunk.constants.values[index]);
        printf("'\n");
        return offset + 4;
    }
    case OP_ADD:
    case OP_SUBTRACT:
    case OP_MULTIPLY:
    case OP_DIVIDE:
    case OP_NEGATE:
    case OP_RETURN:
        printf("%s\n", name);
        return offset + 1;
    }
    printf("Unknown opcode %d\n", opcode);
    return offset + 1;
}
