#include "compiler.h"

#include <assert.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "bytecode.h"
#include "debug.h"
#include "dynamic_array.h"
#include "errors.h"
#include "lexer.h"
#include "map.h"
#include "object.h"
#include "strings.h"
#include "value.h"

// Represents a local variable.
struct local {
    // Pointer to the start of the variable's name.
    // This is not a null-terminated string, so `name_len` must be used to read it.
    const char *name;
    size_t name_len;
    // Depth of the scope that the variable was defined in (see `compiler._scope_depth`)
    int scope_depth;
    bool is_initialising; // Whether the variable is currently being initialised
};

// Represents a break or continue statement that has been compiled.
struct loop_jump_statement {
    // Where the corresponding forward jump instruction was written to in the bytecode chunk
    bytecode_offset instruction;
    // Number of local variables in scope at the statements position
    size_t locals_count;
    int line; // Line that the statement appeared on
};

// Contains state related to the compilation of a loop.
struct loop_state {
    // Array of break statements compiled since compilation of the loop started
    struct loop_jump_statement *break_stmts;
    size_t break_stmts_len; // Number of elements in `break_stmts`
    size_t break_stmts_cap; // Number of elements space has been allocated for in `break_stmts`
    // Array of continue statements compiled since compilation of the loop started
    struct loop_jump_statement *continue_stmts;
    size_t continue_stmts_len; // Number of elements in `continue_stmts`
    size_t continue_stmts_cap; // Number of elements space has been allocated for in `continue_stmts`
};

// Compiler that parses Lox source code and writes the compiled bytecode into a caller provided
// chunk.
// Must be initialised with `compiler_init()` and freed with `compiler_free()` after use.
typedef struct {
    // Dependencies
    vm *_vm; // VM the source is being compiled for
    const char *_source; // Source being compiled
    lexer _lexer; // Lexer set up to lex the source
    struct bytecode_chunk *_out; // Chunk that bytecode is written to
    // Token currently being considered.
    // Don't modify outside of `compiler_advance()`.
    struct token _token;
    // Token after current token in the source.
    // Don't modify outside of `compiler_advance()`.
    struct token _next_token;
    // Token before current token in the source.
    // Don't modify outside of `compiler_advance()`.
    struct token _prev_token;
    // Scope depth of the current token. 0 is global scope, 1 is inside a block at global scope, 2
    // is inside a block in that block, etc.
    // Don't modify outside of `compiler_begin_scope()` and `compiler_end_scope()`.
    int _scope_depth;
    // Indicates that an error has been encountered which needs recovering from.
    // Don't enable outside of `compiler_report_errorf()`.
    bool _in_panic_mode;
    // Whether the compiler has encountered any errors.
    // Don't modify outside of `compiler_report_errorf()`.
    bool _had_error;
    // Map from constant values to their index in the bytecode chunk's constants.
    // Don't use outside of `compiler_emit_constant_index()`.
    map _constant_indexes;
    // Array of local variables in scope at the current token's position, ordered by the position of
    // their declaration in the source.
    // Don't use outside of `compiler_end_scope()`, `compiler_add_local()`,
    // `compiler_resolve_local()`, and `compiler_finish_initialising_local()`.
    struct local *_locals;
    // Number of local variables in scope at the current token's position.
    // Don't modify outside of `compiler_end_scope()` and `compiler_declare_local()`.
    size_t _locals_count;
    // Number of elements space has been allocated for in the locals array.
    // Don't use outside of `compiler_declare_local()`.
    size_t _locals_cap;
    // Number of elements that would be in the value stack if execution reached the current token's
    // position.
    // This is increased or decreased when the opcode for an instruction which increases or
    // decreases the number of elements in the stack is emitted.
    // Don't use outside of `compiler_emit_opcode()` and `compiler_emit_pops()`.
    stack_index _stack_depth;
    // State of the loop currently being compiled.
    // Don't use outside of `compiler_begin_loop()`, `compiler_end_loop()`,
    // `compiler_compile_break_stmt()`, `compiler_finish_compiling_break_stmts()`,
    // `compiler_compile_continue_stmt()`, or `compiler_finish_compiling_continue_stmts()`.
    struct loop_state _loop_state;
} compiler;

static struct loop_state compiler_begin_loop(compiler *);
static void compiler_compile_decl(compiler *);
static void compiler_compile_expr(compiler *);
static void compiler_compile_expr_stmt(compiler *);
static void compiler_compile_stmt(compiler *);
static void compiler_compile_var_decl(compiler *);
static void compiler_end_loop(compiler *, struct loop_state);
static void compiler_finish_compiling_break_stmts(compiler *);
static void compiler_finish_compiling_continue_stmts(compiler *);

// Calls `compiler_report_errorf()` with a range spanning the provided token.
#define COMPILER_REPORT_TOKEN_ERRORF(compiler, token, format, ...)                           \
    compiler_report_errorf((compiler), (token).start, (token).start + (token).len, (format), \
                           __VA_ARGS__)

// Prints an error relating to the source range `[start, end)` with a formatted message and enters
// panic mode. If the compiler is already in panic mode, then this is a no-op. See
// `print_invalid_range_error()` for how the error is printed.
static __attribute__((format(printf, 4, 5))) void compiler_report_errorf(compiler *compiler,
                                                                         const char *start,
                                                                         const char *end,
                                                                         const char *format, ...)
{
    if (compiler->_in_panic_mode)
        return;
    compiler->_in_panic_mode = true;
    compiler->_had_error = true;
    char *msg;
    va_list args;
    va_start(args, format);
    int size = vasprintf(&msg, format, args);
    if (size < 0) {
        panicf("clox: compiler: encoding error formatting \"%s\"\n", format);
    }
    va_end(args);
    print_invalid_range_error(msg, compiler->_source, start, end);
    free(msg);
}

// Moves the current token forwards in the source until it's valid. An error is reported for each
// `TOKEN_ERROR` passed.
static void compiler_advance(compiler *compiler)
{
    while (true) {
        compiler->_prev_token = compiler->_token;
        compiler->_token = compiler->_next_token;
        compiler->_next_token = lexer_next(&compiler->_lexer);
        if (compiler->_token.type != TOKEN_ERROR)
            break;
        COMPILER_REPORT_TOKEN_ERRORF(compiler, compiler->_token, "%s", compiler->_token.error_msg);
    }
}

// Initialises `compiler` for compiling `source` and writing the compiled bytecode into `out`.
// `source` and `out` must remain valid until `compiler_free()` is called.
static void compiler_init(compiler *compiler, vm *vm, const char *source,
                          struct bytecode_chunk *out)
{
    compiler->_vm = vm;
    compiler->_source = source;
    lexer_init(&compiler->_lexer, source);
    compiler->_out = out;

    compiler->_prev_token = (struct token){ .type = TOKEN_ERROR };
    compiler->_token = (struct token){ .type = TOKEN_ERROR };
    compiler->_next_token = lexer_next(&compiler->_lexer);
    compiler_advance(compiler); // Populate current token
    compiler->_scope_depth = 0;
    compiler->_in_panic_mode = false;
    compiler->_had_error = false;

    map_init(&compiler->_constant_indexes);
    compiler->_locals = NULL;
    compiler->_locals_count = 0;
    compiler->_locals_cap = 0;
    compiler->_stack_depth = 0;
    compiler->_loop_state = (struct loop_state){ 0 };
}

// Frees the memory associated with `compiler`.
static void compiler_free(compiler *compiler)
{
    lexer_free(&compiler->_lexer);
    map_free(compiler->_vm, &compiler->_constant_indexes);
    free(compiler->_locals);
}

// Calls `compiler_match()` with a variable number of arguments.
#define COMPILER_MATCH(compiler, ...)                              \
    compiler_match((compiler), (enum token_type[]){ __VA_ARGS__ }, \
                   sizeof((enum token_type[]){ __VA_ARGS__ }) / sizeof(enum token_type))

// Reports whether the current token is one of the `count` given `types` and advances the compiler
// if so. Use the `COMPILER_MATCH` macro to call this function with a variable number of arguments.
static bool compiler_match(compiler *compiler, enum token_type *types, int count)
{
    for (int i = 0; i < count; i++)
        if (compiler->_token.type == types[i]) {
            compiler_advance(compiler);
            return true;
        }
    return false;
}

// Checks whether the current token has `type`. If it does, the compiler is advanced. Otherwise, an
// error is reported.
static void compiler_expect(compiler *compiler, enum token_type type)
{
    if (!COMPILER_MATCH(compiler, type))
        COMPILER_REPORT_TOKEN_ERRORF(compiler, compiler->_token, "expected '%s'",
                                     token_type_string(type));
}

// Returns the offset which will be written to next in the output bytecode chunk.
static bytecode_offset compiler_bytecode_offset(compiler *compiler)
{
    return compiler->_out->len;
}

// Writes `opcode` corresponding to `line` to the output bytecode chunk.
static void compiler_emit_opcode(compiler *compiler, enum opcode opcode, int line)
{
    bytecode_chunk_write(compiler->_out, opcode, line);
    switch (opcode) {
    case OP_CONSTANT:
    case OP_NIL:
    case OP_TRUE:
    case OP_FALSE:
    case OP_GET_GLOBAL:
    case OP_GET_LOCAL:
        compiler->_stack_depth++;
        break;
    case OP_POP:
    case OP_DEFINE_GLOBAL:
    case OP_DEFINE_GLOBAL_FINAL:
    case OP_EQUAL:
    case OP_NOT_EQUAL:
    case OP_LESS:
    case OP_LESS_EQUAL:
    case OP_GREATER:
    case OP_GREATER_EQUAL:
    case OP_ADD:
    case OP_SUBTRACT:
    case OP_MULTIPLY:
    case OP_DIVIDE:
    case OP_MODULO:
    case OP_PRINT:
        compiler->_stack_depth--;
        break;
    case OP_POP_N:
        // This reduces the stack depth by n but we don't know what n is until the operand has been
        // written, so we reduce the stack size in `compiler_emit_pops()` instead.
    case OP_SET_GLOBAL:
    case OP_DEFINE_LOCAL:
    case OP_DEFINE_LOCAL_FINAL:
    case OP_SET_LOCAL:
    case OP_NOT:
    case OP_NEGATE:
    case OP_JUMP:
    case OP_JUMP_BACK:
    case OP_JUMP_IF_TRUE:
    case OP_JUMP_IF_FALSE:
    case OP_RETURN: // TODO
        break;
    }
    if (compiler->_stack_depth > compiler->_out->max_stack_depth)
        compiler->_out->max_stack_depth = compiler->_stack_depth;
}

// Writes `operand` corresponding to `line` to the output bytecode chunk.
static void compiler_emit_operand(compiler *compiler, uint8_t operand, int line)
{
    bytecode_chunk_write(compiler->_out, operand, line);
}

// Ensures that value `v` corresponding to `line` is in the output bytecode chunk's constants,
// encodes the corresponding index, and writes it to the chunk.
static void compiler_emit_constant_index(compiler *compiler, value v, int line)
{
    value index_value;
    constant_index index;
    if (map_get(&compiler->_constant_indexes, v, &index_value)) {
        index = value_as_number(index_value);
    } else {
        index = bytecode_chunk_add_constant(compiler->_out, v);
        if (index > MAX_CONSTANT_INDEX)
            panicf("maximum number of constants (%llu) used", (uint64_t)MAX_CONSTANT_INDEX + 1);
        map_set(compiler->_vm, &compiler->_constant_indexes, v, value_number(index));
    }
    if (index < 255) {
        compiler_emit_operand(compiler, index, line);
    } else {
        compiler_emit_operand(compiler, 0xFF, line);
        compiler_emit_operand(compiler, (index >> 16) & 0xFF, line);
        compiler_emit_operand(compiler, (index >> 8) & 0xFF, line);
        compiler_emit_operand(compiler, (index >> 0) & 0xFF, line);
    }
}

// Encodes and writes `index` to the output bytecode chunk.
static void compiler_emit_stack_index(compiler *compiler, stack_index index, int line)
{
    if (index < 255) {
        compiler_emit_operand(compiler, index, line);
    } else {
        compiler_emit_operand(compiler, 0xFF, line);
        compiler_emit_operand(compiler, (index >> 16) & 0xFF, line);
        compiler_emit_operand(compiler, (index >> 8) & 0xFF, line);
        compiler_emit_operand(compiler, (index >> 0) & 0xFF, line);
    }
}

static bool is_forward_jump_opcode(enum opcode opcode)
{
    return opcode == OP_JUMP || opcode == OP_JUMP_IF_TRUE || opcode == OP_JUMP_IF_FALSE;
}

// Writes a forward jump instruction using `opcode` corresponding to `line` to the output bytecode
// chunk and returns the offset of the written instruction in the chunk.
// A placeholder jump offset operand is written which can be set later by passing the returned
// bytecode offset to `compiler_set_jump_offset()`.
static bytecode_offset compiler_emit_forward_jump(compiler *compiler, enum opcode opcode, int line)
{
    assert(is_forward_jump_opcode(opcode));
    bytecode_offset jump = compiler_bytecode_offset(compiler);
    compiler_emit_opcode(compiler, opcode, line);
    compiler_emit_operand(compiler, 0xFF, line);
    compiler_emit_operand(compiler, 0xFF, line);
    return jump;
}

// Writes a pop instruction corresponding to `line` to the output bytecode that pops `count` values
// from the stack.
static void compiler_emit_pops(compiler *compiler, pop_count count, int line)
{
    if (count == 0)
        return;
    if (count == 1) {
        compiler_emit_opcode(compiler, OP_POP, line);
    } else {
        compiler_emit_opcode(compiler, OP_POP_N, line);
        if (count < 255) {
            compiler_emit_operand(compiler, count, line);
        } else {
            compiler_emit_operand(compiler, 0xFF, line);
            compiler_emit_operand(compiler, (count >> 16) & 0xFF, line);
            compiler_emit_operand(compiler, (count >> 8) & 0xFF, line);
            compiler_emit_operand(compiler, (count >> 0) & 0xFF, line);
        }
        // This is decremented in `compiler_emit_opcode()` for all other instructions. We need to
        // handle this one here because the effect on the stack depth is not static. We don't need
        // to worry about updating the output bytecode's max stack depth this instruction will
        // always decrease the current stack depth.
        compiler->_stack_depth -= count;
    }
}

// Sets the operand of the forward jump instruction at offset `target` in the output bytecode chunk
// so that it jumps to the next offset written to. I.e. if this function is called just before
// another which emits some bytecode, the jump instruction will jump to that emitted bytecode.
static void compiler_set_forward_jump_offset(compiler *compiler, bytecode_offset target)
{
    enum opcode opcode = compiler->_out->instructions[target];
    assert(is_forward_jump_opcode(opcode));
    // `target` is the offset of the jump opcode which is then followed by 2 bytes for the operand.
    // The jump offset counts from the offset after the operand bytes we need to take this away from
    // the result (1 for the opcode, 2 for the operand).
    uint64_t jump_offset = compiler_bytecode_offset(compiler) - target - 3;
    if (jump_offset > MAX_JUMP_OFFSET)
        panicf("maximum jump offset (%llu) exceeded", (uint64_t)MAX_JUMP_OFFSET);
    compiler->_out->instructions[target + 1] = (jump_offset >> 8) & 0xFF;
    compiler->_out->instructions[target + 2] = (jump_offset >> 0) & 0xFF;
}

// Writes a backward jump instruction corresponding to `line` to the output bytecode chunk that
// jumps back to `target`.
static void compiler_emit_backward_jump(compiler *compiler, bytecode_offset target, int line)
{
    compiler_emit_opcode(compiler, OP_JUMP_BACK, line);
    // The jump offset counts from the offset after the 2 operand bytes which we haven't written yet
    // so we need we need to add this to the result.
    uint64_t jump_offset = compiler_bytecode_offset(compiler) - target + 2;
    if (jump_offset > MAX_JUMP_OFFSET)
        panicf("maximum jump offset (%llu) exceeded", (uint64_t)MAX_JUMP_OFFSET);
    compiler_emit_operand(compiler, (jump_offset >> 8) & 0xFF, line);
    compiler_emit_operand(compiler, (jump_offset >> 0) & 0xFF, line);
}

// Returns the identifier at the current token and advances the compiler past it.
// If the current token is not an identifier, then as error is reported, the compiler is not
// advanced, and `NULL` is returned.
static struct object_string *compiler_parse_ident(compiler *compiler, const char *error_msg)
{
    if (!COMPILER_MATCH(compiler, TOKEN_IDENT)) {
        COMPILER_REPORT_TOKEN_ERRORF(compiler, compiler->_token, "%s", error_msg);
        return NULL;
    }
    return object_string_copy(compiler->_vm, compiler->_prev_token.start,
                              compiler->_prev_token.len);
}

// Signals that a new lexical scope has been entered.
static void compiler_begin_scope(compiler *compiler)
{
    compiler->_scope_depth++;
}

// Signals that a lexical scope has been exited on `line`.
static void compiler_end_scope(compiler *compiler, int line)
{
    size_t start_locals_count = compiler->_locals_count;
    for (ptrdiff_t i = compiler->_locals_count - 1;
         i >= 0 && compiler->_locals[i].scope_depth == compiler->_scope_depth; i--) {
        compiler->_locals_count--;
    }
    size_t locals_in_scope = start_locals_count - compiler->_locals_count;
    compiler_emit_pops(compiler, locals_in_scope, line);
    compiler->_scope_depth--;
}

// Returns the index in the VM's value stack of the local variable that `ident` refers to or
// `MAX_STACK_INDEX + 1` if one is not found.
static stack_index compiler_resolve_local(compiler *compiler, struct token ident)
{
    for (ptrdiff_t i = compiler->_locals_count - 1; i >= 0; i--) {
        struct local local = compiler->_locals[i];
        if (local.name_len == ident.len && memcmp(local.name, ident.start, local.name_len) == 0) {
            if (local.is_initialising)
                COMPILER_REPORT_TOKEN_ERRORF(compiler, ident, "'%.*s' read in its own initialiser",
                                             (int)ident.len, ident.start);
            return i;
        }
    }
    return MAX_STACK_INDEX + 1;
}

// Ensures that when MAX_STACK_INDEX + 1 is a valid value to return from
_Static_assert((constant_index)-1 > 0, "stack_index must be unsigned");
_Static_assert(MAX_STACK_INDEX < (stack_index)-1, "MAX_STACK_INDEX must be < max(stack_index)");

// Reports whether the string `s` of `len` characters is the blank identifier `_`.
static bool is_blank_ident(const char *s, size_t len)
{
    return len == 1 && s[0] == '_';
}

// Declares the local variable corresponding to `ident` or reports an error if one has already been
// declared at the current scope depth.
// A declared local variable name can later be resolved to its index in the VM's value stack with
// `compiler_resolve_local()`.
// The variable starts in an initialising state. This function returns its index in the VM's value
// stack which should be passed to `compiler_finish_initialising_local()` once the declaration
// introducing the variable has been compiled.
static stack_index compiler_declare_local(compiler *compiler, struct token ident)
{
    for (ptrdiff_t i = compiler->_locals_count - 1;
         i >= 0 && compiler->_locals[i].scope_depth == compiler->_scope_depth; i--) {
        struct local local = compiler->_locals[i];
        if (local.name_len == ident.len && memcmp(local.name, ident.start, local.name_len) == 0 &&
            !is_blank_ident(local.name, local.name_len))
            COMPILER_REPORT_TOKEN_ERRORF(compiler, ident, "'%.*s' has already been declared",
                                         (int)ident.len, ident.start);
    }
    stack_index index = compiler->_locals_count;
    DYNAMIC_ARRAY_GROW(compiler->_locals, compiler->_locals_cap, compiler->_locals_count + 1);
    if (index > MAX_STACK_INDEX)
        panicf("maximum number of local variables (%llu) used", (uint64_t)MAX_STACK_INDEX + 1);
    struct local *local = &compiler->_locals[compiler->_locals_count++];
    local->name = ident.start;
    local->name_len = ident.len;
    local->scope_depth = compiler->_scope_depth;
    local->is_initialising = true;
    return index;
}

// Marks the local variable stored at index `i` in the VM's value stack as initialised.
static void compiler_finish_initialising_local(compiler *compiler, stack_index i)
{
    compiler->_locals[i].is_initialising = false;
}

// Emits the bytecode for the loop jump statement starting at the current `type `token and advances
// the compiler past it.
// A loop jump statement can't be compiled completely until the point in the loop that the statement
// should jump past has been compiled. Once this has been done, pass the returned `struct
// loop_jump_statement` to `compiler_finish_compiling_loop_jump_stmt()`.
static struct loop_jump_statement compiler_compile_loop_jump_stmt(compiler *compiler,
                                                                  enum token_type type)
{
    assert(type == TOKEN_BREAK || type == TOKEN_CONTINUE);
    int line = compiler->_token.line;
    compiler_expect(compiler, type);
    compiler_expect(compiler, TOKEN_SEMICOLON);
    bytecode_offset jump = compiler_emit_forward_jump(compiler, OP_JUMP, line);
    return (struct loop_jump_statement){ .instruction = jump,
                                         .locals_count = compiler->_locals_count,
                                         .line = line };
}

// Finishes the compilation of `stmt`.
// Must be called straight after all bytecode that the statement should jump past has been emitted.
static void compiler_finish_compiling_loop_jump_stmt(compiler *compiler,
                                                     struct loop_jump_statement stmt)
{
    // For every loop jump statement, we will have emitted a forward jump instruction whose offset
    // we need to update so that it points to the next bytecode offset written to. If the statement
    // came after some local variables had been declared, the jump will skip over the instructions
    // popping those variables from the stack, so we need to do that as well.
    compiler_set_forward_jump_offset(compiler, stmt.instruction);
    size_t locals_declared_before_break = stmt.locals_count - compiler->_locals_count;
    compiler_emit_pops(compiler, locals_declared_before_break, stmt.line);
}

// Precedence levels of the different operators, ordered by increasing precedence.
enum precedence_level {
    // Default value used when a precedence level is expected but none are applicable
    PREC_NONE,
    PREC_COMMA, // ,
    PREC_ASSIGNMENT, // =
    PREC_TERNARY, // ?:
    PREC_LOGICAL_OR, // or
    PREC_LOGICAL_AND, // and
    PREC_EQUALITY, // == !=
    PREC_RELATIONAL, // < <= > >=
    PREC_ADDITIVE, // + -
    PREC_MULTIPLICATIVE, // * / %
    PREC_UNARY, // ! -
    PREC_POSTFIX, // () [] .
};

static void compiler_compile_expr_at_level(compiler *, enum precedence_level);

// Returns the precedence level of `type` as an infix operator or `PREC_NONE` if `type` is not an
// infix operator.
static enum precedence_level infix_operator_precedence_level(enum token_type type)
{
    switch (type) {
    case TOKEN_COMMA:
        return PREC_COMMA;
    case TOKEN_QUESTION:
        return PREC_TERNARY;
    case TOKEN_OR:
        return PREC_LOGICAL_OR;
    case TOKEN_AND:
        return PREC_LOGICAL_AND;
    case TOKEN_EQUAL_EQUAL:
    case TOKEN_BANG_EQUAL:
        return PREC_EQUALITY;
    case TOKEN_LESS:
    case TOKEN_LESS_EQUAL:
    case TOKEN_GREATER:
    case TOKEN_GREATER_EQUAL:
        return PREC_RELATIONAL;
    case TOKEN_PLUS:
    case TOKEN_MINUS:
        return PREC_ADDITIVE;
    case TOKEN_ASTERISK:
    case TOKEN_SLASH:
    case TOKEN_PERCENT:
        return PREC_MULTIPLICATIVE;
    // case TOKEN_LEFT_PAREN:
    // case TOKEN_LEFT_BRACK:
    // case TOKEN_DOT:
    //     return PREC_POSTFIX;
    default:
        return PREC_NONE;
    }
}

// Type of function that emits the bytecode for the expression starting at the current token
// consisting only of operators with at least the given precedence `level`, and advances the
// compiler past it.
// If an unexpected token is encountered, then an error is reported and the compiler is not advanced
// past it.
typedef void (*expr_compiler)(compiler *compiler, enum precedence_level level);

// Type of function which behaves in the same way as `expr_compiler` except it compiles the infix
// expression whose first operand has already been compiled. That is, it expects the current token
// to be the infix operator.
typedef void (*infix_expr_compiler)(compiler *compiler, enum precedence_level level);

// `expr_compiler` for a keyword literal expression.
static void compiler_compile_keyword_literal_expr(compiler *compiler, enum precedence_level level)
{
    (void)level;
    enum opcode instruction;
    if (COMPILER_MATCH(compiler, TOKEN_NIL)) {
        instruction = OP_NIL;
    } else if (COMPILER_MATCH(compiler, TOKEN_TRUE)) {
        instruction = OP_TRUE;
    } else if (COMPILER_MATCH(compiler, TOKEN_FALSE)) {
        instruction = OP_FALSE;
    } else {
        COMPILER_REPORT_TOKEN_ERRORF(compiler, compiler->_token, "%s", "expected literal keyword");
        return;
    }
    compiler_emit_opcode(compiler, instruction, compiler->_prev_token.line);
}

// `expr_compiler` for an identifier expression.
static void compiler_compile_ident_expr(compiler *compiler, enum precedence_level level)
{
    int line = compiler->_token.line;
    struct token ident = compiler->_token;
    struct object_string *name = compiler_parse_ident(compiler, "expected identifier");
    if (name == NULL)
        return;

    bool is_get = true;
    if (compiler->_token.type == TOKEN_EQUAL) {
        if (level > PREC_ASSIGNMENT) {
            COMPILER_REPORT_TOKEN_ERRORF(compiler, compiler->_token, "%s",
                                         "invalid assignment target");
            return;
        }
        compiler_advance(compiler);
        compiler_compile_expr_at_level(compiler, PREC_ASSIGNMENT);
        is_get = false;
    }

    if (is_blank_ident(name->data, name->len)) {
        if (is_get)
            COMPILER_REPORT_TOKEN_ERRORF(compiler, ident, "%s", "'_' cannot be used as a value");
        else
            return;
    }

    stack_index index = compiler_resolve_local(compiler, ident);
    if (index <= MAX_STACK_INDEX) {
        enum opcode instruction = is_get ? OP_GET_LOCAL : OP_SET_LOCAL;
        compiler_emit_opcode(compiler, instruction, line);
        compiler_emit_stack_index(compiler, index, line);
    } else {
        enum opcode instruction = is_get ? OP_GET_GLOBAL : OP_SET_GLOBAL;
        compiler_emit_opcode(compiler, instruction, line);
        compiler_emit_constant_index(compiler, value_string(name), line);
    }
}

// `expr_compiler` for a string literal expression.
static void compiler_compile_string_literal_expr(compiler *compiler, enum precedence_level level)
{
    (void)level;
    compiler_expect(compiler, TOKEN_STRING);
    // First and last characters are " so discount these from the length and start data 1 character
    // in
    struct object_string *string = object_string_copy(
        compiler->_vm, compiler->_prev_token.start + 1, compiler->_prev_token.len - 2);
    compiler_emit_opcode(compiler, OP_CONSTANT, compiler->_prev_token.line);
    compiler_emit_constant_index(compiler, value_string(string), compiler->_prev_token.line);
}

// `expr_compiler` for a number literal expression.
static void compiler_compile_number_literal_expr(compiler *compiler, enum precedence_level level)
{
    (void)level;
    compiler_expect(compiler, TOKEN_NUMBER);
    double number = strtod(compiler->_prev_token.start, NULL);
    compiler_emit_opcode(compiler, OP_CONSTANT, compiler->_prev_token.line);
    compiler_emit_constant_index(compiler, value_number(number), compiler->_prev_token.line);
}

// `expr_compiler` for a group expression.
static void compiler_compile_group_expr(compiler *compiler, enum precedence_level level)
{
    (void)level;
    compiler_expect(compiler, TOKEN_LEFT_PAREN);
    compiler_compile_expr(compiler);
    compiler_expect(compiler, TOKEN_RIGHT_PAREN);
}

// `expr_compiler` for a unary expression.
static void compiler_compile_unary_expr(compiler *compiler, enum precedence_level level)
{
    (void)level;
    int line = compiler->_token.line;
    enum opcode instruction;
    if (COMPILER_MATCH(compiler, TOKEN_MINUS)) {
        instruction = OP_NEGATE;
    } else if (COMPILER_MATCH(compiler, TOKEN_BANG)) {
        instruction = OP_NOT;
    } else {
        COMPILER_REPORT_TOKEN_ERRORF(compiler, compiler->_token, "%s", "expected unary operator");
        return;
    }
    compiler_compile_expr_at_level(compiler, PREC_UNARY);
    compiler_emit_opcode(compiler, instruction, line);
}

// Returns the compiler for an expression starting with a `type` token or `NULL` if there is none.
static expr_compiler expr_compiler_for_prefix(enum token_type type)
{
    switch (type) {
    case TOKEN_NIL:
    case TOKEN_TRUE:
    case TOKEN_FALSE:
        return compiler_compile_keyword_literal_expr;
    case TOKEN_IDENT:
        return compiler_compile_ident_expr;
    case TOKEN_STRING:
        return compiler_compile_string_literal_expr;
    case TOKEN_NUMBER:
        return compiler_compile_number_literal_expr;
    case TOKEN_LEFT_PAREN:
        return compiler_compile_group_expr;
    case TOKEN_MINUS:
    case TOKEN_BANG:
        return compiler_compile_unary_expr;
    default:
        return NULL;
    }
}

// `infix_expr_compiler` for a comma expression.
static void compiler_compile_comma_expr(compiler *compiler, enum precedence_level level)
{
    (void)level;
    int line = compiler->_token.line;
    compiler_expect(compiler, TOKEN_COMMA);
    compiler_compile_expr_at_level(compiler, PREC_COMMA);
    compiler_emit_pops(compiler, 1, line);
}


// `infix_expr_compiler` for a ternary expression.
static void compiler_compile_ternary_expr(compiler *compiler, enum precedence_level level)
{
    (void)level;
    int line = compiler->_token.line;
    compiler_expect(compiler, TOKEN_QUESTION);
    bytecode_offset else_jump = compiler_emit_forward_jump(compiler, OP_JUMP_IF_FALSE, line);
    compiler_emit_pops(compiler, 1, line);
    compiler_compile_expr(compiler);
    bytecode_offset end_jump = compiler_emit_forward_jump(compiler, OP_JUMP, line);
    compiler_expect(compiler, TOKEN_COLON);
    compiler_set_forward_jump_offset(compiler, else_jump);
    compiler_emit_pops(compiler, 1, line);
    compiler_compile_expr_at_level(compiler, PREC_TERNARY);
    compiler_set_forward_jump_offset(compiler, end_jump);
}

// `infix_expr_compiler` for an or expression.
static void compiler_compile_or_expr(compiler *compiler, enum precedence_level level)
{
    int line = compiler->_token.line;
    compiler_expect(compiler, TOKEN_OR);
    bytecode_offset end_jump = compiler_emit_forward_jump(compiler, OP_JUMP_IF_TRUE, line);
    compiler_emit_pops(compiler, 1, line);
    compiler_compile_expr_at_level(compiler, level + 1);
    compiler_set_forward_jump_offset(compiler, end_jump);
}

// `infix_expr_compiler` for an and expression.
static void compiler_compile_and_expr(compiler *compiler, enum precedence_level level)
{
    int line = compiler->_token.line;
    compiler_expect(compiler, TOKEN_AND);
    bytecode_offset end_jump = compiler_emit_forward_jump(compiler, OP_JUMP_IF_FALSE, line);
    compiler_emit_pops(compiler, 1, line);
    compiler_compile_expr_at_level(compiler, level + 1);
    compiler_set_forward_jump_offset(compiler, end_jump);
}

// `infix_expr_compiler` for a binary expression.
static void compiler_compile_binary_expr(compiler *compiler, enum precedence_level level)
{
    int line = compiler->_token.line;
    enum opcode instruction;
    if (COMPILER_MATCH(compiler, TOKEN_PLUS)) {
        instruction = OP_ADD;
    } else if (COMPILER_MATCH(compiler, TOKEN_MINUS)) {
        instruction = OP_SUBTRACT;
    } else if (COMPILER_MATCH(compiler, TOKEN_ASTERISK)) {
        instruction = OP_MULTIPLY;
    } else if (COMPILER_MATCH(compiler, TOKEN_SLASH)) {
        instruction = OP_DIVIDE;
    } else if (COMPILER_MATCH(compiler, TOKEN_PERCENT)) {
        instruction = OP_MODULO;
    } else if (COMPILER_MATCH(compiler, TOKEN_LESS)) {
        instruction = OP_LESS;
    } else if (COMPILER_MATCH(compiler, TOKEN_LESS_EQUAL)) {
        instruction = OP_LESS_EQUAL;
    } else if (COMPILER_MATCH(compiler, TOKEN_GREATER)) {
        instruction = OP_GREATER;
    } else if (COMPILER_MATCH(compiler, TOKEN_GREATER_EQUAL)) {
        instruction = OP_GREATER_EQUAL;
    } else if (COMPILER_MATCH(compiler, TOKEN_EQUAL_EQUAL)) {
        instruction = OP_EQUAL;
    } else if (COMPILER_MATCH(compiler, TOKEN_BANG_EQUAL)) {
        instruction = OP_NOT_EQUAL;
    } else {
        COMPILER_REPORT_TOKEN_ERRORF(compiler, compiler->_token, "%s", "expected binary operator");
        return;
    }
    compiler_compile_expr_at_level(compiler, level + 1);
    compiler_emit_opcode(compiler, instruction, line);
}

// Returns the compiler for an infix expression whose operator is a `type` token or `NULL` if there
// is none.
static infix_expr_compiler infix_expr_compiler_for_operator(enum token_type type)
{
    switch (type) {
    case TOKEN_COMMA:
        return compiler_compile_comma_expr;
    case TOKEN_QUESTION:
        return compiler_compile_ternary_expr;
    case TOKEN_OR:
        return compiler_compile_or_expr;
    case TOKEN_AND:
        return compiler_compile_and_expr;
    case TOKEN_PLUS:
    case TOKEN_MINUS:
    case TOKEN_ASTERISK:
    case TOKEN_SLASH:
    case TOKEN_PERCENT:
    case TOKEN_LESS:
    case TOKEN_LESS_EQUAL:
    case TOKEN_GREATER:
    case TOKEN_GREATER_EQUAL:
    case TOKEN_EQUAL_EQUAL:
    case TOKEN_BANG_EQUAL:
        return compiler_compile_binary_expr;
    default:
        return NULL;
    }
}

// `expr_compiler` for any expression.
static void compiler_compile_expr_at_level(compiler *compiler, enum precedence_level level)
{
    expr_compiler prefix_compiler = expr_compiler_for_prefix(compiler->_token.type);
    if (prefix_compiler == NULL) {
        COMPILER_REPORT_TOKEN_ERRORF(compiler, compiler->_token, "%s", "expected expression");
        return;
    }
    prefix_compiler(compiler, level);

    enum precedence_level infix_level;
    while ((infix_level = infix_operator_precedence_level(compiler->_token.type)) >= level) {
        infix_expr_compiler infix_compiler =
            infix_expr_compiler_for_operator(compiler->_token.type);
        infix_compiler(compiler, infix_level);
    }
}

// Emits the bytecode for the expression starting the current token and advances the compiler past
// it.
static void compiler_compile_expr(compiler *compiler)
{
    compiler_compile_expr_at_level(compiler, PREC_COMMA);
}

// Emits the bytecode for the print statement starting at the current token and advances the
// compiler past it.
static void compiler_compile_print_stmt(compiler *compiler)
{
    int line = compiler->_token.line;
    compiler_expect(compiler, TOKEN_PRINT);
    compiler_compile_expr(compiler);
    compiler_expect(compiler, TOKEN_SEMICOLON);
    compiler_emit_opcode(compiler, OP_PRINT, line);
}

// Emits the bytecode for the if statement starting at the current token and advances the compiler
// past it.
static void compiler_compile_if_stmt(compiler *compiler)
{
    int if_line = compiler->_token.line;
    compiler_expect(compiler, TOKEN_IF);
    compiler_expect(compiler, TOKEN_LEFT_PAREN);

    // Condition
    int condition_line = compiler->_token.line;
    compiler_compile_expr(compiler);
    compiler_expect(compiler, TOKEN_RIGHT_PAREN);
    bytecode_offset else_jump = compiler_emit_forward_jump(compiler, OP_JUMP_IF_FALSE, if_line);

    // Then branch
    compiler_emit_pops(compiler, 1, condition_line);
    compiler_compile_stmt(compiler);
    bytecode_offset end_jump = compiler_emit_forward_jump(compiler, OP_JUMP, if_line);

    // Else branch
    compiler_set_forward_jump_offset(compiler, else_jump);
    compiler_emit_pops(compiler, 1, condition_line);
    if (COMPILER_MATCH(compiler, TOKEN_ELSE)) {
        compiler_compile_stmt(compiler);
    }
    compiler_set_forward_jump_offset(compiler, end_jump);
}

// Emits the bytecode for the while statement starting at the current token and advances the
// compiler past it.
static void compiler_compile_while_stmt(compiler *compiler)
{
    struct loop_state prev_loop_state = compiler_begin_loop(compiler);

    bytecode_offset loop_start = compiler_bytecode_offset(compiler);
    int while_line = compiler->_token.line;
    compiler_expect(compiler, TOKEN_WHILE);
    compiler_expect(compiler, TOKEN_LEFT_PAREN);

    // Condition
    int condition_line = compiler->_token.line;
    compiler_compile_expr(compiler);
    compiler_expect(compiler, TOKEN_RIGHT_PAREN);
    bytecode_offset end_jump = compiler_emit_forward_jump(compiler, OP_JUMP_IF_FALSE, while_line);

    // Body
    compiler_emit_pops(compiler, 1, condition_line);
    compiler_compile_stmt(compiler);
    compiler_finish_compiling_continue_stmts(compiler);
    compiler_emit_backward_jump(compiler, loop_start, while_line);
    compiler_set_forward_jump_offset(compiler, end_jump);
    compiler_emit_pops(compiler, 1, condition_line);
    compiler_finish_compiling_break_stmts(compiler);

    compiler_end_loop(compiler, prev_loop_state);
}

// Emits the bytecode for the for statement starting at the current token and advances the
// compiler past it.
static void compiler_compile_for_stmt(compiler *compiler)
{
    struct loop_state prev_loop_state = compiler_begin_loop(compiler);
    compiler_begin_scope(compiler);

    int for_line = compiler->_token.line;
    compiler_expect(compiler, TOKEN_FOR);
    compiler_expect(compiler, TOKEN_LEFT_PAREN);

    // Initialiser
    if (COMPILER_MATCH(compiler, TOKEN_SEMICOLON))
        ;
    else if (compiler->_token.type == TOKEN_VAR)
        compiler_compile_var_decl(compiler);
    else
        compiler_compile_expr_stmt(compiler);

    // Condition
    bool has_condition = !COMPILER_MATCH(compiler, TOKEN_SEMICOLON);
    bytecode_offset condition_start = compiler_bytecode_offset(compiler);
    int condition_line = compiler->_token.line;
    bytecode_offset end_jump;
    if (has_condition) {
        compiler_compile_expr(compiler);
        compiler_expect(compiler, TOKEN_SEMICOLON);
        end_jump = compiler_emit_forward_jump(compiler, OP_JUMP_IF_FALSE, for_line);
        compiler_emit_pops(compiler, 1, condition_line);
    }

    // Increment
    bool has_increment = !COMPILER_MATCH(compiler, TOKEN_RIGHT_PAREN);
    bytecode_offset increment_start;
    int increment_line = compiler->_token.line;
    bytecode_offset increment_body_jump;
    if (has_increment) {
        increment_body_jump = compiler_emit_forward_jump(compiler, OP_JUMP, for_line);
        increment_start = compiler_bytecode_offset(compiler);
        compiler_compile_expr(compiler);
        compiler_expect(compiler, TOKEN_RIGHT_PAREN);
        compiler_emit_pops(compiler, 1, increment_line);
        if (has_condition)
            compiler_emit_backward_jump(compiler, condition_start, for_line);
    }

    // Body
    if (has_increment)
        compiler_set_forward_jump_offset(compiler, increment_body_jump);
    compiler_compile_stmt(compiler);
    compiler_finish_compiling_continue_stmts(compiler);
    bytecode_offset body_jump_back_target = has_increment ? increment_start : condition_start;
    compiler_emit_backward_jump(compiler, body_jump_back_target, for_line);
    if (has_condition) {
        compiler_set_forward_jump_offset(compiler, end_jump);
        compiler_emit_pops(compiler, 1, condition_line);
    }
    compiler_finish_compiling_break_stmts(compiler);

    compiler_end_scope(compiler, compiler->_prev_token.line);
    compiler_end_loop(compiler, prev_loop_state);
}

// Signals that compilation of a loop has started.
// Must be called before any bytecode for the loop is emitted.
// Returns the state of the loop previously being compiled which must be passed to
// `compiler_end_loop()` after all bytecode for the loop has been emitted.
static struct loop_state compiler_begin_loop(compiler *compiler)
{
    struct loop_state prev_loop_state = compiler->_loop_state;
    compiler->_loop_state.break_stmts = NULL;
    compiler->_loop_state.break_stmts_len = 0;
    compiler->_loop_state.break_stmts_cap = 0;
    compiler->_loop_state.continue_stmts = NULL;
    compiler->_loop_state.continue_stmts_len = 0;
    compiler->_loop_state.continue_stmts_cap = 0;
    return prev_loop_state;
}

// Signals that compilation of a loop has ended.
// Must be called after all bytecode for the loop has been emitted and passed the state of the
// previous loop being compiled returned by `compiler_begin_loop()`.
static void compiler_end_loop(compiler *compiler, struct loop_state prev_loop_state)
{
    free(compiler->_loop_state.break_stmts);
    free(compiler->_loop_state.continue_stmts);
    compiler->_loop_state = prev_loop_state;
}

// Emits the bytecode for the break statement starting at the current token and advances the
// compiler past it.
// A break statement can't be compiled completely until the whole loop that contains it has been
// compiled. Once this has been done, call `compiler_finish_compiling_break_stmts()`.
static void compiler_compile_break_stmt(compiler *compiler)
{
    struct loop_jump_statement stmt = compiler_compile_loop_jump_stmt(compiler, TOKEN_BREAK);
    DYNAMIC_ARRAY_GROW(compiler->_loop_state.break_stmts, compiler->_loop_state.break_stmts_cap,
                       compiler->_loop_state.break_stmts_len + 1);
    compiler->_loop_state.break_stmts[compiler->_loop_state.break_stmts_len++] = stmt;
}

// Finishes the compilation of any break statements encountered since compilation of the current
// loop started.
// Must be called straight after all bytecode for the loop has been emitted.
static void compiler_finish_compiling_break_stmts(compiler *compiler)
{
    for (size_t i = 0; i < compiler->_loop_state.break_stmts_len; i++)
        compiler_finish_compiling_loop_jump_stmt(compiler, compiler->_loop_state.break_stmts[i]);
}

// Emits the bytecode for the continue statement starting at the current token and advances the
// compiler past it.
// A continue statement can't be compiled completely until the whole loop that contains it has been
// compiled. Once this has been done, call `compiler_finish_compiling_continue_stmts()`.
static void compiler_compile_continue_stmt(compiler *compiler)
{
    struct loop_jump_statement stmt = compiler_compile_loop_jump_stmt(compiler, TOKEN_CONTINUE);
    DYNAMIC_ARRAY_GROW(compiler->_loop_state.continue_stmts,
                       compiler->_loop_state.continue_stmts_cap,
                       compiler->_loop_state.continue_stmts_len + 1);
    compiler->_loop_state.continue_stmts[compiler->_loop_state.continue_stmts_len++] = stmt;
}

// Finishes the compilation of any continue statements encountered since compilation of the current
// loop started.
// Must be called straight after the bytecode for the loop body statement has been emitted.
static void compiler_finish_compiling_continue_stmts(compiler *compiler)
{
    for (size_t i = 0; i < compiler->_loop_state.continue_stmts_len; i++)
        compiler_finish_compiling_loop_jump_stmt(compiler, compiler->_loop_state.continue_stmts[i]);
}

// Emits the bytecode for the block starting at the current token and advances the compiler past it.
static void compiler_compile_block(compiler *compiler)
{
    compiler_begin_scope(compiler);
    compiler_expect(compiler, TOKEN_LEFT_BRACE);
    while (compiler->_token.type != TOKEN_RIGHT_BRACE && compiler->_token.type != TOKEN_EOF)
        compiler_compile_decl(compiler);
    compiler_expect(compiler, TOKEN_RIGHT_BRACE);
    compiler_end_scope(compiler, compiler->_prev_token.line);
}

// Emits the bytecode for the statement starting at the current token and advances the compiler past
// it.
static void compiler_compile_expr_stmt(compiler *compiler)
{
    int line = compiler->_token.line;
    compiler_compile_expr(compiler);
    compiler_expect(compiler, TOKEN_SEMICOLON);
    compiler_emit_pops(compiler, 1, line);
}

// Emits the bytecode for the statement starting at the current token and advances the compiler past
// it.
static void compiler_compile_stmt(compiler *compiler)
{
    switch (compiler->_token.type) {
    case TOKEN_PRINT:
        compiler_compile_print_stmt(compiler);
        break;
    case TOKEN_IF:
        compiler_compile_if_stmt(compiler);
        break;
    case TOKEN_WHILE:
        compiler_compile_while_stmt(compiler);
        break;
    case TOKEN_FOR:
        compiler_compile_for_stmt(compiler);
        break;
    case TOKEN_BREAK:
        compiler_compile_break_stmt(compiler);
        break;
    case TOKEN_CONTINUE:
        compiler_compile_continue_stmt(compiler);
        break;
    case TOKEN_LEFT_BRACE:
        compiler_compile_block(compiler);
        break;
    default:
        compiler_compile_expr_stmt(compiler);
        break;
    }
}

// Emits the bytecode for the variable declaration (using either var or final) starting at the
// current token and advances the compiler past it.
static void compiler_compile_var_decl(compiler *compiler)
{
    int line = compiler->_token.line;
    bool is_final = false;
    if (COMPILER_MATCH(compiler, TOKEN_FINAL))
        is_final = true;
    else
        compiler_expect(compiler, TOKEN_VAR);

    struct token ident = compiler->_token;
    struct object_string *name = compiler_parse_ident(compiler, "expected variable name");
    if (name == NULL)
        return;

    stack_index local_index;
    if (compiler->_scope_depth > 0)
        local_index = compiler_declare_local(compiler, ident);

    if (COMPILER_MATCH(compiler, TOKEN_EQUAL))
        compiler_compile_expr(compiler);
    else
        compiler_emit_opcode(compiler, OP_NIL, line);
    compiler_expect(compiler, TOKEN_SEMICOLON);

    if (is_blank_ident(name->data, name->len))
        return;

    if (compiler->_scope_depth > 0) {
        compiler_finish_initialising_local(compiler, local_index);
        if (is_final) {
            compiler_emit_opcode(compiler, OP_DEFINE_LOCAL_FINAL, line);
            compiler_emit_constant_index(compiler, value_string(name), line);
        } else {
            compiler_emit_opcode(compiler, OP_DEFINE_LOCAL, line);
        }
    } else {
        enum opcode instruction = is_final ? OP_DEFINE_GLOBAL_FINAL : OP_DEFINE_GLOBAL;
        compiler_emit_opcode(compiler, instruction, line);
        compiler_emit_constant_index(compiler, value_string(name), line);
    }
}

// Sync advances the compiler until the current token is at the start of a statement.
static void compiler_sync(compiler *compiler)
{
    while (true) {
        switch (compiler->_token.type) {
        case TOKEN_EOF:
        case TOKEN_PRINT:
        case TOKEN_VAR:
        case TOKEN_FINAL:
        case TOKEN_IF:
        case TOKEN_WHILE:
        case TOKEN_FOR:
        case TOKEN_BREAK:
        case TOKEN_CONTINUE:
            // case TOKEN_RETURN:
            // case TOKEN_CLASS:
            return;
        default:
            compiler_advance(compiler);
            if (compiler->_prev_token.type == TOKEN_SEMICOLON)
                return;
            break;
        }
    }
}

// Emits the bytecode for the declaration starting at the current token and advances the compiler
// past it.
static void compiler_compile_decl(compiler *compiler)
{
    switch (compiler->_token.type) {
    case TOKEN_VAR:
    case TOKEN_FINAL:
        compiler_compile_var_decl(compiler);
        break;
    default:
        compiler_compile_stmt(compiler);
        break;
    }
    if (compiler->_in_panic_mode) {
        compiler_sync(compiler);
        compiler->_in_panic_mode = false;
    }
}

// Emits the bytecode for the program and reports whether compilation was successful.
static bool compiler_compile(compiler *compiler)
{
    while (compiler->_token.type != TOKEN_EOF)
        compiler_compile_decl(compiler);
    compiler_emit_opcode(compiler, OP_RETURN, compiler->_token.line);
    bool success = !compiler->_had_error;
#ifdef DEBUG_PRINT_BYTECODE
    if (success)
        disassemble(compiler->_out, "code");
#endif
    return success;
}

bool compile(vm *vm, const char *source, struct bytecode_chunk *out)
{
    compiler compiler;
    compiler_init(&compiler, vm, source, out);
    bool success = compiler_compile(&compiler);
    compiler_free(&compiler);
    return success;
}
