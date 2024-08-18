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

// Error describes an error that occurred during the execution of a Lox program.
// It can describe any error which can be attributed to a range of characters in the source code.
type Error struct {
	msg   string
	start token.Position
	end   token.Position
}

// NewError creates a [*Error].
// The start and end positions are the range of characters in the source code that the error applies to.
// The error message is constructed from the given format string and arguments, as in [fmt.Sprintf].
func NewError(start token.Position, end token.Position, format string, args ...any) error {
	return &Error{
		msg:   fmt.Sprintf(format, args...),
		start: start,
		end:   end,
	}
}

// NewErrorFromToken creates a [*Error] which describes a problem with the given [token.Token].
func NewErrorFromToken(tok token.Token, format string, args ...any) error {
	return NewError(tok.Start, tok.End, format, args...)
}

// NewErrorFromNode creates a [*Error] which describes a problem with the given [ast.Node].
func NewErrorFromNode(node ast.Node, format string, args ...any) error {
	return NewError(node.Start(), node.End(), format, args...)
}

// NewErrorFromNodeRange creates a [*Error] which describes a problem with the range of characters that the given
// [ast.Node] cover.
func NewErrorFromNodeRange(start, end ast.Node, format string, args ...any) error {
	return NewError(start.Start(), end.End(), format, args...)
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
	bold := color.New(color.Bold)
	red := color.New(color.FgRed)

	var b strings.Builder
	buildString := func() string {
		return strings.TrimSuffix(b.String(), "\n")
	}

	bold.Fprint(&b, e.start, ": ", red.Sprint("error: "), e.msg, "\n")

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
	fmt.Fprintln(&b, lines[0])
	if e.start == e.end {
		// There's nothing to highlight
		return buildString()
	}

	if len(lines) == 1 {
		fmt.Fprint(&b, strings.Repeat(" ", runewidth.StringWidth(string(lines[0][:e.start.Column]))))
		red.Fprintln(&b, strings.Repeat("~", runewidth.StringWidth(string(lines[0][e.start.Column:e.end.Column]))))
	} else {
		fmt.Fprint(&b, strings.Repeat(" ", runewidth.StringWidth(string(lines[0][:e.start.Column]))))
		red.Fprintln(&b, strings.Repeat("~", runewidth.StringWidth(string(lines[0][e.start.Column:]))))
		for _, line := range lines[1 : len(lines)-1] {
			fmt.Fprintln(&b, string(line))
			red.Fprintln(&b, strings.Repeat("~", runewidth.StringWidth(string(line))))
		}
		if lastLine := lines[len(lines)-1]; len(lastLine) > 0 {
			fmt.Fprintln(&b, string(lastLine))
			red.Fprintln(&b, strings.Repeat("~", runewidth.StringWidth(string(lastLine[:e.end.Column]))))
		}
	}

	return buildString()
}

// Errors is a list of [*Error]s.
type Errors []*Error

// Add adds a [*Error] to the list of errors.
// The parameters are the same as for [NewError].
func (e *Errors) Add(start token.Position, end token.Position, format string, args ...any) {
	*e = append(*e, &Error{
		msg:   fmt.Sprintf(format, args...),
		start: start,
		end:   end,
	})
}

// AddFromToken adds a [*Error] to the list of errors.
// The parameters are the same as for [NewErrorFromToken].
func (e *Errors) AddFromToken(tok token.Token, format string, args ...any) {
	e.Add(tok.Start, tok.End, format, args...)
}

// AddFromNode adds a [*Error] to the list of errors.
// The parameters are the same as for [NewErrorFromNode].
func (e *Errors) AddFromNode(node ast.Node, format string, args ...any) {
	e.Add(node.Start(), node.End(), format, args...)
}

// AddFromNodeRange adds a [*Error] to the list of errors.
// The parameters are the same as for [NewErrorFromNodeRange].
func (e *Errors) AddFromNodeRange(start, end ast.Node, format string, args ...any) {
	e.Add(start.Start(), end.End(), format, args...)
}

// Err orders the errors in the list by their position in the source code and returns them as a single error.
func (e Errors) Err() error {
	if len(e) == 0 {
		return nil
	}
	slices.SortFunc([]*Error(e), func(e1, e2 *Error) int {
		return e1.start.Compare(e2.start)
	})
	var errs []error
	for _, err := range e {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
