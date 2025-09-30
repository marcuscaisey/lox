// Package loxerr defines the type which describes an error during execution of a Lox program.
package loxerr

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"

	"github.com/marcuscaisey/lox/golox/ansi"
	"github.com/marcuscaisey/lox/golox/token"
)

// Type is the type of an [Error].
type Type int

const (
	// Fatal errors cause execution of the program to fail. For example, a parser error or a division by zero.
	Fatal Type = iota
	// NonFatal errors don't cause execution of the program to fail. For example, a variable has been declared but
	// not used.
	NonFatal
)

// Error describes an error that occurred during the execution of a Lox program.
// It can describe any error which can be attributed to a range of characters in the source code.
type Error struct {
	Type  Type
	Msg   string
	start token.Position
	end   token.Position
}

// Newf creates a [*Error].
// The error message is constructed from the given format string and arguments, as in [fmt.Sprintf].
func Newf(rang token.Range, typ Type, format string, args ...any) error {
	return newf(rang.Start(), rang.End(), typ, format, args...)
}

// NewSpanningRangesf creates an [*Error] which spans the given [token.Range]s.
// The error message is constructed from the given format string and arguments, as in [fmt.Sprintf].
func NewSpanningRangesf(start, end token.Range, typ Type, message string, args ...any) error {
	return newf(start.Start(), end.End(), typ, message, args...)
}

func newf(start, end token.Position, typ Type, format string, args ...any) error {
	return &Error{
		Type:  typ,
		Msg:   fmt.Sprintf(format, args...),
		start: start,
		end:   end,
	}
}

// Start returns the position of the first character in the range affected by the error.
func (e *Error) Start() token.Position {
	return e.start
}

// End returns the position immediately after the last character in the range affected by the error.
func (e *Error) End() token.Position {
	return e.end
}

// Error formats the error by displaying the error message and highlighting the range of characters in the source code
// that the error applies to.
//
// For example:
//
//	2:7: error: unterminated string literal
//	print "bar;
//	      ~~~~~
func (e *Error) Error() string {
	var b strings.Builder
	buildString := func() string {
		return strings.TrimSuffix(b.String(), "\n")
	}

	class := "${RED}error"
	if e.Type != Fatal {
		class = "${BLUE}hint"
	}
	ansi.Fprintf(&b, "${BOLD}%m: "+class+"${DEFAULT}: %s${DEFAULT}${RESET_BOLD}\n", e.start, e.Msg)

	lines := make([]string, e.end.Line-e.start.Line+1)
	for i := e.start.Line; i <= e.end.Line; i++ {
		line := e.start.File.Line(i)
		if !utf8.Valid(line) {
			// If any of the lines are not valid UTF-8 then we can't display the source code, so just return the error
			// message on its own. This is a very rare case and it's not worth the effort to handle it any better.
			return buildString()
		}
		lines[i-e.start.Line] = string(line)
	}

	printLine := func(line string) {
		ansi.Fprint(&b, "${FAINT}", line, "${RESET_BOLD}\n")
	}
	printLineHighlight := func(line string, start, end int) {
		leadingWhitespace := strings.Repeat(" ", runewidth.StringWidth(line[:start]))
		tildes := strings.Repeat("~", runewidth.StringWidth(line[start:end]))
		ansi.Fprint(&b, leadingWhitespace, "${FAINT}${RED}", tildes, "${DEFAULT}${RESET_BOLD}\n")
	}

	printLine(lines[0])
	if e.start == e.end {
		// There's nothing to highlight
		return buildString()
	}

	if len(lines) == 1 {
		printLineHighlight(lines[0], e.start.Column, e.end.Column)
	} else {
		printLineHighlight(lines[0], e.start.Column, len(lines[0]))
		for _, line := range lines[1 : len(lines)-1] {
			printLine(line)
			printLineHighlight(line, 0, len(line))
		}
		if lastLine := lines[len(lines)-1]; len(lastLine) > 0 {
			printLine(lastLine)
			printLineHighlight(lastLine, 0, e.end.Column)
		}
	}

	return buildString()
}

// Errors is a list of [*Error]s.
type Errors []*Error

// Addf adds a [*Error] to the list of errors.
// The parameters are the same as for [Newf].
func (e *Errors) Addf(rang token.Range, typ Type, format string, args ...any) {
	*e = append(*e, Newf(rang, typ, format, args...).(*Error))
}

// AddSpanningRangesf adds an [*Error] to the list of errors.
// The parameters are the same as for [NewSpanningRangesf].
func (e *Errors) AddSpanningRangesf(start, end token.Range, typ Type, format string, args ...any) {
	*e = append(*e, NewSpanningRangesf(start, end, typ, format, args...).(*Error))
}

// Fatal returns a new [Errors] containing only the errors in the list which are fatal.
func (e Errors) Fatal() Errors {
	errs := make(Errors, 0, len(e))
	for _, err := range e {
		if err.Type == Fatal {
			errs = append(errs, err)
		}
	}
	return errs
}

// NonFatal returns a new [Errors] containing only the errors in the list which are not fatal.
func (e Errors) NonFatal() Errors {
	errs := make(Errors, 0, len(e))
	for _, err := range e {
		if err.Type != Fatal {
			errs = append(errs, err)
		}
	}
	return errs
}

// Sort sorts the errors by their start position.
func (e Errors) Sort() {
	slices.SortFunc(e, func(e1, e2 *Error) int {
		return e1.start.Compare(e2.start)
	})
}

// Error formats the errors by concatenating their messages after sorting them by their start position.
func (e Errors) Error() string {
	if len(e) == 0 {
		panic("Error called on empty error list")
	}
	e.Sort()
	msgs := make([]string, len(e))
	for i, err := range e {
		msgs[i] = err.Error()
	}
	return strings.Join(msgs, "\n")
}

// Err returns the error list unchanged if its non-empty, otherwise nil.
// This should be used to return an [Errors] from a function as an [error] so that it becomes an untyped nil if there
// are no errors.
func (e Errors) Err() error {
	if len(e) == 0 {
		return nil
	}
	return e
}
