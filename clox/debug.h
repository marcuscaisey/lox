#ifndef clox_debug_h
#define clox_debug_h

#include "bytecode.h"

// Prints a human-readable representation of `chunk`.
// `name` is a meaningful name that will be printed as well to identify the chunk.
void disassemble(const struct bytecode_chunk *chunk, const char *name);

// Prints a human-readable representation of the instruction at offset `offset` in `chunk`.
// Returns the offset of the following instruction.
int disassemble_instruction(const struct bytecode_chunk *chunk, int offset);

#endif
