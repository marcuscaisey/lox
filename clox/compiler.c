#include "compiler.h"

#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "bytecode.h"
#include "debug.h"
#include "errors.h"
#include "lexer.h"
#include "memory.h"
#include "strings.h"
#include "value.h"

// Compiler that parses Lox source code and writes the compiled bytecode into a caller provided
// chunk.
// Must be initialised with `compiler_init()` and freed with `compiler_free()` after use.
struct compiler {
    const char *_source; // Source being compiled
    struct token _token; // Token currently being considered
    struct token _next_token; // Token after `_token` in the source
    struct token _prev_token; // Token before `_token` in the source
    struct lexer _lexer; // Lexer set up to lex the source
    struct bytecode_chunk *_out; // Chunk that bytecode is written to
    // Panic mode is entered when a syntax error is encountered and exited once it has been
    // recovered from
    bool _in_panic_mode;
    bool _had_error; // Whether the compiler has encountered any syntax errors
};

// Calls `compiler_report_errorf()` with a range spanning the current token.
#define COMPILER_REPORT_TOKEN_ERRORF(compiler, format, ...)                             \
    compiler_report_errorf((compiler), (compiler)->_token.start,                        \
                           (compiler)->_token.start + (compiler)->_token.len, (format), \
                           __VA_ARGS__)

// Prints an error relating to the source range `[start, end)` with a formatted message and enters
// panic mode. If the compiler is already in panic mode, then this is a no-op. See
// `print_invalid_range_error()` for how the error is printed.
static __attribute__((format(printf, 4, 5))) void compiler_report_errorf(struct compiler *compiler,
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
    int size = vsprintf_alloc(&msg, format, args);
    if (size < 0) {
        fprintf(stderr, "compiler: encoding error formatting \"%s\"\n", format);
        abort();
    }
    va_end(args);
    print_invalid_range_error(msg, compiler->_source, start, end);
    deallocate(msg, size);
}

// Moves the current token forwards in the source until it's valid. An error is reported for each
// `TOKEN_ERROR` passed.
static void compiler_advance(struct compiler *compiler)
{
    while (true) {
        compiler->_prev_token = compiler->_token;
        compiler->_token = compiler->_next_token;
        compiler->_next_token = lexer_next(&compiler->_lexer);
        if (compiler->_token.type != TOKEN_ERROR)
            break;
        COMPILER_REPORT_TOKEN_ERRORF(compiler, "%s", compiler->_token.error_msg);
    }
}

// Initialises `compiler` for compiling `source` and writing the compiled bytecode into `out`.
// `source` and `out` must remain valid until `compiler_free()` is called.
static void compiler_init(struct compiler *compiler, const char *source, struct bytecode_chunk *out)
{
    compiler->_source = source;
    lexer_init(&compiler->_lexer, source);
    compiler->_prev_token = (struct token){ .type = TOKEN_ERROR };
    compiler->_token = (struct token){ .type = TOKEN_ERROR };
    compiler->_next_token = lexer_next(&compiler->_lexer);
    compiler_advance(compiler); // Populate current token
    compiler->_out = out;
    compiler->_in_panic_mode = false;
    compiler->_had_error = false;
}

// Frees the memory associated with `compiler`.
static void compiler_free(struct compiler *compiler)
{
    lexer_free(&compiler->_lexer);
}

// Checks whether the current token has `type`. If it does, the compiler is advanced. Otherwise, an
// error is reported.
static void compiler_expect(struct compiler *compiler, enum token_type type)
{
    if (compiler->_token.type == type)
        compiler_advance(compiler);
    else
        COMPILER_REPORT_TOKEN_ERRORF(compiler, "expected '%s'", token_type_string(type));
}

// Calls `compiler_match()` with a variable number of arguments.
#define COMPILER_MATCH(compiler, ...)                              \
    compiler_match((compiler), (enum token_type[]){ __VA_ARGS__ }, \
                   sizeof((enum token_type[]){ __VA_ARGS__ }) / sizeof(enum token_type))

// Reports whether the current token is one of the `count` given `types` and advances the compiler
// if so.
static bool compiler_match(struct compiler *compiler, enum token_type *types, int count)
{
    for (int i = 0; i < count; i++)
        if (compiler->_token.type == types[i]) {
            compiler_advance(compiler);
            return true;
        }
    return false;
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
    // case TOKEN_EQUAL:
    //     return PREC_ASSIGNMENT;
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

static void compiler_compile_expr_at_level(struct compiler *compiler, enum precedence_level level);
static void compiler_compile_expr(struct compiler *compiler);

// Writes the bytecode for the literal keyword in the current token and advances the compiler past
// it.
static void compiler_compile_literal_keyword(struct compiler *compiler)
{
    enum opcode instruction;
    if (COMPILER_MATCH(compiler, TOKEN_NIL)) {
        instruction = OP_NIL;
    } else if (COMPILER_MATCH(compiler, TOKEN_TRUE)) {
        instruction = OP_TRUE;
    } else if (COMPILER_MATCH(compiler, TOKEN_FALSE)) {
        instruction = OP_FALSE;
    } else {
        COMPILER_REPORT_TOKEN_ERRORF(compiler, "%s", "expected literal keyword");
        return;
    }
    bytecode_chunk_write(compiler->_out, instruction, compiler->_prev_token.line);
}

// Writes the bytecode for the number literal in the current token and advances the compiler past
// it.
static void compiler_compile_number(struct compiler *compiler)
{
    compiler_expect(compiler, TOKEN_NUMBER);
    double n = strtod(compiler->_prev_token.start, NULL);
    bytecode_chunk_write_constant(compiler->_out, value_number(n), compiler->_prev_token.line);
}

// Writes the bytecode for the group e_source = source;xpression starting at the current token and advances the
// compiler past it.
static void compiler_compile_group_expr(struct compiler *compiler)
{
    compiler_expect(compiler, TOKEN_LEFT_PAREN);
    compiler_compile_expr(compiler);
    compiler_expect(compiler, TOKEN_RIGHT_PAREN);
}

// Write the bytecode for the unary expression starting at the current token and advances the
// compiler past it.
static void compiler_compile_unary_expr(struct compiler *compiler)
{
    int line = compiler->_token.line;
    enum opcode instruction;
    if (COMPILER_MATCH(compiler, TOKEN_MINUS)) {
        instruction = OP_NEGATE;
    } else if (COMPILER_MATCH(compiler, TOKEN_BANG)) {
        instruction = OP_NOT;
    } else {
        COMPILER_REPORT_TOKEN_ERRORF(compiler, "%s", "expected unary operator");
        return;
    }
    compiler_compile_expr_at_level(compiler, PREC_UNARY);
    bytecode_chunk_write(compiler->_out, instruction, line);
}

// Type of function which writes the bytecode for an expression and advances the compiler past it.
typedef void (*expr_compiler)(struct compiler *compiler);

// Returns the compiler for a prefix expression starting with a `type` token or `NULL` if there is
// none.
static expr_compiler prefix_expr_compiler(enum token_type type)
{
    switch (type) {
    case TOKEN_NIL:
    case TOKEN_TRUE:
    case TOKEN_FALSE:
        return compiler_compile_literal_keyword;
    case TOKEN_NUMBER:
        return compiler_compile_number;
    case TOKEN_LEFT_PAREN:
        return compiler_compile_group_expr;
    case TOKEN_MINUS:
    case TOKEN_BANG:
        return compiler_compile_unary_expr;
    default:
        return NULL;
    }
}

// Writes the bytecode for the right hand side of a binary expression whose operator is at the
// current token and advances the compiler past the it.
static void compiler_compile_binary_expr(struct compiler *compiler)
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
        COMPILER_REPORT_TOKEN_ERRORF(compiler, "%s", "expected binary operator");
        return;
    }
    enum precedence_level current_level =
        infix_operator_precedence_level(compiler->_prev_token.type);
    compiler_compile_expr_at_level(compiler, current_level + 1);
    bytecode_chunk_write(compiler->_out, instruction, line);
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

// Writes the bytecode for the expression starting at the current token consisting only of operators
// with at least the given precedence `level` and advances the compiler past it.
static void compiler_compile_expr_at_level(struct compiler *compiler, enum precedence_level level)
{
    expr_compiler prefix_compiler = prefix_expr_compiler(compiler->_token.type);
    if (prefix_compiler == NULL) {
        COMPILER_REPORT_TOKEN_ERRORF(compiler, "%s", "expected expression");
        return;
    }
    prefix_compiler(compiler);

    while (infix_operator_precedence_level(compiler->_token.type) >= level) {
        expr_compiler infix_compiler = infix_expr_compiler(compiler->_token.type);
        infix_compiler(compiler);
    }
}

// Writes the bytecode for the expression starting the current token and advances the compiler past
// it.
static void compiler_compile_expr(struct compiler *compiler)
{
    compiler_compile_expr_at_level(compiler, PREC_ASSIGNMENT);
}

// Writes the bytecode for the program and reports whether compilation was successful.
static bool compiler_compile(struct compiler *compiler)
{
    compiler_compile_expr(compiler);
    compiler_expect(compiler, TOKEN_SEMICOLON);
    bytecode_chunk_write(compiler->_out, OP_RETURN, compiler->_token.line);
    bool success = !compiler->_had_error;
#ifdef DEBUG
    if (success)
        disassemble(*compiler->_out, "code");
#endif
    return success;
}

bool compile(const char *source, struct bytecode_chunk *out)
{
    struct compiler compiler;
    compiler_init(&compiler, source, out);
    bool success = compiler_compile(&compiler);
    compiler_free(&compiler);
    return success;
}
