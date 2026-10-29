// This module defines the bytecode format and instruction set for the Lox virtual machine (VM).
//
// The Lox VM is an abstract computing machine (not tied to a particular implementation) that
// executes bytecode chunks (sequences of instructions). Each chunk has an associated pool of
// constants that instructions can reference by index.
//
// For each execution of a chunk, the VM maintains a "frame" containing the state of that execution
// and a stack of Lox values (the "value stack") that instructions can manipulate. Each frame has an
// independent value stack.
//
// The VM begins by creating a frame to execute an initial chunk. When a callable is called, a new
// frame is created to execute its associated chunk and execution of the calling chunk is suspended.
// When the callable returns, execution resumes in the calling chunk. Different calls of the same
// callable are associated with indpendent frames.

#ifndef CLOX_BYTECODE_H
#define CLOX_BYTECODE_H

#include <stddef.h>
#include <stdint.h>

#include "line_number.h"
#include "value.h"

typedef int32_t constant_index;

#define MAX_CONSTANT_INDEX ((1 << 24) - 1)

typedef int32_t pop_count;

#define MAX_POP_COUNT ((1 << 24) - 1)

typedef int32_t stack_index;

// Represents the depth of a frame's value stack.
typedef int32_t stack_depth;

// 1 less than `MAX_POP_COUNT` so that it's possible to pop every value from the stack with one
// POP_N instruction.
#define MAX_STACK_INDEX (MAX_POP_COUNT - 1)

typedef int32_t jump_offset;

#define MAX_JUMP_OFFSET ((1 << 16) - 1)

typedef int32_t argument_count;

// Instructions defined by the Lox VM's instruction set.
//
// The comment on each opcode describes the operation and operands, if any.
// Unless stated otherwise:
//   - "Push" and "pop" refer to pushing and popping from the current frame's value stack.
//   - "Bytecode chunk" refers to the currently executing bytecode chunk.
//   - "Constants" refer to the constant pool of the currently executing bytecode chunk.
//
// Operand types
//   constant_index
//     Index of a constant in a bytecode chunk's constant pool.
//     Encoding
//       0 - 254: u8 representing the index value
//       255 - 16,777,215: u8 value of FF followed by u24 representing the index value
//   pop_count
//     Number of values to pop from a value stack.
//     Encoding
//       0 - 254: u8 representing the number
//       255 - 16,777,215: u8 value of FF followed by u24 representing the number
//   stack_index
//     Index of a value in a value stack.
//     Encoding
//       0 - 254: u8 representing the index value
//       255 - 16,777,214: u8 value of FF followed by u24 representing the index value
//   jump_offset
//     Number of bytes for a jump instruction to jump in the bytecode chunk's instructions starting
//     from the next instruction's offset
//     Encoding
//       0 - 65,535: u16 representing the number
//   argument_count
//     Number of arguments passed to a callable.
//     Encoding
//       0 - 254: u8 representing the index value
//       255 - 16,777,215: u8 value of FF followed by u24 representing the index value
//
//   When an operand larger than a u8 is used, the bytes are encoded in big-endian order. I.e. `v`
//   encoded as a u16 is the byte `(v >> 8) & 0xFF` followed by the byte `v & 0xFF`.
enum opcode {
    // Pushes a constant
    // Operands
    //   constant_index - index of the constant
    OP_CONSTANT,
    OP_NIL, // Pushes nil
    OP_TRUE, // Pushes true
    OP_FALSE, // Pushes false
    OP_POP, // Pops a value
    // Pops multiple values
    // Operands
    //   pop_count - number of values to pop
    OP_POP_N,
    OP_DUP, // Pushes the value on top of the stack
    // Pops a value then defines a global variable initialised to that value
    // Operands
    //   constant_index - index of the constant containing the variable name
    OP_DEFINE_GLOBAL,
    // Pops a value then defines a global final variable initialised to that value
    // Operands
    //   constant_index - index of the constant containing the variable name
    OP_DEFINE_GLOBAL_FINAL,
    // Pushes the value of a global variable
    // Operands
    //   constant_index - index of the constant containing the variable name
    OP_GET_GLOBAL,
    // Sets a global variable to the value on top of the stack
    // Operands
    //   constant_index - index of the constant containing the variable name
    OP_SET_GLOBAL,
    // Marks the value on top of the stack as a local variable
    OP_DEFINE_LOCAL,
    // Marks the value on top of the stack as a final local variable
    // Operands
    //   constant_index - index of the constant containing the variable name
    OP_DEFINE_LOCAL_FINAL,
    // Pushes the value at a stack index
    // Operands
    //   stack_index - index of the value
    OP_GET_LOCAL,
    // Sets the value at a stack index to the value on top of the stack
    // Operands
    //   stack_index - index of the value
    OP_SET_LOCAL,
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
    OP_MODULO, // Pops two numbers b, then a, then pushes a % b
    OP_NOT, // Pops a value, then pushes whether it is falsy
    OP_NEGATE, // Pops a number, negates it, then pushes it back
    OP_PRINT, // Pops a value then prints it
    // Jumps forward in the bytecode chunk's instructions
    // Operands
    //   jump_offset - number of bytes to jump
    OP_JUMP,
    // Jumps backward in the bytecode chunk
    // Operands
    //   jump_offset - number of bytes to jump
    OP_JUMP_BACK,
    // Jumps forward in the bytecode chunk's instructions if the value on top of the stack is truthy
    // Operands
    //   jump_offset - number of bytes to jump
    OP_JUMP_IF_TRUE,
    // Jumps forward in the bytecode chunk's instructions if the value on top of the stack is falsy
    // Operands
    //   jump_offset - number of bytes to jump
    OP_JUMP_IF_FALSE,
    // Pops the top n values from the current frame's stack and the callable below them, then calls
    // the callable, passing those values as arguments. A new frame is created to execute the
    // callable's chunk and the passed arguments are made available at indices 0 through
    // `argument_count-1` of its stack. The function's arguments are marked as local variables (as
    // if with the `OP_DEFINE_LOCAL` instruction).
    // Operands
    //   argument_count - number of arguments being passed
    OP_CALL,
    // Pops the top value from the current frame's stack and ends execution of its chunk. If there
    // is a calling frame, the popped value is pushed onto its stack and the execution its is
    // resumed. Otherwise, VM execution terminates.
    OP_RETURN,
};

// Returns the name of the instruction corresponding to `opcode`.
const char *instruction_name(enum opcode opcode);

// Unsigned integer representing an absolute offset into a bytecode chunk's instructions.
typedef size_t bytecode_offset;

// Represents a sequence of bytecode instructions.
// Must be initialised with `bytecode_chunk_init()` before use and freed with
// `bytecode_chunk_deinit()` after use.
struct bytecode_chunk {
    // Array of instructions of the form
    //     `{I1_OPCODE, I1_OPERAND_1, I2_OPCODE, I2_OPERAND_1, I2_OPERAND_2, ...}`
    // Must only be appended to via `bytecode_write()`.
    uint8_t *instructions;
    int *offset_lines; // Line numbers associated with each bytecode offset
    size_t len; // Number of elements in `instructions` and `offset_lines`
    // Internal field, do not use.
    // Number of instructions and offset lines that space has been allocated for.
    size_t _cap;
    // Array of constants referenced by the chunk.
    // Must only be appended to via `bytecode_add_constant()`.
    value *constants;
    size_t constants_len; // Number of constants
    // Internal field, do not use.
    // Number of constants that space has been allocated for.
    size_t _constants_cap;
    // Maximum number of values that will be on the value stack during execution
    stack_depth max_stack_depth;
};

// Initialises `chunk` for use.
void bytecode_chunk_init(struct bytecode_chunk *chunk);

// Frees the memory associated with `chunk`.
void bytecode_chunk_deinit(struct bytecode_chunk *chunk);

// Writes `byte` into the chunk's instructions. `line` is the line number corresponding to the byte.
void bytecode_chunk_write(struct bytecode_chunk *chunk, uint8_t byte, line_number line);

// Stores `value` in the chunk's constants and returns its index.
constant_index bytecode_chunk_add_constant(struct bytecode_chunk *chunk, value value);

#endif
