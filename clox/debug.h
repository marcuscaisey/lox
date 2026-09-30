#ifndef CLOX_DEBUG_H
#define CLOX_DEBUG_H

#include "bytecode.h"

// Prints a human-readable representation of `chunk`.
// `name` is a meaningful name that will be printed as well to identify the chunk.
void disassemble(struct bytecode_chunk chunk, const char *name);

// Prints a human-readable representation of the instruction at offset `offset` in `chunk`.
// Returns the offset of the following instruction.
int disassemble_instruction(struct bytecode_chunk chunk, int offset);

#endif
