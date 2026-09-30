#ifndef CLOX_LEXER_H
#define CLOX_LEXER_H

// Types of token emitted by the lexer.
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
// Pointers contained in a token are borrowed and remain valid until the `lexer_free()` has been
// called.
struct token {
    enum token_type type;
    union {
        struct {
            // Pointer to the start of the lexeme.
            // This is not a null-terminated string, so `len` must be used to read it.
            const char *start;
            int len;
        } lexeme; // Valid when `type != TOKEN_ERROR`.
        // Valid when `type == TOKEN_ERROR`.
        // Null-terminated error message.
        const char *error_msg;
    };
    int line;
};

// Internal type, do not use.
struct _lexer_str {
    char *data;
    int size;
};

// Lexer which reads Lox source code and produces lexical tokens.
// Must be initialised with `lexer_init()` before use and freed with `lexer_free()` after use.
struct lexer {
    // Internal fields, do not use.
    const char *_char; // Points to character currently being considered
    int _line; // Current line number in the source
    struct _lexer_str *_strs; // Strings allocated as part of produced tokens
    int _strs_len; // Number of elements in `_strs`
    int _strs_cap; // Number of elements that space has been allocated for in `_strs`
};

// Initialises `lexer` for lexing `source`.
// `source` is borrowed and must remain valid and unmodified until `lexer_free()` is called.
void lexer_init(struct lexer *lexer, const char *source);

// Frees the memory associated with `lexer`.
void lexer_free(struct lexer *lexer);

// Returns the next token from the source.
// Once the end of the source has been reached, this function will always return a `TOKEN_EOF`
// token.
struct token lexer_next(struct lexer *lexer);

#endif
