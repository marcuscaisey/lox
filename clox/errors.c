#include "errors.h"

#include <stdio.h>

static const char *ansi_reset = "\x1b[0m";
static const char *ansi_bold = "\x1b[1m";
static const char *ansi_faint = "\x1b[2m";
static const char *ansi_red = "\x1b[31m";
static const char *ansi_yellow = "\x1b[33m";
static const char *ansi_default = "\x1b[39m";

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
    fprintf(stderr, "%s", ansi_bold);
    fprintf(stderr, "%s%d", ansi_yellow, line);
    fprintf(stderr, "%s:", ansi_default);
    fprintf(stderr, "%s%d", ansi_yellow, column);
    fprintf(stderr, "%s: ", ansi_default);
    fprintf(stderr, "%serror", ansi_red);
    fprintf(stderr, "%s: ", ansi_default);
    fprintf(stderr, "%s", msg);
    fprintf(stderr, "%s\n", ansi_reset);

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
        fprintf(stderr, "%s", ansi_faint);
        for (const char *p = line_start; *p != '\n' && *p != '\0'; p++) {
            fprintf(stderr, "%c", *p);
        }
        fprintf(stderr, "\n%s", ansi_reset);
        // Print tildes
        fprintf(stderr, "%s", ansi_red);
        const char *p;
        for (p = line_start; *p != '\n' && *p != '\0'; p++) {
            fprintf(stderr, start <= p && p < end ? "~" : " ");
        }
        line_start = p + 1;
        fprintf(stderr, "\n%s", ansi_reset);
    }
}
