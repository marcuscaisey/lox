#include <stdio.h>

#include "bytecode.h"
#include "debug.h"

int main(int argc, char *argv[])
{
    (void)argc;
    (void)argv;
    struct bytecode_chunk chunk;
    bytecode_chunk_init(&chunk);
    bytecode_chunk_write(&chunk, OPCODE_RETURN, 1);
    bytecode_chunk_write_constant(&chunk, 1, 2);
    bytecode_chunk_write_constant(&chunk, 1, 3);
    bytecode_chunk_write_constant(&chunk, 2, 4);
    bytecode_chunk_write_constant(&chunk, 3, 5);
    bytecode_chunk_write_constant(&chunk, 5, 6);
    bytecode_chunk_write_constant(&chunk, 8, 7);
    disassemble(&chunk, "test chunk");
    bytecode_chunk_free(&chunk);
    return 0;
}
