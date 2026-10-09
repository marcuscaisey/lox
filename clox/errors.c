#include "errors.h"

#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>

// Concrete styles
#define ANSI_RESET "\x1b[0m"
#define ANSI_BOLD "\x1b[1m"
#define ANSI_FAINT "\x1b[2m"
#define ANSI_RED "\x1b[31m"
#define ANSI_YELLOW "\x1b[33m"
#define ANSI_DEFAULT "\x1b[39m"

// Semantic styles
static const char *const error_msg_line_style = ANSI_BOLD;
static const char *const location_style = ANSI_YELLOW;
static const char *const location_separator_style = ANSI_DEFAULT;
static const char *const error_style = ANSI_RED;
static const char *const error_msg_style = ANSI_DEFAULT;
static const char *const code_snippet_style = ANSI_FAINT;

void print_invalid_range_error(const char *msg, const char *source, const char *start,
                               const char *end)
{
    int line = 1;
    // Points to the start of the line that start is on
    const char *start_line_start = source;
    for (const char *p = source; p <= start; p++)
        if (*p == '\n') {
            line++;
            start_line_start = p + 1;
        }
    int column = start - start_line_start + 1;

    // Print error message which looks like:
    //   3:21: error: expected expression
    fprintf(stderr, "%s", error_msg_line_style);
    fprintf(stderr, "%s%d", location_style, line);
    fprintf(stderr, "%s:", location_separator_style);
    fprintf(stderr, "%s%d", location_style, column);
    fprintf(stderr, "%s: ", location_separator_style);
    fprintf(stderr, "%serror", error_style);
    fprintf(stderr, "%s: %s", error_msg_style, msg);
    fprintf(stderr, "%s\n", ANSI_RESET);

    // Print highlighted lines which look like:
    //   (-1 + 2) * 3 - -4 + print
    //                       ~~~~~
    // Or if the range [start, end) spans multiple lines, then:
    //   1
    //   ~
    //     +
    //   ~~~
    //       2 = 3;
    //   ~~~~~
    const char *line_start = start_line_start;
    while (line_start < end) {
        // Print line
        fprintf(stderr, "%s", code_snippet_style);
        for (const char *p = line_start; *p != '\n' && *p != '\0'; p++) {
            fprintf(stderr, "%c", *p);
        }
        fprintf(stderr, "\n%s", ANSI_RESET);
        // Print tildes
        fprintf(stderr, "%s", error_style);
        const char *p;
        for (p = line_start; *p != '\n' && *p != '\0'; p++) {
            fprintf(stderr, start <= p && p < end ? "~" : " ");
        }
        line_start = p + 1;
        fprintf(stderr, "\n%s", ANSI_RESET);
    }
}

void print_invalid_line_error(const char *msg, const char *source, int line)
{
    // Print error message which looks like:
    //   3: error: '+' operator cannot be used with types 'number' and 'bool'
    fprintf(stderr, "%s", error_msg_line_style);
    fprintf(stderr, "%s%d", location_style, line);
    fprintf(stderr, "%s: ", location_separator_style);
    fprintf(stderr, "%serror", error_style);
    fprintf(stderr, "%s: %s", error_msg_style, msg);
    fprintf(stderr, "%s\n", ANSI_RESET);

    const char *line_start = source;
    int current_line = 1;
    for (const char *p = line_start; *p != '\0' && current_line != line; p++)
        if (*p == '\n') {
            current_line++;
            line_start = p + 1;
        }

    // Print highlighted line which looks like:
    //   1 + false;
    //   ~~~~~~~~~~
    // Print line
    fprintf(stderr, "%s", code_snippet_style);
    for (const char *p = line_start; *p != '\n' && *p != '\0'; p++) {
        fprintf(stderr, "%c", *p);
    }
    fprintf(stderr, "\n%s", ANSI_RESET);
    // Print tildes
    fprintf(stderr, "%s", error_style);
    const char *p;
    for (p = line_start; *p != '\n' && *p != '\0'; p++) {
        fprintf(stderr, "~");
    }
    fprintf(stderr, "\n%s", ANSI_RESET);
}

void panicf(const char *format, ...)
{
    va_list args;
    va_start(args, format);
    vfprintf(stderr, format, args);
    va_end(args);
    abort();
}
