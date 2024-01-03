// Package interpreter defines the interpreter for the language.
package interpreter

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/fatih/color"
	"github.com/lithammer/dedent"

	"github.com/marcuscaisey/golox/ast"
	"github.com/marcuscaisey/golox/token"
)

// runtimeError is basically the same as syntaxError from the parser package. They'll probably diverge in the future
// though.
type runtimeError struct {
	tok token.Token
	msg string
}

func (e *runtimeError) Error() string {
	// If the token spans multiple lines, only show the first one. I'm not sure what the best way of highlighting and
	// pointing to a multi-line token is.
	tok, _, _ := strings.Cut(e.tok.String(), "\n")
	line := e.tok.Position.File.Line(e.tok.Position.Line)
	data := map[string]any{
		"pos":    e.tok.Position,
		"msg":    e.msg,
		"before": line[:e.tok.Position.Column-1],
		"tok":    tok,
		"after":  line[e.tok.Position.Column+len(tok)-1:],
	}
	funcs := template.FuncMap{
		"red":    color.New(color.FgRed).SprintFunc(),
		"bold":   color.New(color.Bold).SprintFunc(),
		"repeat": strings.Repeat,
	}
	text := strings.TrimSpace(dedent.Dedent(`
		{{ .pos }}: runtime error: {{ .msg }}
		{{ .before }}{{ .tok | bold | red }}{{ .after }}
		{{ repeat " " (len .before) }}{{ repeat "^" (len .tok) | red | bold }}
	`))

	tmpl := template.Must(template.New("").Funcs(funcs).Parse(strings.TrimSpace(dedent.Dedent(text))))
	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		panic(err)
	}
	return b.String()
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
