// Package loxerror defines [LoxError] which is the main error type used in golox code.
package loxerror

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/token"
)

// LoxError describes an error that occurred during the execution of a Lox program.
// It can describe any error which can be attributed to a range of characters in the source code.
type LoxError struct {
	msg   string
	start token.Position
	end   token.Position
}

// New creates a [*LoxError].
// The start and end positions are the range of characters in the source code that the error applies to.
// The error message is constructed from the given format string and arguments, as in [fmt.Sprintf].
func New(start token.Position, end token.Position, format string, args ...any) *LoxError {
	return &LoxError{
		msg:   fmt.Sprintf(format, args...),
		start: start,
		end:   end,
	}
}

// NewFromToken creates a [*LoxError] which describes a problem with the given [token.Token].
func NewFromToken(tok token.Token, format string, args ...interface{}) *LoxError {
	return New(tok.Start, tok.End, format, args...)
}

// NewFromNode creates a [*LoxError] which describes a problem with the given [ast.Node].
func NewFromNode(node ast.Node, format string, args ...interface{}) *LoxError {
	return New(node.Start(), node.End(), format, args...)
}

// NewFromNodeRange creates a [*LoxError] which describes a problem with the range of characters that the given
// [ast.Node] cover.
func NewFromNodeRange(start, end ast.Node, format string, args ...interface{}) *LoxError {
	return New(start.Start(), end.End(), format, args...)
}

// Error formats the error by displaying the error message and highlighting the range of characters in the source code
// that the error applies to.
//
// For example:
//
//	test.lox:2:7: error: unterminated string literal
//	print "bar;
//	      ~~~~~
func (e *LoxError) Error() string {
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
