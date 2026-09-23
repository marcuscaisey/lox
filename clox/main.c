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

    bytecode_chunk_write_constant(&chunk, 1.2, 1);
    bytecode_chunk_write(&chunk, OPCODE_NEGATE, 2);
    bytecode_chunk_write(&chunk, OPCODE_RETURN, 3);

    vm_interpret(&vm, &chunk);

    bytecode_chunk_free(&chunk);
    vm_free(&vm);

    return 0;
}
