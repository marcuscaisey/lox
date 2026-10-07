#ifndef CLOX_BYTECODE_H
#define CLOX_BYTECODE_H

#include <stddef.h>
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
    OP_NIL, // Pushes nil
    OP_TRUE, // Pushes true
    OP_FALSE, // Pushes false
    OP_EQUAL, // Pops two numbers then pushes whether they're equal
    OP_NOT_EQUAL, // Pops two numbers then pushes whether they're not equal
    OP_LESS, // Pops two numbers b, then a, then pushes a < b
    OP_LESS_EQUAL, // Pops two numbers b, then a, then pushes a <= b
    OP_GREATER, // Pops two numbers b, then a, then pushes a > b
    OP_GREATER_EQUAL, // Pops two numbers b, then a, then pushes a >= b
    OP_ADD, // Pops two numbers or strings then pushes their sum
    OP_SUBTRACT, // Pops two numbers b, then a, then pushes a - b
    OP_MULTIPLY, // Pops two numbers then pushes their product
    OP_DIVIDE, // Pops two numbers b, then a, then pushes a / b
    OP_NOT, // Pops a value, then pushes whether it is falsey
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
    int *offset_lines; // Line numbers associated with each offset
    size_t len; // Number of elements in `instructions` and `offset_lines`
    // Internal field, do not use.
    // Number of elements that space has been allocated for in `instructions` and `offset_lines`.
    size_t _cap;
    struct value_array constants; // Pool of constants referenced by the bytecode
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

#endif
