// Package lox provides types which are shared by most of the packages in the Lox interpreter.
package lox

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/token"
)

// ErrorRange is a function which returns the range of characters that an error applies to.
type ErrorRange func() (start, end token.Position)

// FromToken returns an [ErrorRange] which describes the range of characters in a token.
func FromToken(tok token.Token) ErrorRange {
	return func() (start, end token.Position) {
		return tok.Start, tok.End
	}
}

// FromTokens returns an [ErrorRange] which describes the range of characters between two tokens.
func FromTokens(start, end token.Token) ErrorRange {
	return func() (star, en token.Position) {
		return start.Start, end.End
	}
}

// FromNode returns an [ErrorRange] which describes the range of characters in a node.
func FromNode(node ast.Node) ErrorRange {
	return func() (start, end token.Position) {
		return node.Start(), node.End()
	}
}

// FromNodes returns an [ErrorRange] which describes the range of characters between two nodes.
func FromNodes(start, end ast.Node) ErrorRange {
	return func() (star, en token.Position) {
		return start.Start(), end.End()
	}
}

// Error describes an error that occurred during the execution of a Lox program.
// It can describe any error which can be attributed to a range of characters in the source code.
type Error struct {
	Msg   string
	Start token.Position
	End   token.Position
}

// NewError creates a [*Error] with the given message and range.
func NewError(rang ErrorRange, message string) error {
	return NewErrorf(rang, "%s", message)
}

// NewErrorf creates a [*Error].
// The error message is constructed from the given format string and arguments, as in [fmt.Sprintf].
func NewErrorf(rang ErrorRange, format string, args ...any) error {
	e := &Error{
		Msg: fmt.Sprintf(format, args...),
	}
	e.Start, e.End = rang()
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
func (e *Errors) Add(rang ErrorRange, message string) {
	*e = append(*e, NewError(rang, message).(*Error))
}

// Addf adds a [*Error] to the list of errors.
// The parameters are the same as for [NewErrorf].
func (e *Errors) Addf(rang ErrorRange, format string, args ...any) {
	*e = append(*e, NewErrorf(rang, format, args...).(*Error))
}

// Err orders the errors in the list by their position in the source code and returns them as a single error.
func (e Errors) Err() error {
	if len(e) == 0 {
		return nil
	}
	slices.SortFunc([]*Error(e), func(e1, e2 *Error) int {
		return e1.Start.Compare(e2.Start)
	})
	var errs []error
	for _, err := range e {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
