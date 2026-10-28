#include "errors.h"

#include <ctype.h>
#include <stdarg.h>
#include <stddef.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

#include "line_number.h"
#include "vm.h"

// Concrete styles. These are used to build semantic styles.
#define ANSI_RESET "\x1b[0m"
#define ANSI_BOLD "\x1b[1m"
#define ANSI_FAINT "\x1b[2m"
#define ANSI_RED "\x1b[31m"
#define ANSI_YELLOW "\x1b[33m"
#define ANSI_DEFAULT "\x1b[39m"

// Semantic styles. These are used in messages.
// These may be modified by `maybe_unset_semantic_styles()` so that function needs to be updated if
// a new style is added.
static const char *error_msg_line_style = ANSI_BOLD;
static const char *title_style = ANSI_BOLD;
static const char *location_style = ANSI_YELLOW;
static const char *location_separator_style = ANSI_DEFAULT;
static const char *error_style = ANSI_RED;
static const char *error_msg_style = ANSI_DEFAULT;
static const char *code_snippet_style = ANSI_FAINT;
static const char *reset_style = ANSI_RESET;

// Sets all of the semantic styles to empty strings if stderr is not a tty.
static void maybe_unset_semantic_styles(void)
{
    if (!isatty(STDERR_FILENO)) {
        error_msg_line_style = "";
        location_style = "";
        location_separator_style = "";
        error_style = "";
        error_msg_style = "";
        code_snippet_style = "";
        reset_style = "";
    }
}

void print_invalid_range_error(const char *msg, const char *source, const char *start,
                               const char *end)
{
    maybe_unset_semantic_styles();

    line_number line = 1;
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
    fprintf(stderr, "%s\n", reset_style);

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
        fprintf(stderr, "\n%s", reset_style);
        // Print tildes
        fprintf(stderr, "%s", error_style);
        const char *p;
        for (p = line_start; *p != '\n' && *p != '\0'; p++) {
            fprintf(stderr, start <= p && p < end ? "~" : " ");
        }
        line_start = p + 1;
        fprintf(stderr, "\n%s", reset_style);
    }
}

void print_invalid_line_error(const char *msg, const char *source, line_number line)
{
    maybe_unset_semantic_styles();

    // Print error message which looks like:
    //   3: error: '+' operator cannot be used with types 'number' and 'bool'
    fprintf(stderr, "%s", error_msg_line_style);
    fprintf(stderr, "%s%d", location_style, line);
    fprintf(stderr, "%s: ", location_separator_style);
    fprintf(stderr, "%serror", error_style);
    fprintf(stderr, "%s: %s", error_msg_style, msg);
    fprintf(stderr, "%s\n", reset_style);

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
    fprintf(stderr, "\n%s", reset_style);
    // Print tildes
    fprintf(stderr, "%s", error_style);
    const char *p;
    for (p = line_start; *p != '\n' && *p != '\0'; p++) {
        fprintf(stderr, "~");
    }
    fprintf(stderr, "\n%s", reset_style);
}

void print_stack_trace(const struct frame *frames, size_t n, const char *source)
{
    maybe_unset_semantic_styles();

    line_number *lines = malloc(n * sizeof(*lines));
    int max_line = 0;
    const char **names = malloc(n * sizeof(*names));
    int max_name_len = 0;
    for (size_t i = 0; i < n; i++) {
        struct frame frame = frames[n - 1 - i];
        lines[i] = frame_last_instruction_line(&frame);
        if (lines[i] > max_line)
            max_line = lines[i];
        if (frame.name != NULL) {
            names[i] = frame.name->data;
            if ((int)frame.name->len > max_name_len)
                max_name_len = frame.name->len;
        } else {
            names[i] = NULL;
        }
    }
    int max_line_width = 0;
    for (; max_line > 0; max_line /= 10)
        max_line_width++;

    fprintf(stderr, "%sStack Trace (most recent call first):\n", title_style);

    // Print frame summary which looks like:
    //   10 in foobar error("ahhhh");
    //
    // Or if the frame has no name:
    //   13           foo();
    fprintf(stderr, "%s", reset_style);
    for (size_t i = 0; i < n; i++) {
        line_number line = lines[i];
        // Print line
        fprintf(stderr, "  %s%-*d ", location_style, max_line_width, line);
        fprintf(stderr, "%s", reset_style);
        // Print callable name
        if (names[i] != NULL) {
            fprintf(stderr, "in %-*s ", max_name_len, names[i]);
        } else {
            fprintf(stderr, "   ");
            for (int i = 0; i < max_name_len; i++)
                fprintf(stderr, " ");
            fprintf(stderr, " ");
        }
        // Print code snippet
        const char *snippet_start = source;
        line_number current_line = 1;
        for (const char *p = source; current_line < line; p++)
            if (*p == '\n') {
                current_line++;
                snippet_start = p + 1;
            }
        while (isspace(*snippet_start))
            snippet_start++;
        fprintf(stderr, "%s", code_snippet_style);
        for (const char *p = snippet_start; *p != '\n' && *p != '\0'; p++) {
            fprintf(stderr, "%c", *p);
        }
        fprintf(stderr, "\n%s", reset_style);
    }
}

void exitf(const char *format, ...)
{
    va_list args;
    va_start(args, format);
    vfprintf(stderr, format, args);
    va_end(args);
    exit(1);
}
