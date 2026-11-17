// This module defines the bytecode format and instruction set for the Lox virtual machine (VM).
//
// The VM executes bytecode chunks (sequences of instructions). Each chunk has a pool of constants
// and a pool of final local variable names that instructions can reference by index.
//
// Execution Chunks
// ----------------
// For each execution of a chunk, the VM maintains a "frame" containing the state of that execution
// and a stack of Lox values (the "value stack") that instructions can manipulate and reference by
// index. Each frame has an independent value stack.
//
// The VM begins by creating a frame to execute an initial chunk. When a function is called, a new
// frame is created to execute its chunk and execution of the calling chunk is suspended. When the
// function returns, its frame is destroyed and execution resumes in the calling chunk. Different
// calls of the same function are associated with independent frames.
//
// Global Variables
// ----------------
// The VM maintains a set of global variables that are shared between executions of different
// chunks. Each global variable is identified by an index that remains stable for the lifetime of
// the VM.
//
// Closures
// --------
// A called function's chunk can normally only reference global variables and values in its frame's
// value stack. A closure can be created that associates a function with a set of "upvalues". These
// are values that aren't global and don't live on the value stack of the function's frame. When a
// closure is called, its function's chunk is executed in the same way as a normal function's is,
// but the chunk can also reference the closure's upvalues by index.
//
// An upvalue can either be "open" or "closed". An open upvalue aliases a value in a frame's value
// stack. A closed upvalue is stored by the VM separately from the frame's value stack. An upvalue
// is open when it's created and must be closed before the value it aliases is popped from the
// stack. A function's chunk references an upvalue by an index defined by its closure. Different
// closures can associate functions with the same upvalue which is shared between them.

#ifndef CLOX_BYTECODE_H
#define CLOX_BYTECODE_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#include "line_number.h"
#include "value.h"
#include "vm_fwd.h"

static const int32_t _bytecode_max_u24 = (1 << 24) - 1;

typedef int32_t constant_index;
static const constant_index bytecode_max_constant_index = _bytecode_max_u24;

typedef int32_t final_local_name_index;
static const final_local_name_index bytecode_max_final_local_name_index = _bytecode_max_u24;

typedef int32_t pop_count;
static const pop_count bytecode_max_pop_count = _bytecode_max_u24;

typedef int32_t global_index;
static const global_index bytecode_max_global_index = _bytecode_max_u24;

typedef int32_t stack_index;
// 1 less than `bytecode_max_pop_count` so that it's possible to pop every value from the stack
// with one POP_N instruction.
static const stack_index bytecode_max_stack_index = bytecode_max_pop_count - 1;

typedef int32_t jump_offset;
static const jump_offset bytecode_max_jump_offset = (1 << 16) - 1;

// Size in bytes of a jump offset value.
static const size_t bytecode_jump_offset_size = 2;

typedef int32_t argument_count;
static const argument_count bytecode_max_argument_count = _bytecode_max_u24;

typedef int32_t upvalue_count;
static const upvalue_count bytecode_max_upvalue_count = _bytecode_max_u24;

typedef int32_t upvalue_index;
// 1 less than `bytecode_max_upvalue_count` so that it's possible to reference every upvalue by index
static const upvalue_index bytecode_max_upvalue_index = bytecode_max_upvalue_count - 1;

// Instructions defined by the Lox VM's instruction set.
//
// The comment on each opcode describes the operation and operands, if any.
// Unless stated otherwise:
//   - "Push" and "pop" refer to pushing and popping from the current frame's value stack.
//   - "Bytecode chunk" refers to the currently executing bytecode chunk.
//   - "Upvalue" refers to an upvalue of the closure associated with the current frame.
//   - "Constants" refer to the constant pool of the currently executing bytecode chunk.
//   - "Final local variable names" refer to the final local variable name pool of the currently
//     executing bytecode chunk.
//
// Operand types
//   constant_index
//     Index of a constant in a bytecode chunk's constant pool.
//     Range: 0 - 16,777,215
//   final_local_name_index
//     Index of a name in a bytecode chunk's final local variable name pool.
//     Range: 0 - 16,777,215
//   pop_count
//     Number of values to pop from a value stack.
//     Range: 0 - 16,777,215
//   global_index
//     Index of a global variable maintained by the VM.
//     Range: 0 - 16,777,215
//   upvalue_index
//     Index of a closure's upvalue.
//     Range: 0 - 16,777,214
//   stack_index
//     Index of a value in a value stack.
//     Range: 0 - 16,777,214
//   jump_offset
//     Number of bytes for a jump instruction to jump in the bytecode chunk's instructions starting
//     from the next instruction's offset
//     Range: 0 - 65,535
//   argument_count
//     Number of arguments passed to a function.
//     Range: 0 - 16,777,215
//   upvalue_count
//     Number of upvalues that a closure associates with a function.
//     Range: 0 - 16,777,215
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
    //   global_index - index of the global variable
    OP_DEFINE_GLOBAL,
    // Pops a value then defines a global final variable initialised to that value
    // Operands
    //   global_index - index of the global variable
    OP_DEFINE_GLOBAL_FINAL,
    // Pushes the value of a global variable
    // Operands
    //   global_index - index of the global variable
    OP_GET_GLOBAL,
    // Sets a global variable to the value on top of the stack
    // Operands
    //   global_index - index of the global variable
    OP_SET_GLOBAL,
    // Pushes the value of an upvalue
    // Operands
    //   upvalue_index - index of the upvalue
    OP_GET_UPVALUE,
    // Sets the value of an upvalue
    // Operands
    //   upvalue_index - index of the upvalue
    OP_SET_UPVALUE,
    // Marks the value on top of the stack as a local variable
    OP_DEFINE_LOCAL,
    // Marks the value on top of the stack as a final local variable
    // Operands
    //   final_local_name_index - index of the final local variable name
    OP_DEFINE_LOCAL_FINAL,
    // Pushes the value from the value stack
    // Operands
    //   stack_index - index of the value
    OP_GET_LOCAL,
    // Sets the value in the value stack to the value on top of the stack. The value assigned to
    // must have been marked local or final local variable.
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
    // Pops the top n values from the current frame's stack and the function below them, then calls
    // the function, passing those values as arguments.
    //
    // A new frame is created to execute the function's chunk and the passed arguments are made
    // available at indices 0 through `n-1` of its stack. The function's arguments are marked as
    // local variables (as if with the `OP_DEFINE_LOCAL` instruction).
    //
    // Operands
    //   argument_count - number of arguments being passed
    OP_CALL,
    // Creates a closure associating a function with a set of upvalues and pushes it.
    //
    // The operands after the upvalue count specify the sources of the upvalues. They can refer to
    // either:
    //   a. A value in the current frame's value stack. In this case, an open upvalue aliasing the
    //      value is created if one doesn't exist and this upvalue is associated with the function.
    //   b. An upvalue of the closure associated with the current frame. In this case, that upvalue
    //      is associated with the function.
    // The source operands are grouped into pairs where the value of the first operand determines
    // what the index in the second one refers to. If the first is 1, then the second refers to (a),
    // otherwise it refers to (b). The order of the pairs determines the indexes that the function's
    // chunk can reference its associated upvalues with.
    //
    // Operands
    //   constant_index - index of the function
    //   upvalue_count - number of upvalues
    //   u8 - 1 or 0
    //   stack_index or upvalue_index - index for upvalue 0
    //   u8 - 1 or 0
    //   stack_index or upvalue_index - index for upvalue 1
    //   ...
    //   u8 - 1 or 0
    //   stack_index or upvalue_index - index for upvalue n-1
    OP_CLOSURE,
    // Closes the open upvalue aliasing the value on top of the stack
    OP_CLOSE_UPVALUE,
    // Closes the open upvalues that alias any of the top n values of the stack (if any)
    // Operands
    //   pop_count - number of open upvalues to close
    OP_CLOSE_UPVALUE_N,
    // Pops the top value from the current frame's stack, closes the open upvalues that alias any of
    // the values remaining on the stack, and ends execution of its chunk. If there is a calling
    // frame, the popped value is pushed onto its stack and its execution is resumed. Otherwise, VM
    // execution terminates.
    OP_RETURN,
};

// Returns the name of the instruction corresponding to `opcode`.
const char *instruction_name(enum opcode opcode);

// Unsigned integer representing an absolute offset into a bytecode chunk's instructions.
typedef size_t bytecode_offset;

// Represents a sequence of bytecode instructions.
// Must be initialised with `bytecode_chunk_init()` before use and deinitialised with
// `bytecode_chunk_deinit()` after use.
struct bytecode_chunk {
    // Array of instructions of the form
    //     `{I1_OPCODE, I1_OPERAND_1, I2_OPCODE, I2_OPERAND_1, I2_OPERAND_2, ...}`
    // Must only be appended to via `bytecode_chunk_write()`.
    uint8_t *instructions;
    int *offset_lines; // Line numbers associated with each bytecode offset
    size_t len; // Number of elements in `instructions` and `offset_lines`
    // Internal field, do not use.
    // Number of instructions and offset lines that space has been allocated for.
    size_t _cap;
    // Array of constants referenced by the chunk.
    // Must only be appended to via `bytecode_chunk_add_constant()`.
    value *constants;
    size_t constants_len; // Number of constants
    // Internal field, do not use.
    // Number of constants that space has been allocated for.
    size_t _constants_cap;
    // Array of final local variable names referenced by the chunk.
    // Must only be appended to via `bytecode_chunk_add_final_local_name()`.
    char **final_local_names;
    size_t final_local_names_len; // Number of final local variable names
    // Internal field, do not use.
    // Number of final local variable names that space has been allocated for.
    size_t _final_local_names_cap;
    // Maximum number of values that will be on the value stack during execution
    size_t max_stack_depth;
    // Internal field, do not use.
    // Whether the VM manages dynamic memory allocation for this chunk.
    bool _is_vm_managed;
};

// Initialises `chunk` for use.
// `is_vm_managed` determines whether the VM manages dynamic memory allocation for `chunk`. If a
// chunk is not VM managed, then `NULL` can be passed for the `vm *` parameter to all
// `bytecode_chunk_*()` functions.
void bytecode_chunk_init(struct bytecode_chunk *chunk, bool is_vm_managed);

// Frees the memory associated with `chunk`.
void bytecode_chunk_deinit(vm *vm, struct bytecode_chunk *chunk);

// Writes `byte` associated with `line` to the chunk's instructions.
void bytecode_chunk_write(vm *vm, struct bytecode_chunk *chunk, uint8_t byte, line_number line);

// Stores `value` in the chunk's constants and returns its index.
size_t bytecode_chunk_add_constant(vm *vm, struct bytecode_chunk *chunk, value value);

// Stores a copy of the first `len` bytes of `name` in the chunk's final local variable name pool
// and returns its index.
size_t bytecode_chunk_add_final_local_name(vm *vm, struct bytecode_chunk *chunk, const char *name,
                                           size_t len);

// Writes `value` as a big-endian u16 to `out`.
// Internal function, do not use.
static inline void _bytecode_write_u16(uint8_t *out, int32_t value)
{
    out[0] = (value >> 8) & 0xFF;
    out[1] = value & 0xFF;
}

// Writes `value` associated with `line` to the chunk's instructions as a big-endian u16.
// Internal function, do not use.
static inline void _bytecode_chunk_write_u16(vm *vm, struct bytecode_chunk *chunk, int32_t value,
                                             line_number line)
{
    uint8_t bytes[2];
    _bytecode_write_u16(bytes, value);
    bytecode_chunk_write(vm, chunk, bytes[0], line);
    bytecode_chunk_write(vm, chunk, bytes[1], line);
}

// Writes `value` associated with line to the chunk's instructions as a big-endian u24.
// Internal function, do not use.
static inline void _bytecode_chunk_write_u24(vm *vm, struct bytecode_chunk *chunk, int32_t value,
                                             line_number line)
{
    bytecode_chunk_write(vm, chunk, (value >> 16) & 0xFF, line);
    bytecode_chunk_write(vm, chunk, (value >> 8) & 0xFF, line);
    bytecode_chunk_write(vm, chunk, value & 0xFF, line);
}

// Writes `value` associated with `line` to the chunk's instructions as a u8 if it's < 0xFF,
// otherwise as 0xFF followed by `value` as a big-endian u24.
// Internal function, do not use.
static inline void _bytecode_chunk_write_compact_u24(vm *vm, struct bytecode_chunk *chunk,
                                                     int32_t value, line_number line)
{
    if (value < 0xFF) {
        bytecode_chunk_write(vm, chunk, value, line);
    } else {
        bytecode_chunk_write(vm, chunk, 0xFF, line);
        _bytecode_chunk_write_u24(vm, chunk, value, line);
    }
}

// Writes `opcode` associated with `line` to the chunk's instructions.
static inline void bytecode_chunk_write_opcode(vm *vm, struct bytecode_chunk *chunk,
                                               enum opcode opcode, line_number line)
{
    bytecode_chunk_write(vm, chunk, opcode, line);
}

// Writes `index` associated with `line` to the chunk's instructions.
static inline void bytecode_chunk_write_constant_index(vm *vm, struct bytecode_chunk *chunk,
                                                       constant_index index, line_number line)
{
    _bytecode_chunk_write_compact_u24(vm, chunk, index, line);
}

// Writes `index` associated with `line` to the chunk's instructions.
static inline void bytecode_chunk_write_final_local_name_index(vm *vm, struct bytecode_chunk *chunk,
                                                               final_local_name_index index,
                                                               line_number line)
{
    _bytecode_chunk_write_compact_u24(vm, chunk, index, line);
}

// Writes `count` associated with `line` to the chunk's instructions.
static inline void bytecode_write_pop_count(vm *vm, struct bytecode_chunk *chunk, pop_count count,
                                            line_number line)
{
    _bytecode_chunk_write_compact_u24(vm, chunk, count, line);
}

// Writes `index` associated with `line` to the chunk's instructions.
static inline void bytecode_chunk_write_global_index(vm *vm, struct bytecode_chunk *chunk,
                                                     global_index index, line_number line)
{
    _bytecode_chunk_write_compact_u24(vm, chunk, index, line);
}

// Writes `index` associated with `line` to the chunk's instructions.
static inline void bytecode_chunk_write_upvalue_index(vm *vm, struct bytecode_chunk *chunk,
                                                      upvalue_index index, line_number line)
{
    _bytecode_chunk_write_compact_u24(vm, chunk, index, line);
}

// Writes `index` associated with `line` to the chunk's instructions.
static inline void bytecode_chunk_write_stack_index(vm *vm, struct bytecode_chunk *chunk,
                                                    stack_index index, line_number line)
{
    _bytecode_chunk_write_compact_u24(vm, chunk, index, line);
}

// Writes `offset` associated with `line` to the chunk's instructions.
static inline void bytecode_chunk_write_jump_offset(vm *vm, struct bytecode_chunk *chunk,
                                                    jump_offset offset, line_number line)
{
    // Must update `jump_offset_size` if this is updated to write a differently sized value.
    _bytecode_chunk_write_u16(vm, chunk, offset, line);
}

// Sets the operand of the jump instruction at `target` to `offset`.
static inline void bytecode_chunk_set_jump_offset(struct bytecode_chunk *chunk,
                                                  bytecode_offset target, jump_offset offset)
{
    _bytecode_write_u16(chunk->instructions + target + 1, offset);
}

// Writes `count` associated with `line` to the chunk's instructions.
static inline void bytecode_chunk_write_argument_count(vm *vm, struct bytecode_chunk *chunk,
                                                       argument_count count, line_number line)
{
    _bytecode_chunk_write_compact_u24(vm, chunk, count, line);
}

// Writes `count` associated with `line` to the chunk's instructions.
static inline void bytecode_chunk_write_upvalue_count(vm *vm, struct bytecode_chunk *chunk,
                                                      upvalue_count count, line_number line)
{
    _bytecode_chunk_write_compact_u24(vm, chunk, count, line);
}

// Reads the u8 from the sequence of bytes pointed to by `*p` and moves `*p` past it.
static inline uint8_t bytecode_read_u8(uint8_t **p)
{
    return *(*p)++;
}

// Reads the big-endian u16 from the sequence of bytes pointed to by `*p` and moves `*p` past it.
// Internal function, do not use.
static inline uint16_t _bytecode_read_u16(uint8_t **p)
{
    *p += 2;
    return ((*p)[-2] << 8) | ((*p)[-1]);
}

// Reads the big-endian u24 from the sequence of bytes pointed to by `*p` and moves `*p` past it.
// Internal function, do not use.
static inline uint32_t _bytecode_read_u24(uint8_t **p)
{
    *p += 3;
    return ((*p)[-3] << 16) | ((*p)[-2] << 8) | ((*p)[-1]);
}

// Reads the value encoded by `_bytecode_chunk_write_compact_u24()` from the sequence of bytes
// pointed to by `*p` and moves `*p` past it.
// Internal function, do not use.
static inline uint32_t _bytecode_read_compact_u24(uint8_t **p)
{
    uint8_t index = bytecode_read_u8(p);
    if (index < 0xFF)
        return index;
    return _bytecode_read_u24(p);
}

// Reads the opcode from the sequence of bytes pointed to by `*p` and moves `*p` past it.
static inline enum opcode bytecode_read_opcode(uint8_t **p)
{
    return bytecode_read_u8(p);
}

// Reads the constant index from the sequence of bytes pointed to by `*p` and moves `*p` past it.
static inline constant_index bytecode_read_constant_index(uint8_t **p)
{
    return _bytecode_read_compact_u24(p);
}

// Reads the pop count from the sequence of bytes pointed to by `*p` and moves `*p` past it.
static inline pop_count bytecode_read_pop_count(uint8_t **p)
{
    return _bytecode_read_compact_u24(p);
}

// Reads the final local name index from the bytes pointed to by `*p` and moves `*p` past it.
static inline final_local_name_index bytecode_read_final_local_name_index(uint8_t **p)
{
    return _bytecode_read_compact_u24(p);
}

// Reads the global index from the sequence of bytes pointed to by `*p` and moves `*p` past it.
static inline global_index bytecode_read_global_index(uint8_t **p)
{
    return _bytecode_read_compact_u24(p);
}

// Reads the upvalue index from the sequence of bytes pointed to by `*p` and moves `*p` past it.
static inline upvalue_index bytecode_read_upvalue_index(uint8_t **p)
{
    return _bytecode_read_compact_u24(p);
}

// Reads the stack index from the sequence of bytes pointed to by `*p` and moves `*p` past it.
static inline stack_index bytecode_read_stack_index(uint8_t **p)
{
    return _bytecode_read_compact_u24(p);
}

// Reads the jump offset from the sequence of bytes pointed to by `*p` and moves `*p` past it.
static inline jump_offset bytecode_read_jump_offset(uint8_t **p)
{
    return _bytecode_read_u16(p);
}

// Reads the argument count from the sequence of bytes pointed to by `*p` and moves `*p` past it.
static inline argument_count bytecode_read_argument_count(uint8_t **p)
{
    return _bytecode_read_compact_u24(p);
}

// Reads the upvalue count from the sequence of bytes pointed to by `*p` and moves `*p` past it.
static inline upvalue_count bytecode_read_upvalue_count(uint8_t **p)
{
    return _bytecode_read_compact_u24(p);
}

#endif
