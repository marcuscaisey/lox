package lox

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"

	"github.com/marcuscaisey/lox/lox/ansi"
	"github.com/marcuscaisey/lox/lox/token"
)

// Error describes an error that occurred during the execution of a Lox program.
// It can describe any error which can be attributed to a range of characters in the source code.
type Error struct {
	Msg   string
	Start token.Position
	End   token.Position
}

// NewError creates a [*Error] with the given message and range.
func NewError(charRange token.CharacterRange, message string) error {
	return NewErrorf(charRange, "%s", message)
}

// NewErrorf creates a [*Error].
// The error message is constructed from the given format string and arguments, as in [fmt.Sprintf].
func NewErrorf(charRange token.CharacterRange, format string, args ...any) error {
	e := &Error{
		Msg: fmt.Sprintf(format, args...),
	}
	e.Start = charRange.Start()
	e.End = charRange.End()
	return e

}

// Error formats the error by displaying the error message and highlighting the range of characters in the source code
// that the error applies to.
//
// For example:
//
//	test.lox:2:7: error: unterminated string literal
//	print "bar;
//	      ~~~~~
func (e *Error) Error() string {
	var b strings.Builder
	buildString := func() string {
		return strings.TrimSuffix(b.String(), "\n")
	}

	ansi.Fprintf(&b, "${BOLD}%m: ${RED}error${DEFAULT}: %s${DEFAULT}${RESET_BOLD}\n", e.Start, e.Msg)

	lines := make([]string, e.End.Line-e.Start.Line+1)
	for i := e.Start.Line; i <= e.End.Line; i++ {
		line := e.Start.File.Line(i)
		if !utf8.Valid(line) {
			// If any of the lines are not valid UTF-8 then we can't display the source code, so just return the error
			// message on its own. This is a very rare case and it's not worth the effort to handle it any better.
			return buildString()
		}
		lines[i-e.Start.Line] = string(line)
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
	if e.Start == e.End {
		// There's nothing to highlight
		return buildString()
	}

	if len(lines) == 1 {
		printLineHighlight(lines[0], e.Start.Column, e.End.Column)
	} else {
		printLineHighlight(lines[0], e.Start.Column, len(lines[0]))
		for _, line := range lines[1 : len(lines)-1] {
			printLine(line)
			printLineHighlight(line, 0, len(line))
		}
		if lastLine := lines[len(lines)-1]; len(lastLine) > 0 {
			printLine(lastLine)
			printLineHighlight(lastLine, 0, e.End.Column)
		}
	}

	return buildString()
}

// Errors is a list of [*Error]s.
type Errors []*Error

// Add adds a [*Error] to the list of errors.
// The parameters are the same as for [NewError].
func (e *Errors) Add(charRange token.CharacterRange, message string) {
	*e = append(*e, NewError(charRange, message).(*Error))
}

// Addf adds a [*Error] to the list of errors.
// The parameters are the same as for [NewErrorf].
func (e *Errors) Addf(charRange token.CharacterRange, format string, args ...any) {
	*e = append(*e, NewErrorf(charRange, format, args...).(*Error))
}

// Sort sorts the errors by their start position.
func (e Errors) Sort() {
	slices.SortFunc(e, func(e1, e2 *Error) int {
		return e1.Start.Compare(e2.Start)
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
