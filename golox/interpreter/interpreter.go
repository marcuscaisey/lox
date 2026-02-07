// Package interpreter implements an interpreter of Lox programs.
package interpreter

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/marcuscaisey/lox/golox/analyse"
	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/builtins"
	"github.com/marcuscaisey/lox/golox/loxerr"
	"github.com/marcuscaisey/lox/golox/token"
)

// Interpreter is the interpreter for the language.
type Interpreter struct {
	globals      environment
	callStack    *callStack
	builtinStubs []ast.Decl

	replMode bool
}

// Option can be passed to New to configure the interpreter.
type Option func(*Interpreter)

// WithREPLMode configures the interpreter to run in REPL mode.
// In REPL mode, the interpreter prints the result of expression statements.
func WithREPLMode(enabled bool) Option {
	return func(i *Interpreter) {
		i.replMode = enabled
	}
}

// New constructs a new Interpreter with the given options.
// argv
func New(argv []string, opts ...Option) *Interpreter {
	var globals environment = newGlobalEnvironment()
	for name, builtin := range builtinFunctions {
		globals = globals.Define(name, builtin)
	}

	argvValues := make([]loxValue, len(argv))
	for i, arg := range argv {
		argvValues[i] = loxString(arg)
	}
	globals = globals.Define("argv", newLoxList(argvValues))

	interpreter := &Interpreter{
		globals:      globals,
		callStack:    newCallStack(),
		builtinStubs: builtins.MustParseStubs("builtins.lox"),
	}
	for _, opt := range opts {
		opt(interpreter)
	}
	return interpreter
}

// Execute executes a program and returns an error if one occurred.
// Execute can be called multiple times with different programs and the state will be maintained between calls.
func (i *Interpreter) Execute(program *ast.Program) error {
	if err := analyse.Program(program, i.builtinStubs, analyse.WithFatalOnly(true)); err != nil {
		return err
	}
	return i.interpretProgram(program)
}

func (i *Interpreter) interpretProgram(node *ast.Program) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if loxErr, ok := r.(*loxerr.Error); ok {
				err = loxErr
				if i.callStack.Len() > 0 {
					i.callStack.Push("", loxErr.Start())
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

//sumtype:decl
type stmtResult interface {
	isStmtResult()
}

type (
	stmtResultNone     struct{ stmtResult }
	stmtResultBreak    struct{ stmtResult }
	stmtResultContinue struct{ stmtResult }
	stmtResultReturn   struct {
		Value loxValue
		stmtResult
	}
)

func (i *Interpreter) execStmt(env environment, stmt ast.Stmt) (stmtResult, environment) {
	var result stmtResult = stmtResultNone{}
	newEnv := env
	switch stmt := stmt.(type) {
	case *ast.VarDecl:
		newEnv = i.execVarDecl(env, stmt)
	case *ast.FunDecl:
		newEnv = i.execFunDecl(env, stmt)
	case *ast.ClassDecl:
		newEnv = i.execClassDecl(env, stmt)
	case *ast.ExprStmt:
		i.execExprStmt(env, stmt)
	case *ast.PrintStmt:
		i.execPrintStmt(env, stmt)
	case *ast.Block:
		result = i.execBlock(env, stmt)
	case *ast.IfStmt:
		result = i.execIfStmt(env, stmt)
	case *ast.WhileStmt:
		result = i.execWhileStmt(env, stmt)
	case *ast.ForStmt:
		result = i.execForStmt(env, stmt)
	case *ast.BreakStmt:
		result = i.execBreakStmt()
	case *ast.ContinueStmt:
		result = i.execContinueStmt()
	case *ast.ReturnStmt:
		result = i.execReturnStmt(env, stmt)
	case *ast.IllegalStmt, *ast.Comment, *ast.CommentedStmt, *ast.ParamDecl, *ast.MethodDecl:
		panic(fmt.Sprintf("unexpected statement type: %T", stmt))
	}
	return result, newEnv
}

func (i *Interpreter) execVarDecl(env environment, stmt *ast.VarDecl) environment {
	var value loxValue
	if stmt.Initialiser != nil {
		value = i.evalExpr(env, stmt.Initialiser)
	}
	if stmt.Name.String() == token.IdentBlank {
		return env
	}
	newEnv := env.Declare(stmt.Name)
	if stmt.Initialiser != nil {
		newEnv.Assign(stmt.Name, value)
	}
	return newEnv
}

func (i *Interpreter) execFunDecl(env environment, stmt *ast.FunDecl) environment {
	if stmt.Name.String() == token.IdentBlank {
		return env
	}
	newEnv := env.Declare(stmt.Name)
	newEnv.Assign(stmt.Name, newLoxFunction(stmt.Name.String(), stmt.Function, funTypeFunction, newEnv))
	return newEnv
}

func (i *Interpreter) execClassDecl(env environment, stmt *ast.ClassDecl) environment {
	if stmt.Name.String() == token.IdentBlank {
		return env
	}
	var superclass *loxClass
	if stmt.Superclass.IsValid() {
		superclassValue := env.Get(stmt.Superclass)
		var ok bool
		superclass, ok = superclassValue.(*loxClass)
		if !ok {
			panic(loxerr.Newf(stmt.Superclass, loxerr.Fatal, "expected superclass to be a class, got %m", superclassValue.Type()))
		}
	}
	_ = superclass
	newEnv := env.Declare(stmt.Name)
	class := newLoxClass(stmt.Name.String(), superclass, stmt.Methods(), newEnv)
	newEnv.Assign(stmt.Name, class)
	return newEnv
}

func (i *Interpreter) execExprStmt(env environment, stmt *ast.ExprStmt) {
	value := i.evalExpr(env, stmt.Expr)
	if i.replMode {
		fmt.Println(value.String())
	}
}

func (i *Interpreter) execPrintStmt(env environment, stmt *ast.PrintStmt) {
	value := i.evalExpr(env, stmt.Expr)
	fmt.Println(value.String())
}

func (i *Interpreter) execBlock(env environment, stmt *ast.Block) stmtResult {
	return i.executeBlock(env.Child(), stmt.Stmts)
}

func (i *Interpreter) executeBlock(env environment, stmts []ast.Stmt) stmtResult {
	currentEnv := env
	for _, stmt := range stmts {
		var result stmtResult
		result, currentEnv = i.execStmt(currentEnv, stmt)
		if _, ok := result.(stmtResultNone); !ok {
			return result
		}
	}
	return stmtResultNone{}
}

func (i *Interpreter) execIfStmt(env environment, stmt *ast.IfStmt) stmtResult {
	condition := i.evalExpr(env, stmt.Condition)
	if isTruthy(condition) {
		result, _ := i.execStmt(env, stmt.Then)
		return result
	} else if stmt.Else != nil {
		result, _ := i.execStmt(env, stmt.Else)
		return result
	} else {
		return stmtResultNone{}
	}
}

func (i *Interpreter) execWhileStmt(env environment, stmt *ast.WhileStmt) stmtResult {
	for isTruthy(i.evalExpr(env, stmt.Condition)) {
		switch result, _ := i.execStmt(env, stmt.Body); result.(type) {
		case stmtResultBreak:
			return stmtResultNone{}
		case stmtResultReturn:
			return result
		case stmtResultContinue, stmtResultNone:
		}
	}
	return stmtResultNone{}
}

func (i *Interpreter) execForStmt(env environment, stmt *ast.ForStmt) stmtResult {
	childEnv := env.Child()
	if stmt.Initialise != nil {
		_, childEnv = i.execStmt(childEnv, stmt.Initialise)
	}
	for stmt.Condition == nil || isTruthy(i.evalExpr(childEnv, stmt.Condition)) {
		switch result, _ := i.execStmt(childEnv, stmt.Body); result.(type) {
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

func (i *Interpreter) execReturnStmt(env environment, stmt *ast.ReturnStmt) stmtResultReturn {
	var value loxValue = loxNil{}
	if stmt.Value != nil {
		value = i.evalExpr(env, stmt.Value)
	}
	return stmtResultReturn{Value: value}
}

func (i *Interpreter) evalExpr(env environment, expr ast.Expr) loxValue {
	switch expr := expr.(type) {
	case *ast.LiteralExpr:
		return i.evalLiteralExpr(expr)
	case *ast.FunExpr:
		return i.evalFunExpr(env, expr)
	case *ast.ListExpr:
		return i.evalListExpr(env, expr)
	case *ast.IdentExpr:
		return i.evalIdentExpr(env, expr)
	case *ast.AssignmentExpr:
		return i.evalAssignmentExpr(env, expr)
	case *ast.ThisExpr:
		return i.evalThisExpr(env, expr)
	case *ast.SuperExpr:
		return i.evalSuperExpr(env, expr)
	case *ast.CallExpr:
		return i.evalCallExpr(env, expr)
	case *ast.IndexExpr:
		return i.evalIndexExpr(env, expr)
	case *ast.IndexSetExpr:
		return i.evalIndexSetExpr(env, expr)
	case *ast.PropertyExpr:
		return i.evalPropertyExpr(env, expr)
	case *ast.PropertySetExpr:
		return i.evalPropertySetExpr(env, expr)
	case *ast.UnaryExpr:
		return i.evalUnaryExpr(env, expr)
	case *ast.BinaryExpr:
		return i.evalBinaryExpr(env, expr)
	case *ast.TernaryExpr:
		return i.evalTernaryExpr(env, expr)
	case *ast.GroupExpr:
		return i.evalGroupExpr(env, expr)
	}
	panic("unreachable")
}

func (i *Interpreter) evalLiteralExpr(expr *ast.LiteralExpr) loxValue {
	switch tok := expr.Value; tok.Type {
	case token.Number:
		value, err := strconv.ParseFloat(tok.Lexeme, 64)
		if err != nil {
			panic(fmt.Sprintf("unexpected error parsing number literal: %s", err))
		}
		return loxNumber(value)
	case token.String:
		// Double-quoted Go strings can't contain new lines.
		singleLineLexeme := strings.ReplaceAll(tok.Lexeme, "\n", `\n`)
		value, err := strconv.Unquote(singleLineLexeme)
		if err != nil {
			panic(fmt.Sprintf("unexpected error parsing string literal: %s", err))
		}
		return loxString(value)
	case token.True, token.False:
		return loxBool(tok.Type == token.True)
	case token.Nil:
		return loxNil{}
	default:
		panic(fmt.Sprintf("unexpected literal type: %s", tok.Type))
	}
}

func (i *Interpreter) evalFunExpr(env environment, expr *ast.FunExpr) loxValue {
	return newLoxFunction("(anonymous)", expr.Function, funTypeFunction, env)
}

func (i *Interpreter) evalListExpr(env environment, expr *ast.ListExpr) loxValue {
	elements := make([]loxValue, len(expr.Elements))
	for j, element := range expr.Elements {
		elements[j] = i.evalExpr(env, element)
	}
	return newLoxList(elements)
}

func (i *Interpreter) evalIdentExpr(env environment, expr *ast.IdentExpr) loxValue {
	return env.Get(expr.Ident)
}

func (i *Interpreter) evalAssignmentExpr(env environment, expr *ast.AssignmentExpr) loxValue {
	value := i.evalExpr(env, expr.Right)
	if expr.Left.String() != token.IdentBlank {
		env.Assign(expr.Left, value)
	}
	return value
}

func (i *Interpreter) evalThisExpr(env environment, _ *ast.ThisExpr) loxValue {
	return env.GetByName(token.This.String())
}

func (i *Interpreter) evalSuperExpr(env environment, _ *ast.SuperExpr) loxValue {
	superValue := env.GetByName(token.Super.String())
	superclass, ok := superValue.(*loxClass)
	if !ok {
		panic(fmt.Sprintf("unexpected super type: %T", superValue))
	}
	return newLoxSuperObject(superclass, env)
}

func (i *Interpreter) evalCallExpr(env environment, expr *ast.CallExpr) loxValue {
	callee := i.evalExpr(env, expr.Callee)
	args := make([]loxValue, len(expr.Args))
	for j, arg := range expr.Args {
		args[j] = i.evalExpr(env, arg)
	}

	callable, ok := callee.(loxCallable)
	if !ok {
		panic(loxerr.Newf(expr.Callee, loxerr.Fatal, "%m value is not callable", callee.Type()))
	}

	params := callable.Params()
	if len(args) != len(params) {
		wereWas := "were"
		if len(args) == 1 {
			wereWas = "was"
		}
		argumentSuffix := "s"
		if len(params) == 1 {
			argumentSuffix = ""
		}
		panic(loxerr.Newf(
			expr,
			loxerr.Fatal, "%s() accepts %d argument%s but %d %s given", callable.CallableName(), len(params), argumentSuffix, len(args), wereWas,
		))
	}

	result := i.call(expr.Start(), callable, args)
	if errorMsg, ok := result.(errorMsg); ok {
		panic(loxerr.Newf(expr, loxerr.Fatal, "%s", string(errorMsg)))
	}
	return result
}

func (i *Interpreter) evalIndexExpr(env environment, expr *ast.IndexExpr) loxValue {
	subject := i.evalExpr(env, expr.Subject)
	indexable := assertIndexable(subject, expr.Subject)
	index := i.evalExpr(env, expr.Index)
	return indexable.Index(index, expr.Index)
}

func (i *Interpreter) evalIndexSetExpr(env environment, expr *ast.IndexSetExpr) loxValue {
	subject := i.evalExpr(env, expr.Subject)
	indexable := assertIndexable(subject, expr.Subject)
	index := i.evalExpr(env, expr.Index)
	value := i.evalExpr(env, expr.Value)
	indexable.SetIndex(index, expr.Index, value)
	return value
}

func assertIndexable(value loxValue, node ast.Node) loxIndexable {
	indexable, ok := value.(loxIndexable)
	if !ok {
		panic(loxerr.Newf(node, loxerr.Fatal, "%m value is not indexable", value.Type()))
	}
	return indexable
}

func (i *Interpreter) call(location token.Position, callable loxCallable, args []loxValue) loxValue {
	i.callStack.Push(callable.CallableName(), location)
	result := callable.Call(i, args)
	i.callStack.Pop()
	return result
}

func (i *Interpreter) evalPropertyExpr(env environment, expr *ast.PropertyExpr) loxValue {
	object := i.evalExpr(env, expr.Object)
	accessible, ok := object.(loxPropertyAccessible)
	if !ok {
		panic(loxerr.Newf(expr, loxerr.Fatal, "property access is not valid for %m value", object.Type()))
	}
	return accessible.Property(i, expr.Name)
}

func (i *Interpreter) evalPropertySetExpr(env environment, expr *ast.PropertySetExpr) loxValue {
	object := i.evalExpr(env, expr.Object)
	settable, ok := object.(loxPropertySettable)
	if !ok {
		panic(loxerr.Newf(expr, loxerr.Fatal, "property assignment is not valid for %m value", object.Type()))
	}
	value := i.evalExpr(env, expr.Value)
	settable.SetProperty(i, expr.Name, value)
	return value
}

func (i *Interpreter) evalUnaryExpr(env environment, expr *ast.UnaryExpr) loxValue {
	right := i.evalExpr(env, expr.Right)
	if expr.Op.Type == token.Bang {
		// The behaviour of ! is independent of the type of the operand, so we can implement it here.
		return !isTruthy(right)
	}
	unaryOperand, ok := right.(loxUnaryOperand)
	if !ok {
		panic(newInvalidUnaryOpError(expr.Op, right))
	}
	return unaryOperand.UnaryOp(expr.Op)
}

func (i *Interpreter) evalBinaryExpr(env environment, expr *ast.BinaryExpr) loxValue {
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
		return loxBool(left.Equals(right))
	case token.BangEqual:
		return loxBool(!left.Equals(right))
	default:
	}

	binaryOperand, ok := left.(loxBinaryOperand)
	if !ok {
		panic(newInvalidBinaryOpError(expr.Op, left, right))
	}
	return binaryOperand.BinaryOp(expr.Op, right)
}

func (i *Interpreter) evalTernaryExpr(env environment, expr *ast.TernaryExpr) loxValue {
	condition := i.evalExpr(env, expr.Condition)
	if isTruthy(condition) {
		return i.evalExpr(env, expr.Then)
	}
	return i.evalExpr(env, expr.Else)
}

func (i *Interpreter) evalGroupExpr(env environment, expr *ast.GroupExpr) loxValue {
	return i.evalExpr(env, expr.Expr)
}

func isTruthy(obj loxValue) loxBool {
	if truther, ok := obj.(loxTruther); ok {
		return truther.IsTruthy()
	}
	return true
}
