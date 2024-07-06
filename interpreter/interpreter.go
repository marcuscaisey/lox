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
	replMode  bool
}

// Option can be passed to New to configure the interpreter.
type Option func(*Interpreter)

// REPLMode sets the interpreter to REPL mode.
// In REPL mode, the interpreter will print the result of expression statements.
func REPLMode() Option {
	return func(i *Interpreter) {
		i.replMode = true
	}
}

// New constructs a new Interpreter with the given options.
func New(opts ...Option) *Interpreter {
	interpreter := &Interpreter{
		globalEnv: newEnvironment(),
	}
	for _, opt := range opts {
		opt(interpreter)
	}
	return interpreter
}

// Interpret interprets a program and returns an error if one occurred.
// Interpret can be called multiple times with different ASTs and the state will be maintained between calls.
func (i *Interpreter) Interpret(program ast.Program) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if runtimeErr, ok := r.(*runtimeError); ok {
				err = runtimeErr
			} else {
				panic(r)
			}
		}
	}()
	i.interpretProgram(i.globalEnv, program)
	return nil
}

func (i *Interpreter) interpretProgram(env *environment, node ast.Program) {
	for _, stmt := range node.Stmts {
		i.interpretStmt(env, stmt)
	}
}

type stmtResult int

const (
	stmtResultNone stmtResult = iota
	stmtResultBreak
	stmtResultContinue
)

func (i *Interpreter) interpretStmt(env *environment, stmt ast.Stmt) stmtResult {
	switch stmt := stmt.(type) {
	case ast.VarDecl:
		i.interpretVarDecl(env, stmt)
	case ast.ExprStmt:
		i.interpretExprStmt(env, stmt)
	case ast.PrintStmt:
		i.interpretPrintStmt(env, stmt)
	case ast.BlockStmt:
		return i.interpretBlockStmt(env, stmt)
	case ast.IfStmt:
		return i.interpretIfStmt(env, stmt)
	case ast.WhileStmt:
		return i.interpretWhileStmt(env, stmt)
	case ast.ForStmt:
		return i.interpretForStmt(env, stmt)
	case ast.BreakStmt:
		return i.interpretBreakStmt()
	case ast.ContinueStmt:
		return i.interpretContinueStmt()
	default:
		panic(fmt.Sprintf("unexpected statement type: %T", stmt))
	}
	return stmtResultNone
}

func (i *Interpreter) interpretVarDecl(env *environment, stmt ast.VarDecl) {
	var value loxObject
	if stmt.Initialiser != nil {
		value = i.interpretExpr(env, stmt.Initialiser)
	}
	env.Define(stmt.Name, value)
}

func (i *Interpreter) interpretExprStmt(env *environment, stmt ast.ExprStmt) {
	value := i.interpretExpr(env, stmt.Expr)
	if i.replMode {
		fmt.Println(value.String())
	}
}

func (i *Interpreter) interpretPrintStmt(env *environment, stmt ast.PrintStmt) {
	value := i.interpretExpr(env, stmt.Expr)
	fmt.Println(value.String())
}

func (i *Interpreter) interpretBlockStmt(env *environment, stmt ast.BlockStmt) stmtResult {
	childEnv := env.Child()
	for _, stmt := range stmt.Stmts {
		if result := i.interpretStmt(childEnv, stmt); result != stmtResultNone {
			return result
		}
	}
	return stmtResultNone
}

func (i *Interpreter) interpretIfStmt(env *environment, stmt ast.IfStmt) stmtResult {
	condition := i.interpretExpr(env, stmt.Condition)
	if condition.IsTruthy() {
		return i.interpretStmt(env, stmt.Then)
	} else if stmt.Else != nil {
		return i.interpretStmt(env, stmt.Else)
	} else {
		return stmtResultNone
	}
}

func (i *Interpreter) interpretWhileStmt(env *environment, stmt ast.WhileStmt) stmtResult {
	for i.interpretExpr(env, stmt.Condition).IsTruthy() {
		switch i.interpretStmt(env, stmt.Body) {
		case stmtResultBreak:
			return stmtResultNone
		}
	}
	return stmtResultNone
}

func (i *Interpreter) interpretForStmt(env *environment, stmt ast.ForStmt) stmtResult {
	childEnv := env.Child()
	if stmt.Initialise != nil {
		i.interpretStmt(childEnv, stmt.Initialise)
	}
	for stmt.Condition == nil || i.interpretExpr(childEnv, stmt.Condition).IsTruthy() {
		switch i.interpretStmt(childEnv, stmt.Body) {
		case stmtResultBreak:
			return stmtResultNone
		}
		if stmt.Update != nil {
			i.interpretExpr(childEnv, stmt.Update)
		}
	}
	return stmtResultNone
}

func (i *Interpreter) interpretBreakStmt() stmtResult {
	return stmtResultBreak
}

func (i *Interpreter) interpretContinueStmt() stmtResult {
	return stmtResultContinue
}

func (i *Interpreter) interpretExpr(env *environment, expr ast.Expr) loxObject {
	switch expr := expr.(type) {
	case ast.GroupExpr:
		return i.interpretExpr(env, expr.Expr)
	case ast.LiteralExpr:
		return i.interpretLiteralExpr(expr)
	case ast.VariableExpr:
		return i.interpretVariableExpr(env, expr)
	case ast.UnaryExpr:
		return i.interpretUnaryExpr(env, expr)
	case ast.BinaryExpr:
		return i.interpretBinaryExpr(env, expr)
	case ast.TernaryExpr:
		return i.interpretTernaryExpr(env, expr)
	case ast.AssignmentExpr:
		return i.interpretAssignmentExpr(env, expr)
	default:
		panic(fmt.Sprintf("unexpected expression type: %T", expr))
	}
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
	right := i.interpretExpr(env, expr.Right)
	if expr.Op.Type == token.Bang {
		// The behaviour of ! is independent of the type of the operand, so we can implement it here.
		return !right.IsTruthy()
	}
	return right.UnaryOp(expr.Op)
}

func (i *Interpreter) interpretBinaryExpr(env *environment, expr ast.BinaryExpr) loxObject {
	left := i.interpretExpr(env, expr.Left)

	// We check for short-circuiting operators first.
	switch expr.Op.Type {
	case token.Or:
		// The behaviour of or is independent of the types of the operands, so we can implement it here.
		if left.IsTruthy() {
			return left
		} else {
			return i.interpretExpr(env, expr.Right)
		}
	case token.And:
		// The behaviour of and is independent of the types of the operands, so we can implement it here.
		if !left.IsTruthy() {
			return left
		} else {
			return i.interpretExpr(env, expr.Right)
		}
	}

	right := i.interpretExpr(env, expr.Right)
	switch expr.Op.Type {
	case token.Comma:
		// The , operator evaluates both operands and returns the value of the right operand.
		// It's behavior is independent of the types of the operands, so we can implement it here.
		return right
	case token.EqualEqual:
		// The behaviour of == is independent of the types of the operands, so we can implement it here.
		return loxBool(left == right)
	case token.BangEqual:
		// The behaviour of != is independent of the types of the operands, so we can implement it here.
		return loxBool(left != right)
	default:
		return left.BinaryOp(expr.Op, right)
	}
}

func (i *Interpreter) interpretTernaryExpr(env *environment, expr ast.TernaryExpr) loxObject {
	condition := i.interpretExpr(env, expr.Condition)
	if condition.IsTruthy() {
		return i.interpretExpr(env, expr.Then)
	}
	return i.interpretExpr(env, expr.Else)
}

func (i *Interpreter) interpretAssignmentExpr(env *environment, expr ast.AssignmentExpr) loxObject {
	value := i.interpretExpr(env, expr.Right)
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
