package lox

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"

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

var (
	bold     = color.New(color.Bold)
	faint    = color.New(color.Faint)
	red      = color.New(color.FgRed)
	faintRed = color.New(color.Faint, color.FgRed)
)

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

	bold.Fprint(&b, fmt.Sprintf("%m", e.Start), ": ", red.Sprint("error"), ": ", e.Msg, "\n")

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
	faint.Fprintln(&b, lines[0])
	if e.Start == e.End {
		// There's nothing to highlight
		return buildString()
	}

	if len(lines) == 1 {
		fmt.Fprint(&b, strings.Repeat(" ", runewidth.StringWidth(string(lines[0][:e.Start.Column]))))
		faintRed.Fprintln(&b, strings.Repeat("~", runewidth.StringWidth(string(lines[0][e.Start.Column:e.End.Column]))))
	} else {
		fmt.Fprint(&b, strings.Repeat(" ", runewidth.StringWidth(string(lines[0][:e.Start.Column]))))
		faintRed.Fprintln(&b, strings.Repeat("~", runewidth.StringWidth(string(lines[0][e.Start.Column:]))))
		for _, line := range lines[1 : len(lines)-1] {
			faint.Fprintln(&b, string(line))
			faintRed.Fprintln(&b, strings.Repeat("~", runewidth.StringWidth(string(line))))
		}
		if lastLine := lines[len(lines)-1]; len(lastLine) > 0 {
			faint.Fprintln(&b, string(lastLine))
			faintRed.Fprintln(&b, strings.Repeat("~", runewidth.StringWidth(string(lastLine[:e.End.Column]))))
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

// Error formats the errors by concatenating their messages after sorting them by their start position.
func (e Errors) Error() string {
	if len(e) == 0 {
		panic("Error called on empty error list")
	}
	slices.SortFunc([]*Error(e), func(e1, e2 *Error) int {
		return e1.Start.Compare(e2.Start)
	})
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
