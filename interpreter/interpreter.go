// Package interpreter defines the interpreter for the language.
package interpreter

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/fatih/color"
	"github.com/lithammer/dedent"
	"github.com/mattn/go-runewidth"

	"github.com/marcuscaisey/golox/ast"
	"github.com/marcuscaisey/golox/token"
)

// Interpreter is the interpreter for the language.
type Interpreter struct {
	env *environment
}

// New constructs a new Interpreter.
func New() *Interpreter {
	return &Interpreter{
		env: newEnvironment(),
	}
}

// Interpret interprets an AST and returns the result.
// Interpret can be called multiple times with different ASTs and the state will be maintained between calls.
func (i *Interpreter) Interpret(node ast.Node) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if runtimeErr, ok := r.(*runtimeError); ok {
				err = runtimeErr
			} else {
				panic(r)
			}
		}
	}()
	i.interpret(node)
	return nil
}

func (i *Interpreter) interpret(node ast.Node) loxObject {
	switch node := node.(type) {
	case ast.Program:
		return i.interpretProgram(node)
	case ast.VarDecl:
		return i.interpretVarDecl(node)
	case ast.PrintStmt:
		return i.interpretPrintStmt(node)
	case ast.ExprStmt:
		return i.interpretExprStmt(node)
	case ast.GroupExpr:
		return i.interpret(node.Expr)
	case ast.LiteralExpr:
		return i.interpretLiteralExpr(node)
	case ast.VariableExpr:
		return i.interpretVariableExpr(node)
	case ast.UnaryExpr:
		return i.interpretUnaryExpr(node)
	case ast.BinaryExpr:
		return i.interpretBinaryExpr(node)
	case ast.TernaryExpr:
		return i.interpretTernaryExpr(node)
	case ast.AssignmentExpr:
		return i.interpretAssignmentExpr(node)
	default:
		panic(fmt.Sprintf("unexpected node type: %T", node))
	}
}

func (i *Interpreter) interpretProgram(node ast.Program) loxObject {
	var result loxObject = loxNil{}
	for _, stmt := range node.Stmts {
		result = i.interpret(stmt)
	}
	return result
}

func (i *Interpreter) interpretVarDecl(stmt ast.VarDecl) loxObject {
	var value loxObject = loxNil{}
	if stmt.Initialiser != nil {
		value = i.interpret(stmt.Initialiser)
	}
	i.env.Define(stmt.Name, value)
	return loxNil{}
}

func (i *Interpreter) interpretPrintStmt(stmt ast.PrintStmt) loxObject {
	value := i.interpret(stmt.Expr)
	fmt.Println(value.String())
	return loxNil{}
}

func (i *Interpreter) interpretExprStmt(stmt ast.ExprStmt) loxObject {
	i.interpret(stmt.Expr)
	return loxNil{}
}

func (i *Interpreter) interpretLiteralExpr(expr ast.LiteralExpr) loxObject {
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

func (i *Interpreter) interpretVariableExpr(expr ast.VariableExpr) loxObject {
	return i.env.Get(expr.Name)
}

func (i *Interpreter) interpretUnaryExpr(expr ast.UnaryExpr) loxObject {
	right := i.interpret(expr.Right)
	if expr.Op.Type == token.Bang {
		// The behaviour of ! is independent of the type of the operand, so we can implement it here.
		return !right.IsTruthy()
	}
	return right.UnaryOp(expr.Op)
}

func (i *Interpreter) interpretBinaryExpr(expr ast.BinaryExpr) loxObject {
	left := i.interpret(expr.Left)
	right := i.interpret(expr.Right)
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

func (i *Interpreter) interpretTernaryExpr(expr ast.TernaryExpr) loxObject {
	condition := i.interpret(expr.Condition)
	if condition.IsTruthy() {
		return i.interpret(expr.Then)
	}
	return i.interpret(expr.Else)
}

func (i *Interpreter) interpretAssignmentExpr(expr ast.AssignmentExpr) loxObject {
	value := i.interpret(expr.Right)
	i.env.Assign(expr.Left, value)
	return value
}

type runtimeError struct {
	tok token.Token
	msg string
}

func (e *runtimeError) Error() string {
	// If the token spans multiple lines, only show the first one. I'm not sure what the best way of pointing to a
	// multi-line token is.
	tok, _, _ := strings.Cut(e.tok.String(), "\n")
	line := e.tok.Position.File.Line(e.tok.Position.Line)
	data := map[string]any{
		"pos":    e.tok.Position,
		"msg":    e.msg,
		"before": string(line[:e.tok.Position.Column]),
		"tok":    tok,
		"after":  string(line[e.tok.Position.Column+len(tok):]),
	}
	funcs := template.FuncMap{
		"red":    color.New(color.FgRed).SprintFunc(),
		"bold":   color.New(color.Bold).SprintFunc(),
		"repeat": strings.Repeat,
		"stringWidth": func(s string) int {
			return runewidth.StringWidth(s)
		},
	}
	text := strings.TrimSpace(dedent.Dedent(`
		{{ .pos }}: runtime error: {{ .msg }}
		{{ .before }}{{ .tok | bold | red }}{{ .after }}
		{{ repeat " " (stringWidth .before) }}{{ repeat "^" (stringWidth .tok) | red | bold }}
	`))

	tmpl := template.Must(template.New("").Funcs(funcs).Parse(strings.TrimSpace(dedent.Dedent(text))))
	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		panic(err)
	}
	return b.String()
}
