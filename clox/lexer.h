#ifndef CLOX_LEXER_H
#define CLOX_LEXER_H

// Types of token emitted by the lexer.
#include <stddef.h>
enum token_type {
    TOKEN_ERROR,
    TOKEN_EOF,
    // Keywords
    TOKEN_PRINT,
    TOKEN_VAR,
    TOKEN_TRUE,
    TOKEN_FALSE,
    TOKEN_NIL,
    TOKEN_IF,
    TOKEN_ELSE,
    TOKEN_AND,
    TOKEN_OR,
    TOKEN_WHILE,
    TOKEN_FOR,
    TOKEN_BREAK,
    TOKEN_CONTINUE,
    TOKEN_FUN,
    TOKEN_RETURN,
    TOKEN_CLASS,
    TOKEN_THIS,
    TOKEN_SUPER,
    TOKEN_STATIC,
    TOKEN_GET,
    TOKEN_SET,
    TOKEN_TRY,
    // Literals
    TOKEN_IDENT,
    TOKEN_STRING,
    TOKEN_NUMBER,
    // Symbols
    TOKEN_SEMICOLON,
    TOKEN_COMMA,
    TOKEN_DOT,
    TOKEN_EQUAL,
    TOKEN_PLUS,
    TOKEN_MINUS,
    TOKEN_ASTERISK,
    TOKEN_SLASH,
    TOKEN_PERCENT,
    TOKEN_LESS,
    TOKEN_LESS_EQUAL,
    TOKEN_GREATER,
    TOKEN_GREATER_EQUAL,
    TOKEN_EQUAL_EQUAL,
    TOKEN_BANG_EQUAL,
    TOKEN_BANG,
    TOKEN_QUESTION,
    TOKEN_COLON,
    TOKEN_LEFT_PAREN,
    TOKEN_RIGHT_PAREN,
    TOKEN_LEFT_BRACK,
    TOKEN_RIGHT_BRACK,
    TOKEN_LEFT_BRACE,
    TOKEN_RIGHT_BRACE,
};

// Returns a string representation of `type`. If `type` is represented in Lox source code by a fixed
// string, then this is what's returned.
const char *token_type_string(enum token_type type);

// A lexical token produced by the lexer.
// Pointers contained in a token remain valid until `lexer_free()` has been called.
struct token {
    enum token_type type;
    // Pointer to the start of the lexeme in the source.
    // This is not a null-terminated string, so `len` must be used to read it.
    const char *start;
    size_t len;
    const char *error_msg; // Null-terminated error message. Valid when `type == TOKEN_ERROR`.
    int line; // Line that the token starts on
};

// Lexer which reads Lox source code and produces lexical tokens.
// Must be initialised with `lexer_init()` before use and freed with `lexer_free()` after use.
typedef struct {
    // Internal fields, do not use.
    const char *_source_start; // Points to start of source
    const char *_char; // Points to character in source currently being considered
    int _line; // Line number of the current character
    char **_strs; // Strings allocated as part of produced tokens
    size_t _strs_len; // Number of elements in `_strs`
    size_t _strs_cap; // Number of elements that space has been allocated for in `_strs`
} lexer;

// Initialises `lexer` for lexing `source`.
// `source` must remain valid until `lexer_free()` is called.
void lexer_init(lexer *lexer, const char *source);

// Frees the memory associated with `lexer`.
void lexer_free(lexer *lexer);

// Returns the next token from the source.
// Once the end of the source has been reached, this function will always return a `TOKEN_EOF`
// token.
struct token lexer_next(lexer *lexer);

#endif
