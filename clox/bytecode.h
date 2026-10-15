#ifndef CLOX_BYTECODE_H
#define CLOX_BYTECODE_H

#include <stddef.h>
#include <stdint.h>

#include "value.h"

// Unsigned integer representing an index into a bytecode chunk's constants.
typedef uint32_t constant_index;

// The largest possible constant index.
#define MAX_CONSTANT_INDEX ((1 << 24) - 1)

// Unsigned integer representing an index into the VM's value stack.
typedef uint32_t stack_index;

// The largest possible stack index.
#define MAX_STACK_INDEX ((1 << 24) - 1)

// Unsigned integer representing the number of bytes for a jump instruction to jump forwards in the
// bytecode.
typedef uint16_t jump_offset;

#define MAX_JUMP_OFFSET UINT16_MAX

// Instruction types supported by the Lox bytecode.
// Instructions operate on a stack-based virtual machine.
//
// The comment on each opcode describes the operation and operands, if any. "Push" and "pop" refer
// to pushing and popping from the virtual machine's value stack. "Constants" refer to the bytecode
// chunk's constants.
//
// Operand types
//   constant_index
//     0 - 254: u8 representing the index value
//     255 - 16,777,215: u8 value of FF followed by u24 representing the index value
//   stack_index
//     0 - 254: u8 representing the index value
//     255 - 16,777,215: u8 value of FF followed by u24 representing the index value
//   jump_offset
//     u16
//
//   When an operand larger than a u8 is used, the bytes are encoded in big-endian order. I.e. `v`
//   encoded as a u16 is the byte `(v >> 8) & 0xFF` followed by the byte `v & 0xFF`.
enum opcode {
    // Pushes a constant
    // Operands:
    //   constant_index - index of the constant
    OP_CONSTANT,
    OP_NIL, // Pushes nil
    OP_TRUE, // Pushes true
    OP_FALSE, // Pushes false
    OP_POP, // Pops a value
    // Pops a value then defines a global variable initialised to that value
    // Operands:
    //   constant_index - index of the constant containing the variable name
    OP_DEFINE_GLOBAL,
    // Pops a value then defines a global final variable initialised to that value
    // Operands:
    //   constant_index - index of the constant containing the variable name
    OP_DEFINE_GLOBAL_FINAL,
    // Pushes the value of a global variable
    // Operands:
    //   constant_index - index of the constant containing the variable name
    OP_GET_GLOBAL,
    // Sets a global variable to the value on top of the stack
    // Operands:
    //   constant_index - index of the constant containing the variable name
    OP_SET_GLOBAL,
    // Defines a local variable initialised to the value on top of the stack
    OP_DEFINE_LOCAL,
    // Defines a local final variable initialised to the value on top of the stack
    // Operands:
    //   constant_index - index of the constant containing the variable name
    OP_DEFINE_LOCAL_FINAL,
    // Pushes a local variable from the stack
    // Operands:
    //   stack_index - index of the value in the stack
    OP_GET_LOCAL,
    // Sets a local variable from the stack to the value on top of the stack
    // Operands:
    //   stack_index - index of the value in the stack
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
    OP_NOT, // Pops a value, then pushes whether it is falsey
    OP_NEGATE, // Pops a number, negates it, then pushes it back
    OP_PRINT, // Pops a value then prints it
    // Jumps forward in the bytecode
    // Operands:
    //   jump_offset - number of bytes to jump starting at the end of the instruction including its
    //                 operands
    OP_JUMP,
    // Jumps forward in the bytecode if the value on top of the stack is falsey
    // Operands:
    //   jump_offset - number of bytes to jump starting at the end of the instruction including its
    //                 operands
    OP_JUMP_IF_FALSE,
    // TODO
    OP_RETURN,
};

// Returns the name of the instruction corresponding to `opcode`.
const char *instruction_name(enum opcode opcode);

// Returns the positive or negative amount that the instruction corresponding to `opcode` has on the
// depth of the VM's value stack. For example:
//   - If an instruction pushes a value onto the stack, then 1 is returned.
//   - If an instruction pops two values, then pushes another, then -1 is returned.
int instruction_stack_depth_effect(enum opcode opcode);

// Represents a sequence of bytecode instructions.
// Must be initialised with `bytecode_chunk_init()` before use and freed with
// `bytecode_chunk_free()` after use.
struct bytecode_chunk {
    // Array of bytecode instructions of the form
    //     `{I1_OPCODE, I1_OPERAND_1, I2_OPCODE, I2_OPERAND_1, I2_OPERAND_2, ...}`
    // Must only be appended to via the `bytecode_write_*()` functions.
    uint8_t *instructions;
    int *offset_lines; // Line numbers associated with each bytecode offset
    size_t len; // Number of elements in `instructions` and `offset_lines`
    // Internal field, do not use.
    // Number of elements that space has been allocated for in `instructions` and `offset_lines`.
    size_t _cap;
    struct value_array constants; // Pool of constants referenced by the bytecode
    // Maximum number of values that will be on the VM's value stack during execution
    size_t max_stack_depth;
};

// Initialises `chunk` for use.
void bytecode_chunk_init(struct bytecode_chunk *chunk);

// Frees the memory associated with `chunk`.
void bytecode_chunk_free(struct bytecode_chunk *chunk);

// Writes `byte` into the chunk's instructions. `line` is the line number corresponding to the byte.
void bytecode_chunk_write(struct bytecode_chunk *chunk, uint8_t byte, int line);

// Stores `value` in the chunk's constants and returns the index that can be used to retrieve it.
constant_index bytecode_chunk_add_constant(struct bytecode_chunk *chunk, value value);

#endif
