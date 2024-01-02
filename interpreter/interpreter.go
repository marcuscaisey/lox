// Package interpreter defines the interpreter for the language.
package interpreter

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/marcuscaisey/golox/ast"
	"github.com/marcuscaisey/golox/token"
)

var ansiCodes = map[string]string{
	"RESET":   "\x1b[0m",
	"BOLD":    "\x1b[1m",
	"RED":     "\x1b[31m",
	"DEFAULT": "\x1b[39m",
}

var isTerminal = term.IsTerminal(int(os.Stderr.Fd()))

// runtimeError is basically the same as syntaxError from the parser package. They'll probably diverge in the future
// though.
type runtimeError struct {
	tok token.Token
	msg string
}

func (e *runtimeError) Error() string {
	// Example output:
	// 1:3: runtime error: + operator cannot be used with a number and a string
	// 1 + "foo"
	//   ^
	// TODO: It would be nice to highlight more than just the token that caused the error. I.e. highlight the whole of
	// 1 + "foo" in the example above. This requires adding more information to the AST nodes.
	var b strings.Builder
	line := e.tok.Position.File.Line(e.tok.Position.Line)
	col := e.tok.Position.Column
	before := line[:col-1]
	// If the literal contains a newline, only show the first line. This is a bit hacky but it's good enough for now.
	lit, _, _ := strings.Cut(e.tok.String(), "\n")
	after := line[col+len(lit)-1:]
	fmt.Fprintf(&b, "${BOLD}%s: runtime error: %s${RESET}\n", e.tok.Position, e.msg)
	fmt.Fprintf(&b, "%s${RED}%s${RESET}%s\n", before, lit, after)
	fmt.Fprintf(&b, "${BOLD}${RED}%*s${RESET}", col, strings.Repeat("^", len(lit)))
	msg := b.String()
	for k, v := range ansiCodes {
		if !isTerminal {
			v = ""
		}
		msg = strings.ReplaceAll(msg, fmt.Sprintf("${%s}", k), v)
	}
	return msg
}

// Interpret interprets an AST and returns the result.
func Interpret(node ast.Node) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if runtimeErr, ok := r.(*runtimeError); ok {
				err = runtimeErr
			} else {
				panic(r)
			}
		}
	}()
	result := interpret(node)
	fmt.Println(result)
	return nil
}

func interpret(node ast.Node) loxObject {
	switch node := node.(type) {
	case ast.GroupExpr:
		return interpret(node.Expr)
	case ast.LiteralExpr:
		return interpretLiteralExpr(node)
	case ast.UnaryExpr:
		return interpretUnaryExpr(node)
	case ast.BinaryExpr:
		return interpretBinaryExpr(node)
	case ast.TernaryExpr:
		return interpretTernaryExpr(node)
	default:
		panic(fmt.Sprintf("unexpected node type: %T", node))
	}
}

func interpretLiteralExpr(expr ast.LiteralExpr) loxObject {
	switch value := expr.Value.(type) {
	case float64:
		return loxNumber(value)
	case string:
		return loxString(value)
	case bool:
		return loxBool(value)
	case nil:
		return loxNil{}
	default:
		panic(fmt.Sprintf("unexpected literal type: %T", value))
	}
}

func interpretUnaryExpr(expr ast.UnaryExpr) loxObject {
	right := interpret(expr.Right)
	return right.UnaryOp(expr.Op)
}

func interpretBinaryExpr(expr ast.BinaryExpr) loxObject {
	left := interpret(expr.Left)
	right := interpret(expr.Right)
	switch expr.Op.Type {
	case token.Comma:
		// The , operator evaluates both operands and returns the value of the right operand.
		// It's behavior is independent of the types of the operands, so we can implement it here.
		return right
	case token.Equal:
		// The behaviour of == is independent of the types of the operands, so we can implement it here.
		return loxBool(left == right)
	case token.NotEqual:
		// The behaviour of != is independent of the types of the operands, so we can implement it here.
		return loxBool(left != right)
	default:
		return left.BinaryOp(expr.Op, right)
	}
}

func interpretTernaryExpr(expr ast.TernaryExpr) loxObject {
	condition := interpret(expr.Condition)
	if condition.IsTruthy() {
		return interpret(expr.Then)
	}
	return interpret(expr.Else)
}
