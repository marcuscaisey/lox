// Package interpreter implements an interpreter of Lox programs.
package interpreter

import (
	"fmt"
	"strconv"

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
	builtInStubs []ast.Decl

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
func New(opts ...Option) *Interpreter {
	var globals environment = newGlobalEnvironment()
	for name, builtIn := range builtIns {
		globals = globals.Define(name, builtIn)
	}
	interpreter := &Interpreter{
		globals:      globals,
		callStack:    newCallStack(),
		builtInStubs: builtins.MustParseStubs("built_ins.lox"),
	}
	for _, opt := range opts {
		opt(interpreter)
	}
	return interpreter
}

// Interpret executes a program and returns an error if one occurred.
// Interpret can be called multiple times with different programs and the state will be maintained between calls.
func (i *Interpreter) Interpret(program *ast.Program) error {
	if err := analyse.Program(program, i.builtInStubs, analyse.WithFatalOnly(true)); err != nil {
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
		Value loxObject
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
	var value loxObject
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
		superclassLoxObject := env.Get(stmt.Superclass)
		var ok bool
		superclass, ok = superclassLoxObject.(*loxClass)
		if !ok {
			panic(loxerr.Newf(stmt.Superclass, loxerr.Fatal, "expected superclass to be a class, got %m", superclassLoxObject.Type()))
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
	var value loxObject = loxNil{}
	if stmt.Value != nil {
		value = i.evalExpr(env, stmt.Value)
	}
	return stmtResultReturn{Value: value}
}

func (i *Interpreter) evalExpr(env environment, expr ast.Expr) loxObject {
	switch expr := expr.(type) {
	case *ast.FunExpr:
		return i.evalFunExpr(env, expr)
	case *ast.GroupExpr:
		return i.evalGroupExpr(env, expr)
	case *ast.LiteralExpr:
		return i.evalLiteralExpr(expr)
	case *ast.ListExpr:
		return i.evalListExpr(env, expr)
	case *ast.IdentExpr:
		return i.evalIdentExpr(env, expr)
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
	case *ast.UnaryExpr:
		return i.evalUnaryExpr(env, expr)
	case *ast.BinaryExpr:
		return i.evalBinaryExpr(env, expr)
	case *ast.TernaryExpr:
		return i.evalTernaryExpr(env, expr)
	case *ast.AssignmentExpr:
		return i.evalAssignmentExpr(env, expr)
	case *ast.PropertySetExpr:
		return i.evalPropertySetExpr(env, expr)
	}
	panic("unreachable")
}

func (i *Interpreter) evalFunExpr(env environment, expr *ast.FunExpr) loxObject {
	return newLoxFunction("(anonymous)", expr.Function, funTypeFunction, env)
}

func (i *Interpreter) evalGroupExpr(env environment, expr *ast.GroupExpr) loxObject {
	return i.evalExpr(env, expr.Expr)
}

func (i *Interpreter) evalLiteralExpr(expr *ast.LiteralExpr) loxObject {
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

func (i *Interpreter) evalListExpr(env environment, expr *ast.ListExpr) loxObject {
	elements := make([]loxObject, len(expr.Elements))
	for j, element := range expr.Elements {
		elements[j] = i.evalExpr(env, element)
	}
	result := loxList(elements)
	return &result
}

func (i *Interpreter) evalIdentExpr(env environment, expr *ast.IdentExpr) loxObject {
	return env.Get(expr.Ident)
}

func (i *Interpreter) evalThisExpr(env environment, _ *ast.ThisExpr) loxObject {
	return env.GetByName(token.This.String())
}

func (i *Interpreter) evalSuperExpr(env environment, _ *ast.SuperExpr) loxObject {
	superObject := env.GetByName(token.Super.String())
	superclass, ok := superObject.(*loxClass)
	if !ok {
		panic(fmt.Sprintf("unexpected super type: %T", superObject))
	}
	return newLoxSuperObject(superclass, env)
}

func (i *Interpreter) evalCallExpr(env environment, expr *ast.CallExpr) loxObject {
	callee := i.evalExpr(env, expr.Callee)
	args := make([]loxObject, len(expr.Args))
	for j, arg := range expr.Args {
		args[j] = i.evalExpr(env, arg)
	}

	callable, ok := callee.(loxCallable)
	if !ok {
		panic(loxerr.Newf(expr.Callee, loxerr.Fatal, "%m object is not callable", callee.Type()))
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

func (i *Interpreter) evalIndexExpr(env environment, expr *ast.IndexExpr) loxObject {
	subject := i.evalExpr(env, expr.Subject)
	indexable := assertIndexable(subject, expr.Subject)
	index := i.evalExpr(env, expr.Index)
	return indexable.Index(index, expr.Index)
}

func (i *Interpreter) evalIndexSetExpr(env environment, expr *ast.IndexSetExpr) loxObject {
	subject := i.evalExpr(env, expr.Subject)
	indexable := assertIndexable(subject, expr.Subject)
	index := i.evalExpr(env, expr.Index)
	value := i.evalExpr(env, expr.Value)
	indexable.SetIndex(index, expr.Index, value)
	return value
}

func assertIndexable(value loxObject, node ast.Node) loxIndexable {
	indexable, ok := value.(loxIndexable)
	if !ok {
		panic(loxerr.Newf(node, loxerr.Fatal, "%m value is not indexable", value.Type()))
	}
	return indexable
}

func (i *Interpreter) call(location token.Position, callable loxCallable, args []loxObject) loxObject {
	i.callStack.Push(callable.CallableName(), location)
	result := callable.Call(i, args)
	i.callStack.Pop()
	return result
}

func (i *Interpreter) evalPropertyExpr(env environment, expr *ast.PropertyExpr) loxObject {
	object := i.evalExpr(env, expr.Object)
	accessible, ok := object.(loxPropertyAccessible)
	if !ok {
		panic(loxerr.Newf(expr, loxerr.Fatal, "property access is not valid for %m object", object.Type()))
	}
	return accessible.Property(i, expr.Name)
}

func (i *Interpreter) evalUnaryExpr(env environment, expr *ast.UnaryExpr) loxObject {
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

func (i *Interpreter) evalBinaryExpr(env environment, expr *ast.BinaryExpr) loxObject {
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

func (i *Interpreter) evalTernaryExpr(env environment, expr *ast.TernaryExpr) loxObject {
	condition := i.evalExpr(env, expr.Condition)
	if isTruthy(condition) {
		return i.evalExpr(env, expr.Then)
	}
	return i.evalExpr(env, expr.Else)
}

func (i *Interpreter) evalAssignmentExpr(env environment, expr *ast.AssignmentExpr) loxObject {
	value := i.evalExpr(env, expr.Right)
	if expr.Left.String() != token.IdentBlank {
		env.Assign(expr.Left, value)
	}
	return value
}

func (i *Interpreter) evalPropertySetExpr(env environment, expr *ast.PropertySetExpr) loxObject {
	object := i.evalExpr(env, expr.Object)
	settable, ok := object.(loxPropertySettable)
	if !ok {
		panic(loxerr.Newf(expr, loxerr.Fatal, "property assignment is not valid for %m object", object.Type()))
	}
	value := i.evalExpr(env, expr.Value)
	settable.SetProperty(i, expr.Name, value)
	return value
}

func isTruthy(obj loxObject) loxBool {
	if truther, ok := obj.(loxTruther); ok {
		return truther.IsTruthy()
	}
	return true
}
