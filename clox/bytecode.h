#ifndef CLOX_BYTECODE_H
#define CLOX_BYTECODE_H

#include <stdint.h>

#include "value.h"

// Instruction types supported by the Lox bytecode.
// Instructions operate on a stack-based virtual machine.
// The comment on each opcode describes the operation and operands, if any. Push and pop in these
// descriptions refer to pushing and popping from the virtual machine's value stack.
enum opcode {
    // Pushes a constant with index <= 255
    // Operands:
    //   u8 - index of the constant in `bytecode_chunk.constants`
    OP_CONSTANT,
    // Pushes a constant with index > 255
    // Operands:
    //   u24 - index of the constant in `bytecode_chunk.constants`, encoded in big-endian
    OP_CONSTANT_LONG,
    OP_ADD, // Pops two numbers then pushes their sum
    OP_SUBTRACT, // Pops two numbers b, then a, then pushes a - b
    OP_MULTIPLY, // Pops two numbers then pushes their product
    OP_DIVIDE, // Pops two numbers b, then a, then pushes a / b
    OP_NEGATE, // Pops a number, negates it, then pushes it back
    // TODO
    OP_RETURN,
};

// Returns the name of the instruction corresponding to `opcode`.
const char *instruction_name(enum opcode opcode);

// Represents a sequence of bytecode instructions.
// Must be initialised with `bytecode_chunk_init()` before use and freed with
// `bytecode_chunk_free()` after use.
struct bytecode_chunk {
    // Array of bytecode instructions of the form
    //     `{I1_OPCODE, I1_OPERAND_1, I2_OPCODE, I2_OPERAND_1, I2_OPERAND_2, ...}`
    // Must only be appended to via the `bytecode_write_*()` functions.
    uint8_t *instructions;
    int len; // Number of elements in `instructions`
    struct value_array constants; // Pool of constants referenced by the bytecode
    // Internal fields below, do not use.
    int _cap; // Number of elements that space has been allocated for in `instructions`
    int *_offset_lines; // Line numbers associated with each bytecode offset
};

// Initialises `chunk` for use.
void bytecode_chunk_init(struct bytecode_chunk *chunk);

// Frees the memory associated with `chunk`.
void bytecode_chunk_free(struct bytecode_chunk *chunk);

// Writes `byte` into the chunk's instructions. `line` is the line number corresponding to the byte.
void bytecode_chunk_write(struct bytecode_chunk *chunk, uint8_t byte, int line);

// Stores `value` in the chunk's constants and writes the instruction to load it into the chunk's
// instructions. `line` is the line number corresponding to the value.
void bytecode_chunk_write_constant(struct bytecode_chunk *chunk, value value, int line);

// Returns the line corresponding with `offset` or -1 if not found.
int bytecode_chunk_offset_line(struct bytecode_chunk chunk, int offset);

#endif
