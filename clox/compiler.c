#include "compiler.h"

#include <assert.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#include "bytecode.h"
#include "debug.h"
#include "dynamic_array.h"
#include "errors.h"
#include "lexer.h"
#include "line_number.h"
#include "map.h"
#include "object.h"
#include "strings.h"
#include "value.h"
#include "vm.h"

// Mode that a compiler can operate in.
enum compiler_mode {
    COMPILER_MODE_SCRIPT, // Compiling a top level script
    COMPILER_MODE_FUNCTION, // Compiling a normal function
};

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
    line_number line; // Line that the statement appeared on
};

// State related to the compilation of a loop.
struct loop_state {
    bool is_nested; // Whether the loop is nested inside another one
    // Array of break statements compiled since compilation of the loop started
    struct loop_jump_statement *break_stmts;
    size_t break_stmts_len; // Number of elements in `break_stmts`
    size_t break_stmts_cap; // Number of elements space has been allocated for in `break_stmts`
    // Array of continue statements compiled since compilation of the loop started
    struct loop_jump_statement *continue_stmts;
    size_t continue_stmts_len; // Number of elements in `continue_stmts`
    size_t continue_stmts_cap; // Number of elements space has been allocated for in `continue_stmts`
};

// Reads a stream of tokens from a source and tracks whether errors have been reported while parsing
// them.
// All public fields are read-only.
// Must be initialised with `parser_init()` and deinitialised with `parser_deinit()`.
struct parser {
    struct token token; // Current token in the stream
    struct token next_token; // Token after current token in the source
    struct token prev_token; // Token before current token in the source
    // Indicates that a syntax error has been reported. Use `parser_sync()` to recover from it.
    bool in_panic_mode;
    bool had_error; // Whether any errors have been reported
    // Internal fields below, do not use.
    const char *_source;
    lexer _lexer; // Lexer set up to lex the source
};

// Calls `parser_report_errorf()` with a range spanning the provided token.
#define PARSER_REPORT_ERRORF(parser, token, format, ...)                                 \
    parser_report_errorf((parser), (token).start, (token).start + (token).len, (format), \
                         __VA_ARGS__)

// Works like `parser_report_errorf()`, except it accepts a `va_list` instead of variadic arguments.
static __attribute__((format(printf, 4, 0))) void
parser_report_errorvf(struct parser *parser, const char *start, const char *end, const char *format,
                      va_list args)
{
    parser->had_error = true;
    char *msg;
    xvasprintf(&msg, format, args);
    va_end(args);
    print_invalid_range_error(msg, parser->_source, start, end);
    free(msg);
}

// Prints an error relating to the source range `[start, end)` with a formatted message.
static __attribute__((format(printf, 4, 5))) void parser_report_errorf(struct parser *parser,
                                                                       const char *start,
                                                                       const char *end,
                                                                       const char *format, ...)
{
    va_list args;
    va_start(args, format);
    parser_report_errorvf(parser, start, end, format, args);
    va_end(args);
}

// Calls `parser_report_syntax_errorf()` with a range spanning the provided token.
#define PARSER_REPORT_SYNTAX_ERRORF(parser, token, format, ...)                                 \
    parser_report_syntax_errorf((parser), (token).start, (token).start + (token).len, (format), \
                                __VA_ARGS__)

// Works like `parser_report_syntax_errorf()`, except it accepts a `va_list` instead of variadic
// arguments.
static __attribute__((format(printf, 4, 0))) void
parser_report_syntax_errorvf(struct parser *parser, const char *start, const char *end,
                             const char *format, va_list args)
{
    if (parser->in_panic_mode)
        return;
    parser->in_panic_mode = true;
    parser_report_errorvf(parser, start, end, format, args);
}

// Prints an error relating to the source range `[start, end)` with a formatted message and enters
// panic mode. If the parser is already in panic mode, then this is a no-op
static __attribute__((format(printf, 4, 5))) void
parser_report_syntax_errorf(struct parser *parser, const char *start, const char *end,
                            const char *format, ...)
{
    va_list args;
    va_start(args, format);
    parser_report_syntax_errorvf(parser, start, end, format, args);
    va_end(args);
}

// Moves the current token forwards in the source until it's valid. An error is reported for each
// `TOKEN_ERROR` passed.
static void parser_advance(struct parser *parser)
{
    while (true) {
        parser->prev_token = parser->token;
        parser->token = parser->next_token;
        parser->next_token = lexer_next(&parser->_lexer);
        if (parser->token.type != TOKEN_ERROR)
            break;
        PARSER_REPORT_SYNTAX_ERRORF(parser, parser->token, "%s", parser->token.error_msg);
    }
}

// Initialises `parser` for parsing `source`.
// `source` must remain valid until `parser_deinit()` is called.
static void parser_init(struct parser *parser, const char *source)
{
    parser->_source = source;
    lexer_init(&parser->_lexer, source);
    // Set up tokens so that `parser_advance()` doesn't read and assign any uninitialised
    // values
    parser->prev_token = (struct token){ .type = TOKEN_ERROR };
    parser->token = (struct token){ .type = TOKEN_ERROR };
    parser->next_token = lexer_next(&parser->_lexer);
    parser_advance(parser); // Populate current token
    parser->in_panic_mode = false;
    parser->had_error = false;
}

// Frees the memory associated with `parser`.
static void parser_deinit(struct parser *parser)
{
    lexer_deinit(&parser->_lexer);
}

// Calls `parser_match()` with variadic arguments.
#define PARSER_MATCH(parser, ...)                              \
    parser_match((parser), (enum token_type[]){ __VA_ARGS__ }, \
                 sizeof((enum token_type[]){ __VA_ARGS__ }) / sizeof(enum token_type))

// Reports whether the current token is one of the `count` given `types` and advances the parser if
// so. Use the `PARSER_MATCH` macro to call this function with variadic arguments.
static bool parser_match(struct parser *parser, enum token_type *types, int count)
{
    for (int i = 0; i < count; i++)
        if (parser->token.type == types[i]) {
            parser_advance(parser);
            return true;
        }
    return false;
}

// Advances the parser until the current token is at the start of a statement and exits panic mode.
static void parser_sync(struct parser *parser)
{
    while (true) {
        switch (parser->token.type) {
        case TOKEN_EOF:
        case TOKEN_PRINT:
        case TOKEN_VAR:
        case TOKEN_FINAL:
        case TOKEN_IF:
        case TOKEN_WHILE:
        case TOKEN_FOR:
        case TOKEN_BREAK:
        case TOKEN_CONTINUE:
        case TOKEN_RETURN:
            // case TOKEN_CLASS:
            return;
        default:
            parser_advance(parser);
            if (parser->token.type == TOKEN_SEMICOLON)
                return;
            break;
        }
    }
    parser->in_panic_mode = false;
}

// Works like `parser_expect()`, except it accepts a format string and arguments for the reported
// error's message.
static bool parser_expectf(struct parser *parser, enum token_type type, const char *format, ...)
{
    bool result = true;
    if (!PARSER_MATCH(parser, type)) {
        va_list args;
        va_start(args, format);
        parser_report_syntax_errorvf(parser, parser->token.start,
                                     parser->token.start + parser->token.len, format, args);
        va_end(args);
        result = false;
    }
    return result;
}

// Checks whether the current token has `type`. If it does, the parser is advanced. Otherwise, an
// error is reported.
// Reports whether the token had `type`.
static bool parser_expect(struct parser *parser, enum token_type type)
{
    return parser_expectf(parser, type, "expected '%s'", token_type_string(type));
}

// Compiler that parses Lox source code and writes the compiled bytecode into a caller provided
// chunk.
// Must be initialised with `compiler_init()` before use and deinitialised with `compiler_deinit()`
// after use.
typedef struct compiler {
    // Internal fields below, do not use.
    vm *_vm; // VM that will execute the output bytecode
    struct parser *_parser;
    struct bytecode_chunk *_out; // Chunk that bytecode is written to
    enum opcode _last_opcode; // Last opcode emitted
    enum compiler_mode _mode;
    // Scope depth of the current token. 0 is global scope, 1 is inside a block at global scope, 2
    // is inside a block in that block, etc.
    int _scope_depth;
    // Map from constant values to their index in the bytecode chunk's constants
    map _constant_indexes;
    // Array of local variables in scope at the current token's position, ordered by the position of
    // their declaration in the source
    struct local *_locals;
    // Number of local variables in scope at the current token's position
    stack_depth _locals_count;
    // Number of elements space has been allocated for in the locals array
    stack_depth _locals_cap;
    // Number of elements that would be in the value stack if execution reached the current token's
    // position
    stack_depth _stack_depth;
    bool _in_loop; // Whether a loop is currently being compiled
    struct loop_state _loop_state; // State of the loop currently being compiled
} compiler;

static struct loop_state compiler_begin_loop(compiler *);
static void compiler_compile_block(compiler *compiler);
static void compiler_compile_expr(compiler *);
static void compiler_compile_expr_stmt(compiler *);
static void compiler_compile_stmt(compiler *);
static void compiler_end_loop(compiler *, struct loop_state);
static void compiler_finish_compiling_break_stmts(compiler *);
static void compiler_finish_compiling_continue_stmts(compiler *);

// Initialises `compiler` to compile using `parser` and write the compiled bytecode into `out`.
// `parser` and `out` must remain valid until `compiler_deinit()` is called.
static void compiler_init(compiler *compiler, vm *vm, struct parser *parser,
                          struct bytecode_chunk *out)
{
    compiler->_vm = vm;
    compiler->_parser = parser;
    compiler->_out = out;
    compiler->_last_opcode = -1;
    compiler->_mode = COMPILER_MODE_SCRIPT;
    compiler->_scope_depth = 0;
    map_init(&compiler->_constant_indexes);
    compiler->_locals = NULL;
    compiler->_locals_count = 0;
    compiler->_locals_cap = 0;
    compiler->_stack_depth = 0;
    compiler->_in_loop = false;
    compiler->_loop_state = (struct loop_state){ 0 };
}

// Frees the memory associated with `compiler`.
static void compiler_deinit(compiler *compiler)
{
    map_deinit(compiler->_vm, &compiler->_constant_indexes);
    free(compiler->_locals);
    if (compiler->_in_loop) {
        free(compiler->_loop_state.break_stmts);
        free(compiler->_loop_state.continue_stmts);
    }
}

// Returns the offset which will be written to next in the output bytecode chunk.
static bytecode_offset compiler_bytecode_offset(const compiler *compiler)
{
    return compiler->_out->len;
}

// Writes `opcode` corresponding to `line` to the output bytecode chunk.
static void compiler_emit_opcode(compiler *compiler, enum opcode opcode, line_number line)
{
    bytecode_chunk_write(compiler->_vm, compiler->_out, opcode, line);
    compiler->_last_opcode = opcode;
    switch (opcode) {
    case OP_CONSTANT:
    case OP_NIL:
    case OP_TRUE:
    case OP_FALSE:
    case OP_DUP:
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
    case OP_RETURN:
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
    case OP_CALL:
        // This reduces the stack depth by n but we don't know what n is until the operand has been
        // written, so we reduce the stack size in `compile_compile_call_expr()` instead.
        break;
    }
    if (compiler->_stack_depth > compiler->_out->max_stack_depth)
        compiler->_out->max_stack_depth = compiler->_stack_depth;
}

// Writes `operand` corresponding to `line` to the output bytecode chunk.
static void compiler_emit_operand(const compiler *compiler, uint8_t operand, line_number line)
{
    bytecode_chunk_write(compiler->_vm, compiler->_out, operand, line);
}

// Ensures that value `v` corresponding to `line` is in the output bytecode chunk's constants,
// encodes the corresponding index, and writes it to the chunk.
static void compiler_emit_constant_index(compiler *compiler, value v, line_number line)
{
    value index_value;
    constant_index index;
    if (map_get(&compiler->_constant_indexes, v, &index_value)) {
        index = value_as_number(index_value);
    } else {
        index = bytecode_chunk_add_constant(compiler->_vm, compiler->_out, v);
        if (index > MAX_CONSTANT_INDEX)
            exitf("maximum number of constants (%d) used", MAX_CONSTANT_INDEX + 1);
        map_set(compiler->_vm, &compiler->_constant_indexes, v, number_value(index));
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
static void compiler_emit_stack_index(const compiler *compiler, stack_index index, line_number line)
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
static bytecode_offset compiler_emit_forward_jump(compiler *compiler, enum opcode opcode,
                                                  line_number line)
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
static void compiler_emit_pops(compiler *compiler, pop_count count, line_number line)
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
        // to worry about updating the output bytecode's max stack depth because this instruction
        // will always decrease the current stack depth.
        compiler->_stack_depth -= count;
    }
}

// Sets the operand of the forward jump instruction at offset `target` in the output bytecode chunk
// so that it jumps to the next offset written to. I.e. if this function is called just before
// another which emits some bytecode, the jump instruction will jump to that emitted bytecode.
static void compiler_set_forward_jump_offset(const compiler *compiler, bytecode_offset target)
{
    enum opcode opcode = compiler->_out->instructions[target];
    assert(is_forward_jump_opcode(opcode));
    // `target` is the offset of the jump opcode which is then followed by 2 bytes for the operand.
    // The jump offset counts from the offset after the operand bytes we need to take this away from
    // the result (1 for the opcode, 2 for the operand).
    uint64_t jump_offset = compiler_bytecode_offset(compiler) - target - 3;
    if (jump_offset > MAX_JUMP_OFFSET)
        exitf("maximum jump offset (%llu) exceeded", (uint64_t)MAX_JUMP_OFFSET);
    compiler->_out->instructions[target + 1] = (jump_offset >> 8) & 0xFF;
    compiler->_out->instructions[target + 2] = (jump_offset >> 0) & 0xFF;
}

// Writes a backward jump instruction corresponding to `line` to the output bytecode chunk that
// jumps back to `target`.
static void compiler_emit_backward_jump(compiler *compiler, bytecode_offset target,
                                        line_number line)
{
    compiler_emit_opcode(compiler, OP_JUMP_BACK, line);
    // The jump offset counts from the offset after the 2 operand bytes which we haven't written yet
    // so we need we need to add this to the result.
    uint64_t jump_offset = compiler_bytecode_offset(compiler) - target + 2;
    if (jump_offset > MAX_JUMP_OFFSET)
        exitf("maximum jump offset (%llu) exceeded", (uint64_t)MAX_JUMP_OFFSET);
    compiler_emit_operand(compiler, (jump_offset >> 8) & 0xFF, line);
    compiler_emit_operand(compiler, (jump_offset >> 0) & 0xFF, line);
}

// Signals that a new lexical scope has been entered.
static void compiler_begin_scope(compiler *compiler)
{
    compiler->_scope_depth++;
}

// Signals that a lexical scope has been exited on `line`.
static void compiler_end_scope(compiler *compiler, line_number line)
{
    size_t start_locals_count = compiler->_locals_count;
    for (stack_depth i = compiler->_locals_count - 1;
         i >= 0 && compiler->_locals[i].scope_depth == compiler->_scope_depth; i--) {
        compiler->_locals_count--;
    }
    size_t locals_in_scope = start_locals_count - compiler->_locals_count;
    compiler_emit_pops(compiler, locals_in_scope, line);
    compiler->_scope_depth--;
}

// Reports whether the string `s` of `len` characters is the blank identifier `_`.
static bool is_blank_ident(const char *s, size_t len)
{
    return len == 1 && s[0] == '_';
}

// Reports whether the current token is in the global scope.
static bool in_global_scope(const compiler *compiler)
{
    return compiler->_scope_depth == 0 && compiler->_mode == COMPILER_MODE_SCRIPT;
}

// Declares the variable corresponding to `ident`.
// A declared local variable can later be resolved to its index in the value stack with
// `compiler_resolve_local_variable()`.
static void compiler_declare_variable(compiler *compiler, struct token ident)
{
    if (in_global_scope(compiler))
        return;
    for (stack_depth i = compiler->_locals_count - 1;
         i >= 0 && compiler->_locals[i].scope_depth == compiler->_scope_depth; i--) {
        struct local local = compiler->_locals[i];
        if (local.name_len == ident.len && memcmp(local.name, ident.start, local.name_len) == 0 &&
            !is_blank_ident(local.name, local.name_len))
            PARSER_REPORT_ERRORF(compiler->_parser, ident, "'%.*s' has already been declared",
                                 (int)ident.len, ident.start);
    }
    stack_index index = compiler->_locals_count;
    if (index > MAX_STACK_INDEX)
        exitf("maximum number of local variables (%d) used", MAX_STACK_INDEX + 1);
    DYNAMIC_ARRAY_GROW(compiler->_locals, compiler->_locals_cap, compiler->_locals_count + 1);
    struct local *local = &compiler->_locals[compiler->_locals_count++];
    local->name = ident.start;
    local->name_len = ident.len;
    local->scope_depth = compiler->_scope_depth;
    local->is_initialising = false;
}

// Writes the instruction to define the variable corresponding to `ident` to the output bytecode
// chunk.
// Must be called straight after the bytecode pushing the initialiser onto the value stack has been
// emitted.
static void compiler_emit_variable_definition(compiler *compiler, struct token ident, bool is_final)
{
    if (in_global_scope(compiler)) {
        enum opcode instruction = is_final ? OP_DEFINE_GLOBAL_FINAL : OP_DEFINE_GLOBAL;
        compiler_emit_opcode(compiler, instruction, ident.line);
        struct object_string *name_string =
            object_string_copy(compiler->_vm, ident.start, ident.len);
        compiler_emit_constant_index(compiler, string_value(name_string), ident.line);
    } else if (is_final) {
        compiler_emit_opcode(compiler, OP_DEFINE_LOCAL_FINAL, ident.line);
        struct object_string *name_string =
            object_string_copy(compiler->_vm, ident.start, ident.len);
        compiler_emit_constant_index(compiler, string_value(name_string), ident.line);
    } else {
        compiler_emit_opcode(compiler, OP_DEFINE_LOCAL, ident.line);
    }
}

// Returns the index in the value stack of the local variable that `ident` refers to or `-1` if one
// is not found.
static stack_index compiler_resolve_local_variable(const compiler *compiler, struct token ident)
{
    for (stack_depth i = compiler->_locals_count - 1; i >= 0; i--) {
        struct local local = compiler->_locals[i];
        if (local.name_len == ident.len && memcmp(local.name, ident.start, local.name_len) == 0) {
            if (local.is_initialising)
                PARSER_REPORT_ERRORF(compiler->_parser, ident, "'%.*s' read in its own initialiser",
                                     (int)ident.len, ident.start);
            return i;
        }
    }
    return -1;
}

// Marks the variable corresponding to `ident` as initialising. An error is reported if a local
// variable is resolved while it's initialising.
static void compiler_start_initialising_variable(const compiler *compiler, struct token ident)
{
    for (stack_depth i = compiler->_locals_count - 1;
         i >= 0 && compiler->_locals[i].scope_depth == compiler->_scope_depth; i--) {
        struct local local = compiler->_locals[i];
        if (local.name_len == ident.len && memcmp(local.name, ident.start, local.name_len) == 0)
            compiler->_locals[i].is_initialising = true;
    }
}

// Marks the variable corresponding to `ident` as initialised.
static void compiler_finish_initialising_variable(const compiler *compiler, struct token ident)
{
    for (stack_depth i = compiler->_locals_count - 1;
         i >= 0 && compiler->_locals[i].scope_depth == compiler->_scope_depth; i--) {
        struct local local = compiler->_locals[i];
        if (local.name_len == ident.len && memcmp(local.name, ident.start, local.name_len) == 0)
            compiler->_locals[i].is_initialising = false;
    }
}

// Emits the bytecode for the loop jump statement starting at the current `type` token and advances
// the compiler past it.
// A loop jump statement can't be compiled completely until the point in the loop that the statement
// should jump past has been compiled. Once this has been done, pass the returned `struct
// loop_jump_statement` to `compiler_finish_compiling_loop_jump_stmt()`.
static struct loop_jump_statement compiler_compile_loop_jump_stmt(compiler *compiler,
                                                                  enum token_type type)
{
    assert(type == TOKEN_BREAK || type == TOKEN_CONTINUE);
    if (!compiler->_in_loop)
        PARSER_REPORT_ERRORF(compiler->_parser, compiler->_parser->token,
                             "'%s' can only be used inside a loop", token_type_string(type));
    line_number line = compiler->_parser->token.line;
    parser_expect(compiler->_parser, type);
    parser_expect(compiler->_parser, TOKEN_SEMICOLON);
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
    size_t locals_declared_before_stmt = stmt.locals_count - compiler->_locals_count;
    compiler_emit_pops(compiler, locals_declared_before_stmt, stmt.line);
}

// Type of function being compiled.
enum function_type {
    FUNCTION_TYPE_FUNCTION = COMPILER_MODE_FUNCTION, // Regular function
};

// Compiles and returns the function whose parameters (including opening parenthesis) start at the
// current token. Returns `NULL` if there was an error.
// The first `name_len` bytes of `name` are copied to populate the function's name.
static struct object_function *compiler_compile_fun(compiler *compiler, enum function_type type,
                                                    const char *name, size_t name_len)
{
    struct bytecode_chunk body;
    bytecode_chunk_init(&body, true);
    struct compiler body_compiler;
    compiler_init(&body_compiler, compiler->_vm, compiler->_parser, &body);
    // All function_type members are equal to their equivalent compiler_mode
    body_compiler._mode = (enum compiler_mode)type;

    argument_count arity = 0;
    parser_expect(body_compiler._parser, TOKEN_LEFT_PAREN);
    if (!PARSER_MATCH(body_compiler._parser, TOKEN_RIGHT_PAREN)) {
        do {
            arity++;
            struct token param = body_compiler._parser->token;
            if (!parser_expectf(body_compiler._parser, TOKEN_IDENT, "expected parameter name"))
                goto out_body_compiler_deinit;
            compiler_declare_variable(&body_compiler, param);
        } while (PARSER_MATCH(body_compiler._parser, TOKEN_COMMA));
        parser_expect(body_compiler._parser, TOKEN_RIGHT_PAREN);
    }
    compiler_compile_block(&body_compiler);
    if (body_compiler._last_opcode != OP_RETURN) {
        compiler_emit_opcode(&body_compiler, OP_NIL, body_compiler._parser->prev_token.line);
        compiler_emit_opcode(&body_compiler, OP_RETURN, body_compiler._parser->prev_token.line);
    }

    struct object_function *function =
        object_function_new(compiler->_vm, name, name_len, arity, body);
#ifdef DEBUG_PRINT_BYTECODE
    disassemble(&function->body, function->name->data);
#endif
    return function;

out_body_compiler_deinit:
    compiler_deinit(&body_compiler);
    return NULL;
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
    case TOKEN_LEFT_PAREN:
        // case TOKEN_LEFT_BRACK:
        // case TOKEN_DOT:
        return PREC_POSTFIX;
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

// Type of function that behaves in the same way as `expr_compiler` except it compiles the infix
// expression whose first operand has already been compiled. That is, it expects the current token
// to be the infix operator.
typedef void (*infix_expr_compiler)(compiler *compiler, enum precedence_level level);

// `expr_compiler` for a keyword literal expression.
static void compiler_compile_keyword_literal_expr(compiler *compiler, enum precedence_level level)
{
    (void)level;
    enum opcode instruction;
    if (PARSER_MATCH(compiler->_parser, TOKEN_NIL)) {
        instruction = OP_NIL;
    } else if (PARSER_MATCH(compiler->_parser, TOKEN_TRUE)) {
        instruction = OP_TRUE;
    } else if (PARSER_MATCH(compiler->_parser, TOKEN_FALSE)) {
        instruction = OP_FALSE;
    } else {
        PARSER_REPORT_SYNTAX_ERRORF(compiler->_parser, compiler->_parser->token, "%s",
                                    "expected literal keyword");
        return;
    }
    compiler_emit_opcode(compiler, instruction, compiler->_parser->prev_token.line);
}

// `expr_compiler` for a function expression.
static void compiler_compile_fun_expr(compiler *compiler, enum precedence_level level)
{
    (void)level;
    line_number line = compiler->_parser->token.line;
    parser_expect(compiler->_parser, TOKEN_FUN);
    const char name[] = "(anonymous)";
    struct object_function *function =
        compiler_compile_fun(compiler, FUNCTION_TYPE_FUNCTION, name, sizeof(name));
    if (function == NULL)
        return;
    compiler_emit_opcode(compiler, OP_CONSTANT, line);
    compiler_emit_constant_index(compiler, function_value(function), line);
}

// `expr_compiler` for an identifier expression.
static void compiler_compile_ident_expr(compiler *compiler, enum precedence_level level)
{
    line_number line = compiler->_parser->token.line;
    struct token name = compiler->_parser->token;
    if (!parser_expectf(compiler->_parser, TOKEN_IDENT, "expected identifier"))
        return;

    bool is_get = true;
    if (compiler->_parser->token.type == TOKEN_EQUAL) {
        if (level > PREC_ASSIGNMENT) {
            PARSER_REPORT_SYNTAX_ERRORF(compiler->_parser, compiler->_parser->token, "%s",
                                        "invalid assignment target");
            return;
        }
        parser_advance(compiler->_parser);
        compiler_compile_expr_at_level(compiler, PREC_ASSIGNMENT); // Initialiser
        is_get = false;
    }

    if (is_blank_ident(name.start, name.len)) {
        if (is_get)
            PARSER_REPORT_SYNTAX_ERRORF(compiler->_parser, name, "%s",
                                        "'_' cannot be used as a value");
        else
            return;
    }

    stack_index index = compiler_resolve_local_variable(compiler, name);
    if (index >= 0) {
        enum opcode instruction = is_get ? OP_GET_LOCAL : OP_SET_LOCAL;
        compiler_emit_opcode(compiler, instruction, line);
        compiler_emit_stack_index(compiler, index, line);
    } else {
        enum opcode instruction = is_get ? OP_GET_GLOBAL : OP_SET_GLOBAL;
        compiler_emit_opcode(compiler, instruction, line);
        struct object_string *name_string = object_string_copy(compiler->_vm, name.start, name.len);
        compiler_emit_constant_index(compiler, string_value(name_string), line);
    }
}

// `expr_compiler` for a string literal expression.
static void compiler_compile_string_literal_expr(compiler *compiler, enum precedence_level level)
{
    (void)level;
    parser_expect(compiler->_parser, TOKEN_STRING);
    // First and last characters are " so discount these from the length and start data 1 character
    // in
    struct token token = compiler->_parser->prev_token;
    struct object_string *string =
        object_string_copy(compiler->_vm, token.start + 1, token.len - 2);
    compiler_emit_opcode(compiler, OP_CONSTANT, compiler->_parser->prev_token.line);
    compiler_emit_constant_index(compiler, string_value(string),
                                 compiler->_parser->prev_token.line);
}

// `expr_compiler` for a number literal expression.
static void compiler_compile_number_literal_expr(compiler *compiler, enum precedence_level level)
{
    (void)level;
    parser_expect(compiler->_parser, TOKEN_NUMBER);
    double number = strtod(compiler->_parser->prev_token.start, NULL);
    compiler_emit_opcode(compiler, OP_CONSTANT, compiler->_parser->prev_token.line);
    compiler_emit_constant_index(compiler, number_value(number),
                                 compiler->_parser->prev_token.line);
}

// `expr_compiler` for a group expression.
static void compiler_compile_group_expr(compiler *compiler, enum precedence_level level)
{
    (void)level;
    parser_expect(compiler->_parser, TOKEN_LEFT_PAREN);
    compiler_compile_expr(compiler);
    parser_expect(compiler->_parser, TOKEN_RIGHT_PAREN);
}

// `expr_compiler` for a unary expression.
static void compiler_compile_unary_expr(compiler *compiler, enum precedence_level level)
{
    (void)level;
    line_number line = compiler->_parser->token.line;
    enum opcode instruction;
    if (PARSER_MATCH(compiler->_parser, TOKEN_MINUS)) {
        instruction = OP_NEGATE;
    } else if (PARSER_MATCH(compiler->_parser, TOKEN_BANG)) {
        instruction = OP_NOT;
    } else {
        PARSER_REPORT_SYNTAX_ERRORF(compiler->_parser, compiler->_parser->token, "%s",
                                    "expected unary operator");
        return;
    }
    compiler_compile_expr_at_level(compiler, PREC_UNARY);
    compiler_emit_opcode(compiler, instruction, line);
}

// Returns the compiler for an expression starting with a `type` token or `NULL` if there is none.
static expr_compiler expr_compiler_for_prefix(enum token_type type)
{
    switch (type) {
    case TOKEN_TRUE:
    case TOKEN_FALSE:
    case TOKEN_NIL:
        return compiler_compile_keyword_literal_expr;
    case TOKEN_FUN:
        return compiler_compile_fun_expr;
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
    line_number line = compiler->_parser->token.line;
    parser_expect(compiler->_parser, TOKEN_COMMA);
    compiler_compile_expr_at_level(compiler, PREC_COMMA); // Right operand
    compiler_emit_pops(compiler, 1, line); // Right operand
}

// `infix_expr_compiler` for a ternary expression.
static void compiler_compile_ternary_expr(compiler *compiler, enum precedence_level level)
{
    (void)level;
    line_number line = compiler->_parser->token.line;
    parser_expect(compiler->_parser, TOKEN_QUESTION);
    bytecode_offset else_jump = compiler_emit_forward_jump(compiler, OP_JUMP_IF_FALSE, line);
    compiler_emit_pops(compiler, 1, line); // Condition
    compiler_compile_expr(compiler); // Then value
    bytecode_offset end_jump = compiler_emit_forward_jump(compiler, OP_JUMP, line);
    parser_expect(compiler->_parser, TOKEN_COLON);
    compiler_set_forward_jump_offset(compiler, else_jump);
    compiler_emit_pops(compiler, 1, line); // Condition
    compiler_compile_expr_at_level(compiler, PREC_TERNARY); // Else value
    compiler_set_forward_jump_offset(compiler, end_jump);
}

// `infix_expr_compiler` for an or expression.
static void compiler_compile_or_expr(compiler *compiler, enum precedence_level level)
{
    line_number line = compiler->_parser->token.line;
    parser_expect(compiler->_parser, TOKEN_OR);
    bytecode_offset end_jump = compiler_emit_forward_jump(compiler, OP_JUMP_IF_TRUE, line);
    compiler_emit_pops(compiler, 1, line); // Left operand
    compiler_compile_expr_at_level(compiler, level + 1); // Right operand
    compiler_set_forward_jump_offset(compiler, end_jump);
}

// `infix_expr_compiler` for an and expression.
static void compiler_compile_and_expr(compiler *compiler, enum precedence_level level)
{
    line_number line = compiler->_parser->token.line;
    parser_expect(compiler->_parser, TOKEN_AND);
    bytecode_offset end_jump = compiler_emit_forward_jump(compiler, OP_JUMP_IF_FALSE, line);
    compiler_emit_pops(compiler, 1, line); // Left operand
    compiler_compile_expr_at_level(compiler, level + 1); // Right operand
    compiler_set_forward_jump_offset(compiler, end_jump);
}

// `infix_expr_compiler` for a binary expression.
static void compiler_compile_binary_expr(compiler *compiler, enum precedence_level level)
{
    line_number line = compiler->_parser->token.line;
    enum opcode instruction;
    if (PARSER_MATCH(compiler->_parser, TOKEN_PLUS)) {
        instruction = OP_ADD;
    } else if (PARSER_MATCH(compiler->_parser, TOKEN_MINUS)) {
        instruction = OP_SUBTRACT;
    } else if (PARSER_MATCH(compiler->_parser, TOKEN_ASTERISK)) {
        instruction = OP_MULTIPLY;
    } else if (PARSER_MATCH(compiler->_parser, TOKEN_SLASH)) {
        instruction = OP_DIVIDE;
    } else if (PARSER_MATCH(compiler->_parser, TOKEN_PERCENT)) {
        instruction = OP_MODULO;
    } else if (PARSER_MATCH(compiler->_parser, TOKEN_LESS)) {
        instruction = OP_LESS;
    } else if (PARSER_MATCH(compiler->_parser, TOKEN_LESS_EQUAL)) {
        instruction = OP_LESS_EQUAL;
    } else if (PARSER_MATCH(compiler->_parser, TOKEN_GREATER)) {
        instruction = OP_GREATER;
    } else if (PARSER_MATCH(compiler->_parser, TOKEN_GREATER_EQUAL)) {
        instruction = OP_GREATER_EQUAL;
    } else if (PARSER_MATCH(compiler->_parser, TOKEN_EQUAL_EQUAL)) {
        instruction = OP_EQUAL;
    } else if (PARSER_MATCH(compiler->_parser, TOKEN_BANG_EQUAL)) {
        instruction = OP_NOT_EQUAL;
    } else {
        PARSER_REPORT_SYNTAX_ERRORF(compiler->_parser, compiler->_parser->token, "%s",
                                    "expected binary operator");
        return;
    }
    compiler_compile_expr_at_level(compiler, level + 1); // Right operand
    compiler_emit_opcode(compiler, instruction, line);
}

// `infix_expr_compiler` for a call expression.
static void compiler_compile_call_expr(compiler *compiler, enum precedence_level level)
{
    (void)level;
    line_number line = compiler->_parser->token.line;
    parser_expect(compiler->_parser, TOKEN_LEFT_PAREN);

    argument_count args = 0;
    if (!PARSER_MATCH(compiler->_parser, TOKEN_RIGHT_PAREN)) {
        do {
            args++;
            compiler_compile_expr_at_level(compiler, PREC_ASSIGNMENT);
        } while (PARSER_MATCH(compiler->_parser, TOKEN_COMMA));
        parser_expect(compiler->_parser, TOKEN_RIGHT_PAREN);
    }

    compiler_emit_opcode(compiler, OP_CALL, line);
    if (args < 255) {
        compiler_emit_operand(compiler, args, line);
    } else {
        compiler_emit_operand(compiler, 0xFF, line);
        compiler_emit_operand(compiler, (args >> 16) & 0xFF, line);
        compiler_emit_operand(compiler, (args >> 8) & 0xFF, line);
        compiler_emit_operand(compiler, (args >> 0) & 0xFF, line);
    }
    // This is decremented in `compiler_emit_opcode()` for all other instructions. We need to
    // handle this one here because the effect on the stack depth is not static. We don't need
    // to worry about updating the output bytecode's max stack depth because this instruction will
    // always decrease the current stack depth.
    compiler->_stack_depth -= args;
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
    case TOKEN_LEFT_PAREN:
        return compiler_compile_call_expr;
    default:
        return NULL;
    }
}

// `expr_compiler` for any expression.
static void compiler_compile_expr_at_level(compiler *compiler, enum precedence_level level)
{
    expr_compiler prefix_compiler = expr_compiler_for_prefix(compiler->_parser->token.type);
    if (prefix_compiler == NULL) {
        PARSER_REPORT_SYNTAX_ERRORF(compiler->_parser, compiler->_parser->token, "%s",
                                    "expected expression");
        return;
    }
    prefix_compiler(compiler, level);

    enum precedence_level infix_level;
    while ((infix_level = infix_operator_precedence_level(compiler->_parser->token.type)) >=
           level) {
        infix_expr_compiler infix_compiler =
            infix_expr_compiler_for_operator(compiler->_parser->token.type);
        infix_compiler(compiler, infix_level);
    }
}

// Emits the bytecode for the expression starting the current token and advances the compiler past
// it.
static void compiler_compile_expr(compiler *compiler)
{
    compiler_compile_expr_at_level(compiler, PREC_COMMA);
}

// Emits the bytecode for the variable declaration (using either var or final) starting at the
// current token and advances the compiler past it.
static void compiler_compile_var_decl(compiler *compiler)
{
    line_number line = compiler->_parser->token.line;
    bool is_final = false;
    if (PARSER_MATCH(compiler->_parser, TOKEN_FINAL))
        is_final = true;
    else
        parser_expect(compiler->_parser, TOKEN_VAR);

    struct token name = compiler->_parser->token;
    if (!parser_expectf(compiler->_parser, TOKEN_IDENT, "expected variable name"))
        return;
    compiler_declare_variable(compiler, name);

    compiler_start_initialising_variable(compiler, name);
    if (PARSER_MATCH(compiler->_parser, TOKEN_EQUAL))
        compiler_compile_expr(compiler);
    else
        compiler_emit_opcode(compiler, OP_NIL, line);
    parser_expect(compiler->_parser, TOKEN_SEMICOLON);
    compiler_finish_initialising_variable(compiler, name);

    if (is_blank_ident(name.start, name.len))
        return;

    compiler_emit_variable_definition(compiler, name, is_final);
}

// Emits the bytecode for the function declaration starting at the current token and advances the
// compiler past it.
static void compiler_compile_fun_decl(compiler *compiler)
{
    line_number line = compiler->_parser->token.line;
    parser_expect(compiler->_parser, TOKEN_FUN);
    struct token name = compiler->_parser->token;
    if (!parser_expectf(compiler->_parser, TOKEN_IDENT, "expected function name"))
        return;
    compiler_declare_variable(compiler, name);
    struct object_function *function =
        compiler_compile_fun(compiler, FUNCTION_TYPE_FUNCTION, name.start, name.len);
    if (function == NULL)
        return;
    compiler_emit_opcode(compiler, OP_CONSTANT, line);
    compiler_emit_constant_index(compiler, function_value(function), line);
    bool is_final = false;
    compiler_emit_variable_definition(compiler, name, is_final);
}

// Emits the bytecode for the declaration starting at the current token and advances the compiler
// past it.
static void compiler_compile_decl(compiler *compiler)
{
    switch (compiler->_parser->token.type) {
    case TOKEN_VAR:
    case TOKEN_FINAL:
        compiler_compile_var_decl(compiler);
        break;
    case TOKEN_FUN:
        if (compiler->_parser->next_token.type == TOKEN_IDENT) {
            compiler_compile_fun_decl(compiler);
            break;
        }
    default:
        compiler_compile_stmt(compiler);
        break;
    }
    if (compiler->_parser->in_panic_mode) {
        parser_sync(compiler->_parser);
    }
}

// Emits the bytecode for the block starting at the current token and advances the compiler past it.
// If the declarations inside the block should be defined in a new scope, then
// `compiler_begin_scope()` and `compiler_end_scope()` must be called before and after this.
static void compiler_compile_block(compiler *compiler)
{
    parser_expect(compiler->_parser, TOKEN_LEFT_BRACE);
    while (compiler->_parser->token.type != TOKEN_RIGHT_BRACE &&
           compiler->_parser->token.type != TOKEN_EOF)
        compiler_compile_decl(compiler);
    parser_expect(compiler->_parser, TOKEN_RIGHT_BRACE);
}

// Emits the bytecode for the print statement starting at the current token and advances the
// compiler past it.
static void compiler_compile_print_stmt(compiler *compiler)
{
    line_number line = compiler->_parser->token.line;
    parser_expect(compiler->_parser, TOKEN_PRINT);
    compiler_compile_expr(compiler);
    parser_expect(compiler->_parser, TOKEN_SEMICOLON);
    compiler_emit_opcode(compiler, OP_PRINT, line);
}

// Emits the bytecode for the if statement starting at the current token and advances the compiler
// past it.
static void compiler_compile_if_stmt(compiler *compiler)
{
    int if_line = compiler->_parser->token.line;
    parser_expect(compiler->_parser, TOKEN_IF);

    // Condition
    parser_expect(compiler->_parser, TOKEN_LEFT_PAREN);
    int condition_line = compiler->_parser->token.line;
    compiler_compile_expr(compiler); // Condition
    parser_expect(compiler->_parser, TOKEN_RIGHT_PAREN);
    bytecode_offset else_jump = compiler_emit_forward_jump(compiler, OP_JUMP_IF_FALSE, if_line);

    // Then branch
    compiler_emit_pops(compiler, 1, condition_line); // Condition
    compiler_compile_stmt(compiler); // Then body
    bytecode_offset end_jump = compiler_emit_forward_jump(compiler, OP_JUMP, if_line);

    // Else branch
    compiler_set_forward_jump_offset(compiler, else_jump);
    compiler_emit_pops(compiler, 1, condition_line); // Condition
    if (PARSER_MATCH(compiler->_parser, TOKEN_ELSE)) {
        compiler_compile_stmt(compiler); // Else body
    }
    compiler_set_forward_jump_offset(compiler, end_jump);
}

// Emits the bytecode for the switch statement starting at the current token and advances the
// compiler past it.
static void compiler_compile_switch_stmt(compiler *compiler)
{
    int switch_line = compiler->_parser->token.line;
    parser_expect(compiler->_parser, TOKEN_SWITCH);

    // Switch value
    parser_expect(compiler->_parser, TOKEN_LEFT_PAREN);
    compiler_compile_expr(compiler); // Switch value
    parser_expect(compiler->_parser, TOKEN_RIGHT_PAREN);
    parser_expect(compiler->_parser, TOKEN_LEFT_BRACE);

    bytecode_offset *end_jumps = NULL;
    size_t end_jumps_len = 0;
    size_t end_jumps_cap = 0;

    // Non-default cases
    while (PARSER_MATCH(compiler->_parser, TOKEN_CASE)) {
        line_number line = compiler->_parser->prev_token.line;

        // Compare switch value against case value
        compiler_emit_opcode(compiler, OP_DUP, line); // Switch value
        compiler_compile_expr(compiler); // Case value
        parser_expect(compiler->_parser, TOKEN_COLON);
        compiler_emit_opcode(compiler, OP_EQUAL, line);
        bytecode_offset next_case_jump =
            compiler_emit_forward_jump(compiler, OP_JUMP_IF_FALSE, line);

        // Case body
        compiler_emit_pops(compiler, 2, line); // EQUAL result and switch value
        while (compiler->_parser->token.type != TOKEN_CASE &&
               compiler->_parser->token.type != TOKEN_DEFAULT &&
               compiler->_parser->token.type != TOKEN_RIGHT_BRACE &&
               compiler->_parser->token.type != TOKEN_EOF)
            compiler_compile_stmt(compiler); // Case body
        bytecode_offset end_jump = compiler_emit_forward_jump(compiler, OP_JUMP, line);
        DYNAMIC_ARRAY_GROW(end_jumps, end_jumps_cap, end_jumps_len + 1);
        end_jumps[end_jumps_len++] = end_jump;

        // Jump to next case
        compiler_set_forward_jump_offset(compiler, next_case_jump);
        compiler_emit_pops(compiler, 1, line); // EQUAL result
    }

    compiler_emit_pops(compiler, 1, switch_line); // Switch value

    // Default case
    if (PARSER_MATCH(compiler->_parser, TOKEN_DEFAULT)) {
        parser_expect(compiler->_parser, TOKEN_COLON);
        while (compiler->_parser->token.type != TOKEN_RIGHT_BRACE)
            compiler_compile_stmt(compiler); // Default body
    }

    parser_expect(compiler->_parser, TOKEN_RIGHT_BRACE);

    for (size_t i = 0; i < end_jumps_len; i++)
        compiler_set_forward_jump_offset(compiler, end_jumps[i]);
    free(end_jumps);
}

// Emits the bytecode for the while statement starting at the current token and advances the
// compiler past it.
static void compiler_compile_while_stmt(compiler *compiler)
{
    struct loop_state prev_loop_state = compiler_begin_loop(compiler);

    bytecode_offset loop_start = compiler_bytecode_offset(compiler);
    int while_line = compiler->_parser->token.line;
    parser_expect(compiler->_parser, TOKEN_WHILE);

    // Condition
    parser_expect(compiler->_parser, TOKEN_LEFT_PAREN);
    int condition_line = compiler->_parser->token.line;
    compiler_compile_expr(compiler); // Condition
    parser_expect(compiler->_parser, TOKEN_RIGHT_PAREN);
    bytecode_offset end_jump = compiler_emit_forward_jump(compiler, OP_JUMP_IF_FALSE, while_line);

    // Body
    compiler_emit_pops(compiler, 1, condition_line); // Condition
    compiler_compile_stmt(compiler); // Body
    compiler_finish_compiling_continue_stmts(compiler);
    compiler_emit_backward_jump(compiler, loop_start, while_line);
    compiler_set_forward_jump_offset(compiler, end_jump);
    compiler_emit_pops(compiler, 1, condition_line); // Condition
    compiler_finish_compiling_break_stmts(compiler);

    compiler_end_loop(compiler, prev_loop_state);
}

// Emits the bytecode for the for statement starting at the current token and advances the
// compiler past it.
static void compiler_compile_for_stmt(compiler *compiler)
{
    struct loop_state prev_loop_state = compiler_begin_loop(compiler);
    compiler_begin_scope(compiler);

    int for_line = compiler->_parser->token.line;
    parser_expect(compiler->_parser, TOKEN_FOR);
    parser_expect(compiler->_parser, TOKEN_LEFT_PAREN);

    // Initialiser
    if (PARSER_MATCH(compiler->_parser, TOKEN_SEMICOLON))
        ;
    else if (compiler->_parser->token.type == TOKEN_VAR)
        compiler_compile_var_decl(compiler);
    else
        compiler_compile_expr_stmt(compiler);

    // Condition
    bool has_condition = !PARSER_MATCH(compiler->_parser, TOKEN_SEMICOLON);
    bytecode_offset condition_start = compiler_bytecode_offset(compiler);
    int condition_line = compiler->_parser->token.line;
    bytecode_offset end_jump;
    if (has_condition) {
        compiler_compile_expr(compiler); // Condition
        parser_expect(compiler->_parser, TOKEN_SEMICOLON);
        end_jump = compiler_emit_forward_jump(compiler, OP_JUMP_IF_FALSE, for_line);
        compiler_emit_pops(compiler, 1, condition_line); // Condition
    }

    // Increment expression
    bool has_increment = !PARSER_MATCH(compiler->_parser, TOKEN_RIGHT_PAREN);
    bytecode_offset increment_start;
    int increment_line = compiler->_parser->token.line;
    bytecode_offset increment_body_jump;
    if (has_increment) {
        increment_body_jump = compiler_emit_forward_jump(compiler, OP_JUMP, for_line);
        increment_start = compiler_bytecode_offset(compiler);
        compiler_compile_expr(compiler); // Increment expression
        parser_expect(compiler->_parser, TOKEN_RIGHT_PAREN);
        compiler_emit_pops(compiler, 1, increment_line); // Increment expression value
        if (has_condition)
            compiler_emit_backward_jump(compiler, condition_start, for_line);
    }

    // Body
    if (has_increment)
        compiler_set_forward_jump_offset(compiler, increment_body_jump);
    compiler_compile_stmt(compiler); // Body
    compiler_finish_compiling_continue_stmts(compiler);
    bytecode_offset body_jump_back_target = has_increment ? increment_start : condition_start;
    compiler_emit_backward_jump(compiler, body_jump_back_target, for_line);
    if (has_condition) {
        compiler_set_forward_jump_offset(compiler, end_jump);
        compiler_emit_pops(compiler, 1, condition_line); // Condition
    }
    compiler_finish_compiling_break_stmts(compiler);

    compiler_end_scope(compiler, compiler->_parser->prev_token.line);
    compiler_end_loop(compiler, prev_loop_state);
}

// Signals that compilation of a loop has started.
// Must be called before any bytecode for the loop is emitted.
// Returns the state of the loop previously being compiled which must be passed to
// `compiler_end_loop()` after all bytecode for the loop has been emitted.
static struct loop_state compiler_begin_loop(compiler *compiler)
{
    struct loop_state prev_loop_state = compiler->_loop_state;
    compiler->_loop_state = (struct loop_state){ .is_nested = compiler->_in_loop };
    compiler->_in_loop = true;
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
    compiler->_in_loop = prev_loop_state.is_nested;
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

// Emits the bytecode for the return statement starting at the current token and advances the
// compiler past it.
static void compiler_compile_return_stmt(compiler *compiler)
{
    line_number line = compiler->_parser->token.line;
    parser_expect(compiler->_parser, TOKEN_RETURN);
    if (PARSER_MATCH(compiler->_parser, TOKEN_SEMICOLON)) {
        compiler_emit_opcode(compiler, OP_NIL, line);
    } else {
        compiler_compile_expr(compiler);
        parser_expect(compiler->_parser, TOKEN_SEMICOLON);
    }
    compiler_emit_opcode(compiler, OP_RETURN, line);
}

// Emits the bytecode for the expression statement starting at the current token and advances the
// compiler past it.
static void compiler_compile_expr_stmt(compiler *compiler)
{
    line_number line = compiler->_parser->token.line;
    compiler_compile_expr(compiler);
    parser_expect(compiler->_parser, TOKEN_SEMICOLON);
    compiler_emit_pops(compiler, 1, line); // Expression value
}

// Emits the bytecode for the statement starting at the current token and advances the compiler past
// it.
static void compiler_compile_stmt(compiler *compiler)
{
    switch (compiler->_parser->token.type) {
    case TOKEN_PRINT:
        compiler_compile_print_stmt(compiler);
        break;
    case TOKEN_IF:
        compiler_compile_if_stmt(compiler);
        break;
    case TOKEN_SWITCH:
        compiler_compile_switch_stmt(compiler);
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
    case TOKEN_RETURN:
        compiler_compile_return_stmt(compiler);
        break;
    case TOKEN_LEFT_BRACE:
        compiler_begin_scope(compiler);
        compiler_compile_block(compiler);
        compiler_end_scope(compiler, compiler->_parser->prev_token.line);
        break;
    default:
        compiler_compile_expr_stmt(compiler);
        break;
    }
}

// Emits the bytecode for the program and reports whether compilation was successful.
static bool compiler_compile(compiler *compiler)
{
    while (compiler->_parser->token.type != TOKEN_EOF)
        compiler_compile_decl(compiler);
    compiler_emit_opcode(compiler, OP_NIL, compiler->_parser->token.line);
    compiler_emit_opcode(compiler, OP_RETURN, compiler->_parser->token.line);
    bool success = !compiler->_parser->had_error;
#ifdef DEBUG_PRINT_BYTECODE
    if (success)
        disassemble(compiler->_out, "script");
#endif
    return success;
}

bool compile(vm *vm, const char *source, struct bytecode_chunk *out)
{
    struct parser parser;
    parser_init(&parser, source);
    compiler compiler;
    compiler_init(&compiler, vm, &parser, out);

    bool success = compiler_compile(&compiler);

    compiler_deinit(&compiler);
    parser_deinit(&parser);

    return success;
}
