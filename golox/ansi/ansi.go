// Package ansi implements formatting of output text using ANSI escape sequences by wrapping the [fmt] package.
//
// Format strings (or string arguments to functions which don't accept a format string) can contain placeholders of the
// form ${NAME}, where NAME is the name of an ANSI code. The placeholder will be replaced with the corresponding ANSI
// escape sequence in the output. For functions which accept a format string and arguments, the ${NAME} placeholders are
// replaced after the arguments have been interpolated into the format string. Placeholders can therefore can be
// dynamically defined.
//
// The following ANSI codes are supported:
//   - RESET
//   - BOLD
//   - FAINT
//   - ITALIC
//   - UNDERLINE
//   - BLINKING
//   - INVERSE
//   - HIDDEN
//   - STRIKETHROUGH
//   - RESET_BOLD
//   - RESET_ITALIC
//   - RESET_UNDERLINE
//   - RESET_BLINKING
//   - RESET_INVERSE
//   - RESET_HIDDEN
//   - RESET_STRIKETHROUGH
//   - BLACK
//   - RED
//   - GREEN
//   - YELLOW
//   - BLUE
//   - MAGENTA
//   - CYAN
//   - WHITE
//   - DEFAULT
package ansi

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

func init() {
	ansiOldnew := make([]string, 2*len(ansiCodes))
	emptyOldnew := make([]string, 2*len(ansiCodes))
	for name, ansiCode := range ansiCodes {
		ansiOldnew = append(ansiOldnew, fmt.Sprintf("${%s}", name), fmt.Sprintf("\x1b[%dm", ansiCode))
		emptyOldnew = append(emptyOldnew, fmt.Sprintf("${%s}", name), "")
	}
	ansiReplacer = strings.NewReplacer(ansiOldnew...)
	emptyReplacer = strings.NewReplacer(emptyOldnew...)
}

// Enabled determines whether ANSI escape sequences will be output by the functions in this package.
// If stdout and stderr are both connected to a terminal, this will be true.
var Enabled = term.IsTerminal(int(os.Stdout.Fd())) && term.IsTerminal(int(os.Stderr.Fd()))

var ansiCodes = map[string]int{
	"RESET":               0,
	"BOLD":                1,
	"FAINT":               2,
	"ITALIC":              3,
	"UNDERLINE":           4,
	"BLINKING":            5,
	"INVERSE":             7,
	"HIDDEN":              8,
	"STRIKETHROUGH":       9,
	"RESET_BOLD":          22,
	"RESET_ITALIC":        23,
	"RESET_UNDERLINE":     24,
	"RESET_BLINKING":      25,
	"RESET_INVERSE":       27,
	"RESET_HIDDEN":        28,
	"RESET_STRIKETHROUGH": 29,
	"BLACK":               30,
	"RED":                 31,
	"GREEN":               32,
	"YELLOW":              33,
	"BLUE":                34,
	"MAGENTA":             35,
	"CYAN":                36,
	"WHITE":               37,
	"DEFAULT":             39,
}

var ansiReplacer *strings.Replacer
var emptyReplacer *strings.Replacer

func replace(s string) string {
	if Enabled {
		return ansiReplacer.Replace(s)
	}
	return emptyReplacer.Replace(s)
}

func replaceArgs(a []any) []any {
	for i, arg := range a {
		s, ok := arg.(string)
		if !ok {
			continue
		}
		a[i] = replace(s)
	}
	return a
}

// Fprintf formats according to a format specifier and writes to w.
// It returns the number of bytes written and any write error encountered.
func Fprintf(w io.Writer, format string, a ...any) (n int, err error) {
	return fmt.Fprint(w, Sprintf(format, a...))
}

// Printf formats according to a format specifier and writes to standard output.
// It returns the number of bytes written and any write error encountered.
func Printf(format string, a ...any) (n int, err error) {
	return fmt.Print(Sprintf(format, a...))
}

// Sprintf formats according to a format specifier and returns the resulting string.
func Sprintf(format string, a ...any) string {
	return replace(fmt.Sprintf(format, a...))
}

// Fprint formats using the default formats for its operands and writes to w.
// Spaces are added between operands when neither is a string.
// It returns the number of bytes written and any write error encountered.
func Fprint(w io.Writer, a ...any) (n int, err error) {
	return fmt.Fprint(w, replaceArgs(a)...)
}

// Print formats using the default formats for its operands and writes to standard output.
// Spaces are added between operands when neither is a string.
// It returns the number of bytes written and any write error encountered.
func Print(a ...any) (n int, err error) {
	return fmt.Print(replaceArgs(a)...)
}

// Sprint formats using the default formats for its operands and returns the resulting string.
// Spaces are added between operands when neither is a string.
func Sprint(a ...any) string {
	return fmt.Sprint(replaceArgs(a)...)
}

// Fprintln formats using the default formats for its operands and writes to w.
// Spaces are always added between operands and a newline is appended.
// It returns the number of bytes written and any write error encountered.
func Fprintln(w io.Writer, a ...any) (n int, err error) {
	return fmt.Fprintln(w, replaceArgs(a)...)
}

// Println formats using the default formats for its operands and writes to standard output.
// Spaces are always added between operands and a newline is appended.
// It returns the number of bytes written and any write error encountered.
func Println(a ...any) (n int, err error) {
	return fmt.Println(replaceArgs(a)...)
}

// Sprintln formats using the default formats for its operands and returns the resulting string.
// Spaces are always added between operands and a newline is appended.
func Sprintln(a ...any) string {
	return fmt.Sprintln(replaceArgs(a)...)
}
