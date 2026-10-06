#include "lexer.h"

#include <ctype.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "dynamic_array.h"
#include "strings.h"

const char *token_type_string(enum token_type type)
{
    switch (type) {
    case TOKEN_ERROR:
        return "ERROR";
    case TOKEN_EOF:
        return "EOF";
    case TOKEN_PRINT:
        return "print";
    case TOKEN_VAR:
        return "var";
    case TOKEN_TRUE:
        return "true";
    case TOKEN_FALSE:
        return "false";
    case TOKEN_NIL:
        return "nil";
    case TOKEN_IF:
        return "if";
    case TOKEN_ELSE:
        return "else";
    case TOKEN_AND:
        return "and";
    case TOKEN_OR:
        return "or";
    case TOKEN_WHILE:
        return "while";
    case TOKEN_FOR:
        return "for";
    case TOKEN_BREAK:
        return "break";
    case TOKEN_CONTINUE:
        return "continue";
    case TOKEN_FUN:
        return "fun";
    case TOKEN_RETURN:
        return "return";
    case TOKEN_CLASS:
        return "class";
    case TOKEN_THIS:
        return "this";
    case TOKEN_SUPER:
        return "super";
    case TOKEN_STATIC:
        return "static";
    case TOKEN_GET:
        return "get";
    case TOKEN_SET:
        return "set";
    case TOKEN_TRY:
        return "try";
    case TOKEN_IDENT:
        return "IDENT";
    case TOKEN_STRING:
        return "STRING";
    case TOKEN_NUMBER:
        return "NUMBER";
    case TOKEN_SEMICOLON:
        return ";";
    case TOKEN_COMMA:
        return ",";
    case TOKEN_DOT:
        return ".";
    case TOKEN_EQUAL:
        return "=";
    case TOKEN_PLUS:
        return "+";
    case TOKEN_MINUS:
        return "-";
    case TOKEN_ASTERISK:
        return "*";
    case TOKEN_SLASH:
        return "/";
    case TOKEN_PERCENT:
        return "%";
    case TOKEN_LESS:
        return "<";
    case TOKEN_LESS_EQUAL:
        return "<=";
    case TOKEN_GREATER:
        return ">";
    case TOKEN_GREATER_EQUAL:
        return ">=";
    case TOKEN_EQUAL_EQUAL:
        return "==";
    case TOKEN_BANG_EQUAL:
        return "!=";
    case TOKEN_BANG:
        return "!";
    case TOKEN_QUESTION:
        return "?";
    case TOKEN_COLON:
        return ":";
    case TOKEN_LEFT_PAREN:
        return "(";
    case TOKEN_RIGHT_PAREN:
        return ")";
    case TOKEN_LEFT_BRACK:
        return "[";
    case TOKEN_RIGHT_BRACK:
        return "]";
    case TOKEN_LEFT_BRACE:
        return "{";
    case TOKEN_RIGHT_BRACE:
        return "}";
    }
}

void lexer_init(struct lexer *lexer, const char *source)
{
    lexer->_source_start = source;
    lexer->_char = lexer->_source_start;
    lexer->_line = 1;
    lexer->_strs = NULL;
    lexer->_strs_len = 0;
    lexer->_strs_cap = 0;
}

void lexer_free(struct lexer *lexer)
{
    for (char **s = lexer->_strs; s < lexer->_strs + lexer->_strs_len; s++)
        free(*s);
    free(lexer->_strs);
}

// Moves the current character forwards one character in the source.
// If the current charcter is already at the end of the source, then this is a no-op.
static void lexer_advance(struct lexer *lexer)
{
    if (*lexer->_char == '\0')
        return;
    if (*lexer->_char == '\n')
        lexer->_line++;
    lexer->_char++;
}

// Returns the character following the current character if there is one, otherwise '\0'.
static char lexer_peek(struct lexer lexer)
{
    return *lexer._char != '\0' ? *(lexer._char + 1) : '\0';
}

// Advances the lexer until the current character is semantically meaningful.
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
            if (lexer_peek(*lexer) == '/') {
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

// Works like `sprintf()`, except a new string is allocated for the output which will be freed when
// `lexer_free()` is called.
static __attribute__((format(printf, 2, 3))) char *lexer_sprintf(struct lexer *lexer,
                                                                 const char *format, ...)
{
    char *result;
    va_list args;
    va_start(args, format);
    int size = vasprintf(&result, format, args);
    if (size < 0) {
        fprintf(stderr, "lexer: encoding error formatting \"%s\"\n", format);
        abort();
    }
    va_end(args);

    DYNAMIC_ARRAY_GROW(lexer->_strs, lexer->_strs_cap, lexer->_strs_len + 1);
    lexer->_strs[lexer->_strs_len++] = result;
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
    token.start = lexer->_char;
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
        do {
            if (*lexer->_char == '\0') {
                token.type = TOKEN_ERROR;
                token.error_msg = "unterminated string literal";
                break;
            }
            prev_char = *lexer->_char;
            lexer_advance(lexer);
        } while (prev_char != '"');
        break;
    default: {
        if (isalpha(prev_char) || prev_char == '_') {
            // Advance past identifier
            while (isalnum(*lexer->_char) || *lexer->_char == '_')
                lexer_advance(lexer);
            int len = lexer->_char - token.start;
            token.type = ident_type(token.start, len);

        } else if (isdigit(prev_char)) {
            token.type = TOKEN_NUMBER;
            // Advance past number literal
            while (isdigit(*lexer->_char))
                lexer_advance(lexer);
            if (*lexer->_char == '.' && isdigit(lexer_peek(*lexer))) {
                lexer_advance(lexer); // .
                lexer_advance(lexer); // digit
                while (isdigit(*lexer->_char))
                    lexer_advance(lexer);
            }

        } else {
            token.type = TOKEN_ERROR;
            token.error_msg = lexer_sprintf(lexer, "illegal character '%c'", prev_char);
        }
    }
    }

    token.len = lexer->_char - token.start;

    return token;
}
