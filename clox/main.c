#include <stdio.h>

#include "bytecode.h"
#include "vm.h"

int main(int argc, char *argv[])
{
    (void)argc;
    (void)argv;

    struct vm vm;
    vm_init(&vm);
    struct bytecode_chunk chunk;
    bytecode_chunk_init(&chunk);

    // 4 - (3 * (-2))
    bytecode_chunk_write_constant(&chunk, 4, 1);
    bytecode_chunk_write_constant(&chunk, 3, 1);
    bytecode_chunk_write_constant(&chunk, 2, 1);
    bytecode_chunk_write(&chunk, OPCODE_NEGATE, 1);
    bytecode_chunk_write(&chunk, OPCODE_MULTIPLY, 1);
    bytecode_chunk_write(&chunk, OPCODE_SUBTRACT, 1);

    // 4 - (3 * (0-2))
    bytecode_chunk_write_constant(&chunk, 4, 1);
    bytecode_chunk_write_constant(&chunk, 3, 1);
    bytecode_chunk_write_constant(&chunk, 0, 1);
    bytecode_chunk_write_constant(&chunk, 2, 1);
    bytecode_chunk_write(&chunk, OPCODE_SUBTRACT, 1);
    bytecode_chunk_write(&chunk, OPCODE_MULTIPLY, 1);
    bytecode_chunk_write(&chunk, OPCODE_SUBTRACT, 1);

    // 4 + (-(3 * (-2)))
    bytecode_chunk_write_constant(&chunk, 4, 1);
    bytecode_chunk_write_constant(&chunk, 3, 1);
    bytecode_chunk_write_constant(&chunk, 2, 1);
    bytecode_chunk_write(&chunk, OPCODE_NEGATE, 1);
    bytecode_chunk_write(&chunk, OPCODE_MULTIPLY, 1);
    bytecode_chunk_write(&chunk, OPCODE_NEGATE, 1);
    bytecode_chunk_write(&chunk, OPCODE_ADD, 1);

    bytecode_chunk_write(&chunk, OPCODE_RETURN, 1);

    vm_interpret(&vm, &chunk);

    bytecode_chunk_free(&chunk);
    vm_free(&vm);

    return 0;
}
