#ifndef clox_bytecode_h
#define clox_bytecode_h

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

// Maps bytecode offsets to their corresponding source line numbers.
// Some assumptions are made so that the data can be stored with minimal space and time overhead:
//     1. Lines are inserted in offset order.
//     2. Lines increase with offsets. That is, for offsets i and j, if i < j, then line_i <= line_j.
// Internal type, do not use.
struct _bytecode_offset_lines {
    // Internal fields below, do not use.
    // NOTE: This really should have been an array of structs with offset offset and line members.
    // NOTE: This could also just have been a few fields on the `bytecode_chunk`.
    // Array containing pairs of integers of the form `{offset_1, line_1, offset_2, line_2, ...}`.
    // Pairs are stored in increasing offset order, where for each pair `{offset_i, line_i}`,
    // `offset_i` is the earliest offset that appears on `line_i`.
    //
    // To insert the line `n` for offset `i:
    // - If `data` is empty, then insert `{i, n}`.
    // - Otherwise, `data` looks like `{..., offset_n, line_n}`.
    //   Because of assumption 1 , `i` > `offset_n`. Therefore, because of assumption 2
    //   `n` >= `line_n`.
    //   If `n` == `line_n`, then nothing is inserted because there's already an earlier offset
    //   stored with the same line.
    //   If `n` > `line_n`, then insert `{i, n}`.
    //
    // The get the line for offset `i`, find the last pair `{offset, line}` with `offset` <= `i`.
    int *_data;
    int _len; // Number of elements in `data`
    int _cap; // Number of elements that space has been allocated for in `data`
    int _max_offset; // Maximum offset inserted
};

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
    struct _bytecode_offset_lines _offset_lines;
};

// Initialises `chunk` for use.
void bytecode_chunk_init(struct bytecode_chunk *chunk);

// Frees the memory associated with `chunk`.
void bytecode_chunk_free(struct bytecode_chunk *chunk);

// Writes `byte` into the chunk's instructions. `line` is the line number corresponding to the byte.
void bytecode_chunk_write(struct bytecode_chunk *chunk, uint8_t byte, int line);

// Stores `value` in the chunk's constants and writes the instruction to load it into the chunk's
// instructions. `line` is the line number corresponding to the value.
void bytecode_chunk_write_constant_instruction(struct bytecode_chunk *chunk, value value, int line);

// Returns the line corresponding with `offset` or -1 if not found.
int bytecode_chunk_offset_line(const struct bytecode_chunk *chunk, int offset);

#endif
