// Package interpreter implements an interpreter of Lox programs.
package interpreter

import (
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/marcuscaisey/lox/lox"
	"github.com/marcuscaisey/lox/lox/analysis"
	"github.com/marcuscaisey/lox/lox/ast"
	"github.com/marcuscaisey/lox/lox/token"
)

// Interpreter is the interpreter for the language.
type Interpreter struct {
	globals            *environment
	declDistancesByTok map[token.Token]int
	callStack          *callStack

	printExprStmtResults bool
}

// Option can be passed to New to configure the interpreter.
type Option func(*Interpreter)

// REPLMode sets the interpreter to REPL mode.
// In REPL mode, the interpreter will print the result of expression statements.
func REPLMode() Option {
	return func(i *Interpreter) {
		i.printExprStmtResults = true
	}
}

// New constructs a new Interpreter with the given options.
func New(opts ...Option) *Interpreter {
	globals := newEnvironment()
	for name, builtin := range builtins {
		globals.Set(name, builtin)
	}
	interpreter := &Interpreter{
		globals:            globals,
		declDistancesByTok: map[token.Token]int{},
		callStack:          newCallStack(),
	}
	for _, opt := range opts {
		opt(interpreter)
	}
	return interpreter
}

// Interpret interprets a program and returns an error if one occurred.
// Interpret can be called multiple times with different ASTs and the state will be maintained between calls.
func (i *Interpreter) Interpret(program ast.Program) error {
	declDistancesByTok, errs := analysis.ResolveIdents(program)
	errs = append(errs, analysis.CheckSemantics(program)...)
	if err := errs.Err(); err != nil {
		return err
	}
	maps.Copy(i.declDistancesByTok, declDistancesByTok)
	return i.interpretProgram(program)
}

func (i *Interpreter) interpretProgram(node ast.Program) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if loxErr, ok := r.(*lox.Error); ok {
				err = loxErr
				if i.callStack.Len() > 0 {
					i.callStack.Push("", loxErr.Start)
					err = fmt.Errorf("%w\n\n%s", err, i.callStack.StackTrace())
					i.callStack.Clear()
				}
			} else {
				panic(r)
			}
		}
	}()
	for _, stmt := range node.Stmts {
		i.execStmt(i.globals, stmt)
	}
	return nil
}

//gosumtype:decl stmtResult
type stmtResult interface {
	isStmtResult()
}

type (
	stmtResultNone     struct{ stmtResult }
	stmtResultBreak    struct{ stmtResult }
	stmtResultContinue struct{ stmtResult }
	stmtResultReturn   struct {
		Value loxObject
		stmtResult
	}
)

func (i *Interpreter) execStmt(env *environment, stmt ast.Stmt) stmtResult {
	switch stmt := stmt.(type) {
	case ast.VarDecl:
		i.execVarDecl(env, stmt)
	case ast.FunDecl:
		i.execFunDecl(env, stmt)
	case ast.ClassDecl:
		i.execClassDecl(env, stmt)
	case ast.ExprStmt:
		i.execExprStmt(env, stmt)
	case ast.PrintStmt:
		i.execPrintStmt(env, stmt)
	case ast.BlockStmt:
		return i.execBlockStmt(env, stmt)
	case ast.IfStmt:
		return i.execIfStmt(env, stmt)
	case ast.WhileStmt:
		return i.execWhileStmt(env, stmt)
	case ast.ForStmt:
		return i.execForStmt(env, stmt)
	case ast.BreakStmt:
		return i.execBreakStmt()
	case ast.ContinueStmt:
		return i.execContinueStmt()
	case ast.ReturnStmt:
		return i.execReturnStmt(env, stmt)
	case ast.CommentStmt, ast.InlineCommentStmt, ast.IllegalStmt, ast.MethodDecl:
		panic(fmt.Sprintf("unexpected statement type: %T", stmt))
	}
	return stmtResultNone{}
}

func (i *Interpreter) execVarDecl(env *environment, stmt ast.VarDecl) {
	if stmt.Initialiser != nil {
		env.Define(stmt.Name, i.evalExpr(env, stmt.Initialiser))
	} else {
		env.Declare(stmt.Name)
	}
}

func (i *Interpreter) execFunDecl(env *environment, stmt ast.FunDecl) {
	env.Define(stmt.Name, newLoxFunction(stmt.Name.Lexeme, stmt.Function, funTypeFunction, env))
}

func (i *Interpreter) execClassDecl(env *environment, stmt ast.ClassDecl) {
	env.Define(stmt.Name, newLoxClass(stmt.Name.Lexeme, stmt.Methods(), env))
}

func (i *Interpreter) execExprStmt(env *environment, stmt ast.ExprStmt) {
	value := i.evalExpr(env, stmt.Expr)
	if i.printExprStmtResults {
		fmt.Println(value.String())
	}
}

func (i *Interpreter) execPrintStmt(env *environment, stmt ast.PrintStmt) {
	value := i.evalExpr(env, stmt.Expr)
	fmt.Println(value.String())
}

func (i *Interpreter) execBlockStmt(env *environment, stmt ast.BlockStmt) stmtResult {
	return i.executeBlock(env.Child(), stmt.Stmts)
}

func (i *Interpreter) executeBlock(env *environment, stmts []ast.Stmt) stmtResult {
	for _, stmt := range stmts {
		result := i.execStmt(env, stmt)
		if _, ok := result.(stmtResultNone); !ok {
			return result
		}
	}
	return stmtResultNone{}
}

func (i *Interpreter) execIfStmt(env *environment, stmt ast.IfStmt) stmtResult {
	condition := i.evalExpr(env, stmt.Condition)
	if isTruthy(condition) {
		return i.execStmt(env, stmt.Then)
	} else if stmt.Else != nil {
		return i.execStmt(env, stmt.Else)
	} else {
		return stmtResultNone{}
	}
}

func (i *Interpreter) execWhileStmt(env *environment, stmt ast.WhileStmt) stmtResult {
	for isTruthy(i.evalExpr(env, stmt.Condition)) {
		switch result := i.execStmt(env, stmt.Body).(type) {
		case stmtResultBreak:
			return stmtResultNone{}
		case stmtResultReturn:
			return result
		case stmtResultContinue, stmtResultNone:
		}
	}
	return stmtResultNone{}
}

func (i *Interpreter) execForStmt(env *environment, stmt ast.ForStmt) stmtResult {
	childEnv := env.Child()
	if stmt.Initialise != nil {
		i.execStmt(childEnv, stmt.Initialise)
	}
	for stmt.Condition == nil || isTruthy(i.evalExpr(childEnv, stmt.Condition)) {
		switch result := i.execStmt(childEnv, stmt.Body).(type) {
		case stmtResultBreak:
			return stmtResultNone{}
		case stmtResultReturn:
			return result
		case stmtResultContinue, stmtResultNone:
		}
		if stmt.Update != nil {
			i.evalExpr(childEnv, stmt.Update)
		}
	}
	return stmtResultNone{}
}

func (i *Interpreter) execBreakStmt() stmtResultBreak {
	return stmtResultBreak{}
}

func (i *Interpreter) execContinueStmt() stmtResultContinue {
	return stmtResultContinue{}
}

func (i *Interpreter) execReturnStmt(env *environment, stmt ast.ReturnStmt) stmtResultReturn {
	var value loxObject = loxNil{}
	if stmt.Value != nil {
		value = i.evalExpr(env, stmt.Value)
	}
	return stmtResultReturn{Value: value}
}

func (i *Interpreter) evalExpr(env *environment, expr ast.Expr) loxObject {
	switch expr := expr.(type) {
	case ast.FunExpr:
		return i.evalFunExpr(env, expr)
	case ast.GroupExpr:
		return i.evalGroupExpr(env, expr)
	case ast.LiteralExpr:
		return i.evalLiteralExpr(expr)
	case ast.VariableExpr:
		return i.evalVariableExpr(env, expr)
	case ast.ThisExpr:
		return i.evalThisExpr(env, expr)
	case ast.CallExpr:
		return i.evalCallExpr(env, expr)
	case ast.GetExpr:
		return i.evalGetExpr(env, expr)
	case ast.UnaryExpr:
		return i.evalUnaryExpr(env, expr)
	case ast.BinaryExpr:
		return i.evalBinaryExpr(env, expr)
	case ast.TernaryExpr:
		return i.evalTernaryExpr(env, expr)
	case ast.AssignmentExpr:
		return i.evalAssignmentExpr(env, expr)
	case ast.SetExpr:
		return i.evalSetExpr(env, expr)
	}
	panic("unreachable")
}

func (i *Interpreter) evalFunExpr(env *environment, expr ast.FunExpr) loxObject {
	return newLoxFunction("(anonymous)", expr.Function, funTypeFunction, env)
}

func (i *Interpreter) evalGroupExpr(env *environment, expr ast.GroupExpr) loxObject {
	return i.evalExpr(env, expr.Expr)
}

func (i *Interpreter) evalLiteralExpr(expr ast.LiteralExpr) loxObject {
	switch tok := expr.Value; tok.Type {
	case token.Number:
		value, err := strconv.ParseFloat(tok.Lexeme, 64)
		if err != nil {
			panic(fmt.Sprintf("unexpected error parsing number literal: %s", err))
		}
		return loxNumber(value)
	case token.String:
		return loxString(tok.Lexeme[1 : len(tok.Lexeme)-1]) // Remove surrounding quotes
	case token.True, token.False:
		return loxBool(tok.Type == token.True)
	case token.Nil:
		return loxNil{}
	default:
		panic(fmt.Sprintf("unexpected literal type: %s", tok.Type))
	}
}

func (i *Interpreter) evalVariableExpr(env *environment, expr ast.VariableExpr) loxObject {
	return i.resolveIdent(env, expr.Name)
}

func (i *Interpreter) evalThisExpr(env *environment, expr ast.ThisExpr) loxObject {
	return i.resolveIdent(env, expr.This)
}

func (i *Interpreter) resolveIdent(env *environment, tok token.Token) loxObject {
	distance, ok := i.declDistancesByTok[tok]
	if ok {
		return env.GetAt(distance, tok)
	}
	return i.globals.Get(tok)
}

func (i *Interpreter) evalCallExpr(env *environment, expr ast.CallExpr) loxObject {
	callee := i.evalExpr(env, expr.Callee)
	args := make([]loxObject, len(expr.Args))
	for j, arg := range expr.Args {
		args[j] = i.evalExpr(env, arg)
	}

	callable, ok := callee.(loxCallable)
	if !ok {
		panic(lox.NewErrorf(expr.Callee, "%m object is not callable", callee.Type()))
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
		panic(lox.NewErrorf(
			expr,
			"%s() missing %d argument%s: %s", callable.CallableName(), arity-len(args), argumentSuffix, missingArgsStr,
		))
	case len(args) > arity:
		panic(lox.NewErrorf(
			expr.Args[arity:],
			"%s() accepts %d arguments but %d were given", callable.CallableName(), arity, len(args),
		))
	}

	result := i.call(expr.Start(), callable, args)
	if errorMsg, ok := result.(errorMsg); ok {
		panic(lox.NewError(expr, string(errorMsg)))
	}
	return result
}

func (i *Interpreter) call(location token.Position, callable loxCallable, args []loxObject) loxObject {
	i.callStack.Push(callable.CallableName(), location)
	result := callable.Call(i, args)
	i.callStack.Pop()
	return result
}

func (i *Interpreter) evalGetExpr(env *environment, expr ast.GetExpr) loxObject {
	object := i.evalExpr(env, expr.Object)
	getter, ok := object.(loxGetter)
	if !ok {
		panic(lox.NewErrorf(expr, "property access is not valid for %m object", object.Type()))
	}
	return getter.Get(i, expr.Name)
}

func (i *Interpreter) evalUnaryExpr(env *environment, expr ast.UnaryExpr) loxObject {
	right := i.evalExpr(env, expr.Right)
	if expr.Op.Type == token.Bang {
		// The behaviour of ! is independent of the type of the operand, so we can implement it here.
		return !isTruthy(right)
	}
	unaryOperand, ok := right.(loxUnaryOperand)
	if ok {
		if result := unaryOperand.UnaryOp(expr.Op); result != nil {
			return result
		}
	}
	panic(lox.NewErrorf(expr.Op, "%m operator cannot be used with type %m", expr.Op.Type, right.Type()))
}

func (i *Interpreter) evalBinaryExpr(env *environment, expr ast.BinaryExpr) loxObject {
	left := i.evalExpr(env, expr.Left)

	// We check for short-circuiting operators first.
	switch expr.Op.Type {
	case token.Or:
		// The behaviour of or is independent of the types of the operands, so we can implement it here.
		if isTruthy(left) {
			return left
		} else {
			return i.evalExpr(env, expr.Right)
		}
	case token.And:
		// The behaviour of and is independent of the types of the operands, so we can implement it here.
		if !isTruthy(left) {
			return left
		} else {
			return i.evalExpr(env, expr.Right)
		}
	default:
	}

	right := i.evalExpr(env, expr.Right)
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
		binaryOperand, ok := left.(loxBinaryOperand)
		if ok {
			if result := binaryOperand.BinaryOp(expr.Op, right); result != nil {
				return result
			}
		}
		panic(lox.NewErrorf(expr.Op, "%m operator cannot be used with types %m and %m", expr.Op.Type, left.Type(), right.Type()))
	}
}

func (i *Interpreter) evalTernaryExpr(env *environment, expr ast.TernaryExpr) loxObject {
	condition := i.evalExpr(env, expr.Condition)
	if isTruthy(condition) {
		return i.evalExpr(env, expr.Then)
	}
	return i.evalExpr(env, expr.Else)
}

func (i *Interpreter) evalAssignmentExpr(env *environment, expr ast.AssignmentExpr) loxObject {
	value := i.evalExpr(env, expr.Right)
	distance, ok := i.declDistancesByTok[expr.Left]
	if ok {
		env.AssignAt(distance, expr.Left, value)
	} else {
		i.globals.Assign(expr.Left, value)
	}
	return value
}

func (i *Interpreter) evalSetExpr(env *environment, expr ast.SetExpr) loxObject {
	object := i.evalExpr(env, expr.Object)
	setter, ok := object.(loxSetter)
	if !ok {
		panic(lox.NewErrorf(expr, "property assignment is not valid for %m object", object.Type()))
	}
	value := i.evalExpr(env, expr.Value)
	setter.Set(i, expr.Name, value)
	return value
}

func isTruthy(obj loxObject) loxBool {
	if truther, ok := obj.(loxTruther); ok {
		return truther.IsTruthy()
	}
	return true
}
