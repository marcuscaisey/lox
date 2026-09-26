#include "compiler.h"

#include <stdbool.h>
#include <stdio.h>
#include <string.h>

#include "lexer.h"

void compile(const char *source)
{
    struct lexer lexer;
    lexer_init(&lexer, source);

    int line = -1;
    while (true) {
        struct token token = lexer_next(&lexer);
        if (token.line != line) {
            printf("%4d ", token.line);
            line = token.line;
        } else {
            printf("   | ");
        }
        const char *start = token.lexeme.start;
        int len = token.lexeme.len;
        if (token.type == TOKEN_ERROR) {
            start = token.error_msg;
            len = strlen(start);
        }
        printf("%2d %.*s\n", token.type, len, start);
        if (token.type == TOKEN_EOF)
            break;
    }

    lexer_free(&lexer);
}
