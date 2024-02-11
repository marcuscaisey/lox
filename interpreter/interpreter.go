// Package interpreter defines the interpreter for the language.
package interpreter

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
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
	switch tok := expr.Value; tok.Type {
	case token.Number:
		value, err := strconv.ParseFloat(tok.Literal, 64)
		if err != nil {
			panic(fmt.Sprintf("unexpected error parsing number literal: %s", err))
		}
		return loxNumber(value)
	case token.String:
		return loxString(tok.Literal[1 : len(tok.Literal)-1]) // Remove surrounding quotes
	case token.True, token.False:
		return loxBool(tok.Type == token.True)
	case token.Nil:
		return loxNil{}
	default:
		panic(fmt.Sprintf("unexpected literal type: %h", tok.Type))
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
	bold := color.New(color.Bold)
	red := color.New(color.FgRed)

	line := e.tok.Start.File.Line(e.tok.Start.Line)

	var b strings.Builder
	bold.Fprintln(&b, e.tok.Start, ": ", red.Sprint("runtime error: "), e.msg)
	fmt.Fprintln(&b, string(line))
	fmt.Fprint(&b, strings.Repeat(" ", runewidth.StringWidth(string(line[:e.tok.Start.Column]))))
	red.Fprint(&b, strings.Repeat("~", runewidth.StringWidth(string(line[e.tok.Start.Column:e.tok.End.Column]))))

	return b.String()
}
