#include "compiler.h"

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

// Compiler that parses Lox source code and writes the compiled bytecode into a caller provided
// chunk.
// Must be initialised with `compiler_init()` and freed with `compiler_free()` after use.
typedef struct {
    // Dependencies
    vm *_vm; // VM the source is being compiled for
    const char *_source; // Source being compiled
    lexer _lexer; // Lexer set up to lex the source
    struct bytecode_chunk *_out; // Chunk that bytecode is written to

    // Public runtime state - use these fields directly
    struct token _token; // Token currently being considered
    struct token _next_token; // Token after current token in the source
    struct token _prev_token; // Token before current token in the source
    // Scope depth of the current token. 0 is global scope, 1 is inside a block at global scope, 2
    // is inside a block in that block, etc.
    // Don't modify directly, use `compiler_begin_scope()` and `compiler_end_scope()` instead.
    int _scope_depth;
    bool _in_panic_mode; // Indicates that an error has been encountered which needs recovering from
    bool _had_error; // Whether the compiler has encountered any errors

    // Private runtime state - don't use these fields directly

    // Map from constant values to their index in the bytecode's constants.
    // Don't use directly, use `compiler_emit_constant_index()` instead.
    map _constant_indexes;
    // Array of local variables in scope at the current token's position, ordered by the position of
    // their declaration in the source.
    // Don't use directly, use `compiler_add_local()` and `compiler_resolve_local()` instead.
    struct local *_locals;
    // Number of local variables in scope at the current token's position
    // Don't use directly, use `compiler_add_local()` and `compiler_resolve_local()` instead.
    size_t _locals_count;
    // Number of elements space has been allocated for in the locals array.
    // Don't use directly.
    size_t _locals_cap;
    // Number of elements in the value stack at the current token's position.
    // This is increased or decreased when the opcode for an instruction which increases or
    // decreases the number of elements in the stack is emitted.
    // Don't use directly.
    stack_index _stack_depth;
} compiler;

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

// Writes `opcode` corresponding to `line` to the bytecode.
static void compiler_emit_opcode(compiler *compiler, enum opcode opcode, int line)
{
    bytecode_chunk_write(compiler->_out, opcode, line);
    compiler->_stack_depth += instruction_stack_depth_effect(opcode);
    if (compiler->_stack_depth > compiler->_out->max_stack_depth)
        compiler->_out->max_stack_depth = compiler->_stack_depth;
}

// Writes `opreand` corresponding to `line` to the bytecode.
static void compiler_emit_operand(compiler *compiler, uint8_t operand, int line)
{
    bytecode_chunk_write(compiler->_out, operand, line);
}

// Ensures that value `v` corresponding to `line` is in the bytecode's constants, encodes the
// corresponding index, and writes it to the bytecode.
static void compiler_emit_constant_index(compiler *compiler, value v, int line)
{
    value index_value;
    constant_index index;
    if (map_get(&compiler->_constant_indexes, v, &index_value)) {
        index = value_as_number(index_value);
    } else {
        index = bytecode_chunk_add_constant(compiler->_out, v);
        if (index > MAX_CONSTANT_INDEX)
            panicf("maximum number of constants (%lu) used", (size_t)MAX_CONSTANT_INDEX + 1);
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

// Encodes and writes `index` to the bytecode.
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
    for (ptrdiff_t i = compiler->_locals_count - 1;
         i >= 0 && compiler->_locals[i].scope_depth == compiler->_scope_depth; i--) {
        compiler_emit_opcode(compiler, OP_POP, line);
        compiler->_locals_count--;
    }
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
        if (local.name_len == ident.len && memcmp(local.name, ident.start, local.name_len) == 0)
            COMPILER_REPORT_TOKEN_ERRORF(compiler, ident, "'%.*s' has already been declared",
                                         (int)ident.len, ident.start);
    }
    stack_index index = compiler->_locals_count;
    DYNAMIC_ARRAY_GROW(compiler->_locals, compiler->_locals_cap, compiler->_locals_count + 1);
    if (index > MAX_STACK_INDEX)
        panicf("maximum number of local variables (%lu) used", (size_t)MAX_STACK_INDEX + 1);
    struct local *local = &compiler->_locals[compiler->_locals_count++];
    local->name = ident.start;
    local->name_len = ident.len;
    local->scope_depth = compiler->_scope_depth;
    local->is_initialising = true;
    return index;
}

// Marks the local variable stored at index `i` in the VM's value stack as initialised.
static void compiler_finish_initialising_local(compiler *compiler, size_t i)
{
    compiler->_locals[i].is_initialising = false;
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

// Returns the precedence level of `type` as an infix operator or `PREC_NONE` if `type` is not an
// infix operator.
static enum precedence_level infix_operator_precedence_level(enum token_type type)
{
    switch (type) {
    // case TOKEN_COMMA:
    //     return PREC_COMMA;
    // case TOKEN_QUESTION:
    //     return PREC_TERNARY;
    // case TOKEN_OR:
    //     return PREC_LOGICAL_OR;
    // case TOKEN_AND:
    //     return PREC_LOGICAL_AND;
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
        // case TOKEN_PERCENT:
        return PREC_MULTIPLICATIVE;
    // case TOKEN_LEFT_PAREN:
    // case TOKEN_LEFT_BRACK:
    // case TOKEN_DOT:
    //     return PREC_POSTFIX;
    default:
        return PREC_NONE;
    }
}

static void compiler_compile_expr_at_level(compiler *compiler, enum precedence_level level);
static void compiler_compile_expr(compiler *compiler);

// Type of function which emits the bytecode for the expression starting at the current token,
// consisting only of operators with at least the given precedence `level`, and advances the
// compiler past it.
// If an unexpected token is encountered, then an error is reported and the compiler is not advanced
// past it.
typedef void (*expr_compiler)(compiler *compiler, enum precedence_level level);

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
        compiler_compile_expr(compiler);
        is_get = false;
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

// Returns the compiler for a prefix expression starting with a `type` token or `NULL` if there is
// none.
static expr_compiler prefix_expr_compiler(enum token_type type)
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

// `expr_compiler` for a binary expression.
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

// Returns the compiler for an infix expression whose operator is a `type` token or `NULL` is there
// is none.
static expr_compiler infix_expr_compiler(enum token_type type)
{
    switch (type) {
    case TOKEN_PLUS:
    case TOKEN_MINUS:
    case TOKEN_ASTERISK:
    case TOKEN_SLASH:
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
    expr_compiler prefix_compiler = prefix_expr_compiler(compiler->_token.type);
    if (prefix_compiler == NULL) {
        COMPILER_REPORT_TOKEN_ERRORF(compiler, compiler->_token, "%s", "expected expression");
        return;
    }
    prefix_compiler(compiler, level);

    enum precedence_level infix_level;
    while ((infix_level = infix_operator_precedence_level(compiler->_token.type)) >= level) {
        expr_compiler infix_compiler = infix_expr_compiler(compiler->_token.type);
        infix_compiler(compiler, infix_level);
    }
}

// Emits the bytecode for the expression starting the current token and advances the compiler past
// it.
static void compiler_compile_expr(compiler *compiler)
{
    compiler_compile_expr_at_level(compiler, PREC_ASSIGNMENT);
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

static void compiler_compile_decl(compiler *compiler);

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
    compiler_emit_opcode(compiler, OP_POP, line);
}

// Emits the bytecode for the statement starting at the current token and advances the compiler past
// it.
static void compiler_compile_stmt(compiler *compiler)
{
    switch (compiler->_token.type) {
    case TOKEN_PRINT:
        compiler_compile_print_stmt(compiler);
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
            // case TOKEN_IF:
            // case TOKEN_WHILE:
            // case TOKEN_FOR:
            // case TOKEN_BREAK:
            // case TOKEN_CONTINUE:
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
