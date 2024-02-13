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
	globalEnv *environment
}

// New constructs a new Interpreter.
func New() *Interpreter {
	return &Interpreter{
		globalEnv: newEnvironment(nil),
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
	i.interpret(i.globalEnv, node)
	return nil
}

func (i *Interpreter) interpret(env *environment, node ast.Node) loxObject {
	switch node := node.(type) {
	case ast.Program:
		return i.interpretProgram(env, node)
	case ast.VarDecl:
		return i.interpretVarDecl(env, node)
	case ast.BlockStmt:
		return i.interpretBlockStmt(env, node)
	case ast.ExprStmt:
		return i.interpretExprStmt(env, node)
	case ast.PrintStmt:
		return i.interpretPrintStmt(env, node)
	case ast.GroupExpr:
		return i.interpret(env, node.Expr)
	case ast.LiteralExpr:
		return i.interpretLiteralExpr(node)
	case ast.VariableExpr:
		return i.interpretVariableExpr(env, node)
	case ast.UnaryExpr:
		return i.interpretUnaryExpr(env, node)
	case ast.BinaryExpr:
		return i.interpretBinaryExpr(env, node)
	case ast.TernaryExpr:
		return i.interpretTernaryExpr(env, node)
	case ast.AssignmentExpr:
		return i.interpretAssignmentExpr(env, node)
	default:
		panic(fmt.Sprintf("unexpected node type: %T", node))
	}
}

func (i *Interpreter) interpretProgram(env *environment, node ast.Program) loxObject {
	var result loxObject = loxNil{}
	for _, stmt := range node.Stmts {
		result = i.interpret(env, stmt)
	}
	return result
}

func (i *Interpreter) interpretVarDecl(env *environment, stmt ast.VarDecl) loxObject {
	var value loxObject = loxNil{}
	if stmt.Initialiser != nil {
		value = i.interpret(env, stmt.Initialiser)
	}
	env.Define(stmt.Name, value)
	return loxNil{}
}

func (i *Interpreter) interpretBlockStmt(env *environment, stmt ast.BlockStmt) loxObject {
	blockEnv := newEnvironment(env)
	for _, stmt := range stmt.Stmts {
		i.interpret(blockEnv, stmt)
	}
	return loxNil{}
}

func (i *Interpreter) interpretExprStmt(env *environment, stmt ast.ExprStmt) loxObject {
	i.interpret(env, stmt.Expr)
	return loxNil{}
}

func (i *Interpreter) interpretPrintStmt(env *environment, stmt ast.PrintStmt) loxObject {
	value := i.interpret(env, stmt.Expr)
	fmt.Println(value.String())
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

func (i *Interpreter) interpretVariableExpr(env *environment, expr ast.VariableExpr) loxObject {
	return env.Get(expr.Name)
}

func (i *Interpreter) interpretUnaryExpr(env *environment, expr ast.UnaryExpr) loxObject {
	right := i.interpret(env, expr.Right)
	if expr.Op.Type == token.Bang {
		// The behaviour of ! is independent of the type of the operand, so we can implement it here.
		return !right.IsTruthy()
	}
	return right.UnaryOp(expr.Op)
}

func (i *Interpreter) interpretBinaryExpr(env *environment, expr ast.BinaryExpr) loxObject {
	left := i.interpret(env, expr.Left)
	right := i.interpret(env, expr.Right)
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

func (i *Interpreter) interpretTernaryExpr(env *environment, expr ast.TernaryExpr) loxObject {
	condition := i.interpret(env, expr.Condition)
	if condition.IsTruthy() {
		return i.interpret(env, expr.Then)
	}
	return i.interpret(env, expr.Else)
}

func (i *Interpreter) interpretAssignmentExpr(env *environment, expr ast.AssignmentExpr) loxObject {
	value := i.interpret(env, expr.Right)
	env.Assign(expr.Left, value)
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
