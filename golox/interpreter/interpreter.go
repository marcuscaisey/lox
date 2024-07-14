// Package interpreter defines the interpreter for the language.
package interpreter

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/token"
)

type stmtResult interface {
	stmtResult()
}

type stmtResultNone struct{}

func (stmtResultNone) stmtResult() {}

type stmtResultBreak struct{}

func (stmtResultBreak) stmtResult() {}

type stmtResultContinue struct{}

func (stmtResultContinue) stmtResult() {}

type stmtResultReturn struct {
	Value loxObject
}

func (stmtResultReturn) stmtResult() {}

// Interpreter is the interpreter for the language.
type Interpreter struct {
	globals                   *environment
	localDeclDistancesByIdent map[token.Token]int
	replMode                  bool
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
	globals := newEnvironment()
	for name, fn := range builtinFns {
		globals.Set(name, fn)
	}
	interpreter := &Interpreter{
		globals: globals,
	}
	for _, opt := range opts {
		opt(interpreter)
	}
	return interpreter
}

// Interpret interprets a program and returns an error if one occurred.
// Interpret can be called multiple times with different ASTs and the state will be maintained between calls.
func (i *Interpreter) Interpret(program ast.Program, localDeclDistancesByIdent map[token.Token]int) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if runtimeErr, ok := r.(*runtimeError); ok {
				err = runtimeErr
			} else {
				panic(r)
			}
		}
	}()
	i.localDeclDistancesByIdent = localDeclDistancesByIdent
	i.interpretProgram(program)
	return nil
}

func (i *Interpreter) interpretProgram(node ast.Program) {
	for _, stmt := range node.Stmts {
		i.interpretStmt(i.globals, stmt)
	}
}

func (i *Interpreter) interpretStmt(env *environment, stmt ast.Stmt) stmtResult {
	switch stmt := stmt.(type) {
	case ast.VarDecl:
		i.interpretVarDecl(env, stmt)
	case ast.FunDecl:
		i.interpretFunDecl(env, stmt)
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
	case ast.ReturnStmt:
		return i.interpretReturnStmt(env, stmt)
	default:
		panic(fmt.Sprintf("unexpected statement type: %T", stmt))
	}
	return stmtResultNone{}
}

func (i *Interpreter) interpretVarDecl(env *environment, stmt ast.VarDecl) {
	var value loxObject
	if stmt.Initialiser != nil {
		value = i.interpretExpr(env, stmt.Initialiser)
	}
	env.Define(stmt.Name, value)
}

func (i *Interpreter) interpretFunDecl(env *environment, stmt ast.FunDecl) {
	fun := loxFunction{
		name:    stmt.Name.Literal,
		params:  stmt.Params,
		body:    stmt.Body,
		closure: env,
	}
	env.Define(stmt.Name, fun)
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
	return i.executeBlock(env.Child(), stmt.Stmts)
}

func (i *Interpreter) executeBlock(env *environment, stmts []ast.Stmt) stmtResult {
	for _, stmt := range stmts {
		result := i.interpretStmt(env, stmt)
		if _, ok := result.(stmtResultNone); !ok {
			return result
		}
	}
	return stmtResultNone{}
}

func (i *Interpreter) interpretIfStmt(env *environment, stmt ast.IfStmt) stmtResult {
	condition := i.interpretExpr(env, stmt.Condition)
	if condition.IsTruthy() {
		return i.interpretStmt(env, stmt.Then)
	} else if stmt.Else != nil {
		return i.interpretStmt(env, stmt.Else)
	} else {
		return stmtResultNone{}
	}
}

func (i *Interpreter) interpretWhileStmt(env *environment, stmt ast.WhileStmt) stmtResult {
	for i.interpretExpr(env, stmt.Condition).IsTruthy() {
		switch result := i.interpretStmt(env, stmt.Body).(type) {
		case stmtResultBreak:
			return stmtResultNone{}
		case stmtResultReturn:
			return result
		}
	}
	return stmtResultNone{}
}

func (i *Interpreter) interpretForStmt(env *environment, stmt ast.ForStmt) stmtResult {
	childEnv := env.Child()
	if stmt.Initialise != nil {
		i.interpretStmt(childEnv, stmt.Initialise)
	}
	for stmt.Condition == nil || i.interpretExpr(childEnv, stmt.Condition).IsTruthy() {
		switch result := i.interpretStmt(childEnv, stmt.Body).(type) {
		case stmtResultBreak:
			return stmtResultNone{}
		case stmtResultReturn:
			return result
		}
		if stmt.Update != nil {
			i.interpretExpr(childEnv, stmt.Update)
		}
	}
	return stmtResultNone{}
}

func (i *Interpreter) interpretBreakStmt() stmtResultBreak {
	return stmtResultBreak{}
}

func (i *Interpreter) interpretContinueStmt() stmtResultContinue {
	return stmtResultContinue{}
}

func (i *Interpreter) interpretReturnStmt(env *environment, stmt ast.ReturnStmt) stmtResultReturn {
	var value loxObject = loxNil{}
	if stmt.Value != nil {
		value = i.interpretExpr(env, stmt.Value)
	}
	return stmtResultReturn{Value: value}
}

func (i *Interpreter) interpretExpr(env *environment, expr ast.Expr) loxObject {
	switch expr := expr.(type) {
	case ast.FunExpr:
		return i.interpretFunExpr(env, expr)
	case ast.GroupExpr:
		return i.interpretGroupExpr(env, expr)
	case ast.LiteralExpr:
		return i.interpretLiteralExpr(expr)
	case ast.VariableExpr:
		return i.interpretVariableExpr(env, expr)
	case ast.CallExpr:
		return i.interpretCallExpr(env, expr)
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

func (i *Interpreter) interpretFunExpr(env *environment, expr ast.FunExpr) loxObject {
	return loxFunction{
		name:    fmt.Sprintf("<lambda> at %d:%d", expr.Fun.Start.Line, expr.Fun.Start.Column),
		params:  expr.Params,
		body:    expr.Body,
		closure: env,
	}
}

func (i *Interpreter) interpretGroupExpr(env *environment, expr ast.GroupExpr) loxObject {
	return i.interpretExpr(env, expr.Expr)
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
		panic(fmt.Sprintf("unexpected literal type: %s", tok.Type))
	}
}

func (i *Interpreter) interpretVariableExpr(env *environment, expr ast.VariableExpr) loxObject {
	return i.resolveIdent(env, expr.Name)
}

func (i *Interpreter) resolveIdent(env *environment, tok token.Token) loxObject {
	distance, ok := i.localDeclDistancesByIdent[tok]
	if ok {
		return env.GetAt(distance, tok)
	}
	return i.globals.Get(tok)
}

func (i *Interpreter) interpretCallExpr(env *environment, expr ast.CallExpr) loxObject {
	callee := i.interpretExpr(env, expr.Callee)
	args := make([]loxObject, len(expr.Args))
	for j, arg := range expr.Args {
		args[j] = i.interpretExpr(env, arg)
	}

	callable, ok := callee.(loxCallable)
	if !ok {
		panic(newNodeRuntimeErrorf(expr.Callee, "%h object is not callable", callee.Type()))
	}

	params := callable.Params()
	arity := len(params)
	switch {
	case len(args) < arity:
		argumentSuffix := ""
		if arity-len(args) > 1 {
			argumentSuffix = "s"
		}
		missingArgs := params[len(args):]
		var missingArgsStr string
		switch len(missingArgs) {
		case 1:
			missingArgsStr = missingArgs[0]
		case 2:
			missingArgsStr = missingArgs[0] + " and " + missingArgs[1]
		default:
			missingArgsStr = strings.Join(missingArgs[:len(missingArgs)-1], ", ") + ", and " + missingArgs[len(missingArgs)-1]
		}
		panic(newNodeRuntimeErrorf(
			expr,
			"%s() missing %d argument%s: %s", callable.Name(), arity-len(args), argumentSuffix, missingArgsStr,
		))
	case len(args) > arity:
		panic(newNodeRangeRuntimeErrorf(
			expr.Args[arity],
			expr.Args[len(args)-1],
			"%s() accepts %d arguments but %d were given", callable.Name(), arity, len(args),
		))
	}

	return callable.Call(i, args)
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
	distance, ok := i.localDeclDistancesByIdent[expr.Left]
	if ok {
		env.AssignAt(distance, expr.Left, value)
	} else {
		i.globals.Assign(expr.Left, value)
	}
	return value
}

type runtimeError struct {
	start token.Position
	end   token.Position
	msg   string
}

func newRuntimeErrorf(start token.Position, end token.Position, format string, args ...interface{}) *runtimeError {
	return &runtimeError{
		start: start,
		end:   end,
		msg:   fmt.Sprintf(format, args...),
	}
}

func newTokenRuntimeErrorf(tok token.Token, format string, args ...interface{}) *runtimeError {
	return newRuntimeErrorf(tok.Start, tok.End, format, args...)
}

func newNodeRuntimeErrorf(node ast.Node, format string, args ...interface{}) *runtimeError {
	return newRuntimeErrorf(node.Start(), node.End(), format, args...)
}

func newNodeRangeRuntimeErrorf(start, end ast.Node, format string, args ...interface{}) *runtimeError {
	return newRuntimeErrorf(start.Start(), end.End(), format, args...)
}

func (e *runtimeError) Error() string {
	bold := color.New(color.Bold)
	red := color.New(color.FgRed)

	line := e.start.File.Line(e.start.Line)

	var b strings.Builder
	bold.Fprint(&b, e.start, ": ", red.Sprint("runtime error: "), e.msg, "\n")
	fmt.Fprintln(&b, string(line))
	fmt.Fprint(&b, strings.Repeat(" ", runewidth.StringWidth(string(line[:e.start.Column]))))
	red.Fprint(&b, strings.Repeat("~", runewidth.StringWidth(string(line[e.start.Column:e.end.Column]))))

	return b.String()
}
