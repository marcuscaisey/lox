#include "lexer.h"

#include <ctype.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdio.h>
#include <string.h>

#include "memory.h"

void lexer_init(struct lexer *lexer, const char *source)
{
    lexer->_char = source;
    lexer->_line = 1;
    lexer->_strs = NULL;
    lexer->_strs_len = 0;
    lexer->_strs_cap = 0;
}

void lexer_free(struct lexer *lexer)
{
    for (struct _lexer_str *s = lexer->_strs; s < lexer->_strs + lexer->_strs_len; s++)
        deallocate(s->data, s->size);
    deallocate(lexer->_strs, lexer->_strs_cap);
}

// Updates `_char` to point to the next character in the source and updates `_line` if the next line
// has been reached.
// If `_char` is already at the end of the source, then this is a no-op.
static void lexer_advance(struct lexer *lexer)
{
    if (*lexer->_char == '\0')
        return;
    if (*lexer->_char == '\n')
        lexer->_line++;
    lexer->_char++;
}

// Returns the character following `_char` if there is one, otherwise '\0'.
static char lexer_peek(const struct lexer *lexer)
{
    return *lexer->_char != '\0' ? *(lexer->_char + 1) : '\0';
}

// Advances the lexer until `_char` does not point at a character which is not semantically
// meaningful.
static void lexer_skip_ignored(struct lexer *lexer)
{
    while (true) {
        switch (*lexer->_char) {
        // Whitespace
        case ' ':
        case '\r':
        case '\t':
        case '\n':
            lexer_advance(lexer);
            break;
        // Comments
        case '/':
            if (lexer_peek(lexer) == '/') {
                lexer_advance(lexer); // first /
                lexer_advance(lexer); // second /
                while (*lexer->_char != '\n' && *lexer->_char != '\0')
                    lexer_advance(lexer);
            } else {
                return;
            }
            break;
        default:
            return;
        }
    }
}

// Allocates and returns a string formatted as if with `sprintf()` which will be freed when
// `lexer_free()` is called.
static char *lexer_sprintf(struct lexer *lexer, const char *format, ...)
{
    if (lexer->_strs_len + 1 > lexer->_strs_cap) {
        size_t current_size = lexer->_strs_cap * sizeof(*lexer->_strs);
        lexer->_strs_cap = lexer->_strs_cap < 8 ? 8 : lexer->_strs_cap * 2;
        size_t target_size = lexer->_strs_cap * sizeof(*lexer->_strs);
        lexer->_strs = reallocate(lexer->_strs, current_size, target_size);
    }

    va_list args;
    va_start(args, format);
    int size = vsnprintf(NULL, 0, format, args) + 1;
    va_end(args);

    char *result = allocate(size);
    va_start(args, format);
    vsnprintf(result, size, format, args);
    va_end(args);

    lexer->_strs[lexer->_strs_len++] = (struct _lexer_str){ .data = result, .size = size };
    return result;
}

// Returns the type of the `len` long identifier starting at `start`. If the identifier is a
// keyword, then that type (`TOKEN_PRINT`, `TOKEN_VAR`, etc) is returned, otherwise `TOKEN_IDENT` is
// returned.
static enum token_type ident_type(const char *start, int len)
{
    // Returns `keyword_type` if the identifier is equal to `name` starting from `offset`, otherwise `TOKEN_IDENT`
#define KEYWORD_TYPE_OR_IDENT(keyword_type, name, offset)                                  \
    len == sizeof(name) - 1 && strncmp(start + offset, &name[offset], len - offset) == 0 ? \
        keyword_type :                                                                     \
        TOKEN_IDENT

    switch (start[0]) {
    case 'a':
        return KEYWORD_TYPE_OR_IDENT(TOKEN_AND, "and", 1);
    case 'b':
        return KEYWORD_TYPE_OR_IDENT(TOKEN_BREAK, "break", 2);
    case 'c':
        if (len > 1) {
            switch (start[1]) {
            case 'l':
                return KEYWORD_TYPE_OR_IDENT(TOKEN_CLASS, "class", 2);
            case 'o':
                return KEYWORD_TYPE_OR_IDENT(TOKEN_CONTINUE, "continue", 2);
            }
        }
        break;
    case 'e':
        return KEYWORD_TYPE_OR_IDENT(TOKEN_ELSE, "else", 1);
    case 'f':
        if (len > 1) {
            switch (start[1]) {
            case 'a':
                return KEYWORD_TYPE_OR_IDENT(TOKEN_FALSE, "false", 2);
            case 'o':
                return KEYWORD_TYPE_OR_IDENT(TOKEN_FOR, "for", 2);
            case 'u':
                return KEYWORD_TYPE_OR_IDENT(TOKEN_FUN, "fun", 2);
            }
        }
        break;
    case 'g':
        return KEYWORD_TYPE_OR_IDENT(TOKEN_GET, "get", 1);
    case 'i':
        return KEYWORD_TYPE_OR_IDENT(TOKEN_IF, "if", 1);
    case 'n':
        return KEYWORD_TYPE_OR_IDENT(TOKEN_NIL, "nil", 1);
    case 'o':
        return KEYWORD_TYPE_OR_IDENT(TOKEN_OR, "or", 1);
    case 'p':
        return KEYWORD_TYPE_OR_IDENT(TOKEN_PRINT, "print", 1);
    case 'r':
        return KEYWORD_TYPE_OR_IDENT(TOKEN_RETURN, "return", 1);
    case 's':
        if (len > 1) {
            switch (start[1]) {
            case 'e':
                return KEYWORD_TYPE_OR_IDENT(TOKEN_SET, "set", 2);
            case 't':
                return KEYWORD_TYPE_OR_IDENT(TOKEN_STATIC, "static", 2);
            case 'u':
                return KEYWORD_TYPE_OR_IDENT(TOKEN_SUPER, "super", 2);
            }
        }
        break;
    case 't':
        if (len > 1) {
            switch (start[1]) {
            case 'h':
                return KEYWORD_TYPE_OR_IDENT(TOKEN_THIS, "this", 2);
            case 'r':
                if (len > 2) {
                    switch (start[2]) {
                    case 'u':
                        return KEYWORD_TYPE_OR_IDENT(TOKEN_TRUE, "true", 3);
                    case 'y':
                        return KEYWORD_TYPE_OR_IDENT(TOKEN_TRY, "try", 3);
                    }
                }
            }
        }
        break;
    case 'v':
        return KEYWORD_TYPE_OR_IDENT(TOKEN_VAR, "var", 1);
    case 'w':
        return KEYWORD_TYPE_OR_IDENT(TOKEN_WHILE, "while", 1);
    }

    return TOKEN_IDENT;

#undef KEYWORD_OR_IDENT_TYPE
}

struct token lexer_next(struct lexer *lexer)
{
    lexer_skip_ignored(lexer);

    struct token token;
    token.lexeme.start = lexer->_char;
    token.line = lexer->_line;

    char prev_char = *lexer->_char;
    lexer_advance(lexer);
    switch (prev_char) {
    case '\0':
        token.type = TOKEN_EOF;
        break;
    case ';':
        token.type = TOKEN_SEMICOLON;
        break;
    case ',':
        token.type = TOKEN_COMMA;
        break;
    case '.':
        token.type = TOKEN_DOT;
        break;
    case '=':
        token.type = TOKEN_EQUAL;
        if (*lexer->_char == '=') {
            lexer_advance(lexer);
            token.type = TOKEN_EQUAL_EQUAL;
        }
        break;
    case '+':
        token.type = TOKEN_PLUS;
        break;
    case '-':
        token.type = TOKEN_MINUS;
        break;
    case '*':
        token.type = TOKEN_ASTERISK;
        break;
    case '/':
        token.type = TOKEN_SLASH;
        break;
    case '%':
        token.type = TOKEN_PERCENT;
        break;
    case '<':
        token.type = TOKEN_LESS;
        if (*lexer->_char == '=') {
            lexer_advance(lexer);
            token.type = TOKEN_LESS_EQUAL;
        }
        break;
    case '>':
        token.type = TOKEN_GREATER;
        if (*lexer->_char == '=') {
            lexer_advance(lexer);
            token.type = TOKEN_GREATER_EQUAL;
        }
        break;
    case '!':
        token.type = TOKEN_BANG;
        if (*lexer->_char == '=') {
            lexer_advance(lexer);
            token.type = TOKEN_BANG_EQUAL;
        }
        break;
    case '?':
        token.type = TOKEN_QUESTION;
        break;
    case ':':
        token.type = TOKEN_COLON;
        break;
    case '(':
        token.type = TOKEN_LEFT_PAREN;
        break;
    case ')':
        token.type = TOKEN_RIGHT_PAREN;
        break;
    case '[':
        token.type = TOKEN_LEFT_BRACK;
        break;
    case ']':
        token.type = TOKEN_RIGHT_BRACK;
        break;
    case '{':
        token.type = TOKEN_LEFT_BRACE;
        break;
    case '}':
        token.type = TOKEN_RIGHT_BRACE;
        break;
    case '"':
        token.type = TOKEN_STRING;
        // Advance past string literal
        while (true) {
            if (*lexer->_char == '\0') {
                token.type = TOKEN_ERROR;
                token.error_msg = "unterminated string literal";
                return token;
            }
            char prev_char = *lexer->_char;
            lexer_advance(lexer);
            if (prev_char == '"')
                break;
        }
        break;
    default: {
        if (isalpha(prev_char) || prev_char == '_') {
            // Advance past identifier
            while (isalnum(*lexer->_char) || *lexer->_char == '_')
                lexer_advance(lexer);
            int len = lexer->_char - token.lexeme.start;
            token.type = ident_type(token.lexeme.start, len);
            break;

        } else if (isdigit(prev_char)) {
            token.type = TOKEN_NUMBER;
            // Advance past number literal
            while (isdigit(*lexer->_char))
                lexer_advance(lexer);
            if (*lexer->_char == '.' && isdigit(lexer_peek(lexer))) {
                lexer_advance(lexer); // .
                lexer_advance(lexer); // digit
                while (isdigit(*lexer->_char))
                    lexer_advance(lexer);
            }
            break;
        }

        token.type = TOKEN_ERROR;
        token.error_msg = lexer_sprintf(lexer, "illegal character '%c'", prev_char);
        return token;
    }
    }

    token.lexeme.len = lexer->_char - token.lexeme.start;

    return token;
}
