#ifndef CLOX_DEBUG_H
#define CLOX_DEBUG_H

#include <stdint.h>

struct bytecode_chunk;

// Prints a human-readable representation of `chunk`.
// `name` is a meaningful name that will be printed as well to identify the chunk.
void disassemble(const struct bytecode_chunk *chunk, const char *name);

// Prints a human-readable representation of the instruction pointed to by the instruction pointer
// `ip` in `chunk`.
// Returns a pointer to the following instruction.
uint8_t *disassemble_instruction(const struct bytecode_chunk *chunk, uint8_t *ip);

#endif
