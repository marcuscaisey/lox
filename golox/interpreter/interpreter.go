// Package interpreter implements an interpreter of Lox programs.
package interpreter

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/marcuscaisey/lox/golox/analysis"
	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/loxerr"
	"github.com/marcuscaisey/lox/golox/stubbuiltins"
	"github.com/marcuscaisey/lox/golox/token"
)

// Interpreter is the interpreter for the language.
type Interpreter struct {
	globals      environment
	callStack    *callStack
	stubBuiltins []ast.Decl

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
	for name, builtin := range builtins {
		globals = globals.Define(name, builtin)
	}
	interpreter := &Interpreter{
		globals:      globals,
		callStack:    newCallStack(),
		stubBuiltins: stubbuiltins.MustParse("builtins.lox"),
	}
	for _, opt := range opts {
		opt(interpreter)
	}
	return interpreter
}

// Interpret executes a program and returns an error if one occurred.
// Interpret can be called multiple times with different programs and the state will be maintained between calls.
func (i *Interpreter) Interpret(program *ast.Program) error {
	_, errs := analysis.ResolveIdents(program, i.stubBuiltins, analysis.WithREPLMode(i.replMode))
	errs = append(errs, analysis.CheckSemantics(program)...)
	errs = errs.Fatal()
	if err := errs.Err(); err != nil {
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
	if stmt.Name.Token.Lexeme == token.PlaceholderIdent {
		return env
	}
	newEnv := env.Declare(stmt.Name)
	if stmt.Initialiser != nil {
		newEnv.Assign(stmt.Name, value)
	}
	return newEnv
}

func (i *Interpreter) execFunDecl(env environment, stmt *ast.FunDecl) environment {
	if stmt.Name.Token.Lexeme == token.PlaceholderIdent {
		return env
	}
	newEnv := env.Declare(stmt.Name)
	newEnv.Assign(stmt.Name, newLoxFunction(stmt.Name.Token.Lexeme, stmt.Function, funTypeFunction, newEnv))
	return newEnv
}

func (i *Interpreter) execClassDecl(env environment, stmt *ast.ClassDecl) environment {
	if stmt.Name.Token.Lexeme == token.PlaceholderIdent {
		return env
	}
	newEnv := env.Declare(stmt.Name)
	newEnv.Assign(stmt.Name, newLoxClass(stmt.Name.Token.Lexeme, stmt.Methods(), newEnv))
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
	case *ast.IdentExpr:
		return i.evalIdentExpr(env, expr)
	case *ast.ThisExpr:
		return i.evalThisExpr(env, expr)
	case *ast.CallExpr:
		return i.evalCallExpr(env, expr)
	case *ast.GetExpr:
		return i.evalGetExpr(env, expr)
	case *ast.UnaryExpr:
		return i.evalUnaryExpr(env, expr)
	case *ast.BinaryExpr:
		return i.evalBinaryExpr(env, expr)
	case *ast.TernaryExpr:
		return i.evalTernaryExpr(env, expr)
	case *ast.AssignmentExpr:
		return i.evalAssignmentExpr(env, expr)
	case *ast.SetExpr:
		return i.evalSetExpr(env, expr)
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

func (i *Interpreter) evalIdentExpr(env environment, expr *ast.IdentExpr) loxObject {
	return env.Get(expr.Ident)
}

func (i *Interpreter) evalThisExpr(env environment, expr *ast.ThisExpr) loxObject {
	return env.Get(&ast.Ident{Token: expr.This})
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
		panic(loxerr.Newf(
			expr,
			loxerr.Fatal, "%s() missing %d argument%s: %s", callable.CallableName(), arity-len(args), argumentSuffix, missingArgsStr,
		))
	case len(args) > arity:
		panic(loxerr.NewSpanningRangesf(
			expr.Args[arity],
			expr.Args[len(expr.Args)-1],
			loxerr.Fatal, "%s() accepts %d arguments but %d were given", callable.CallableName(), arity, len(args),
		))
	}

	result := i.call(expr.Start(), callable, args)
	if errorMsg, ok := result.(errorMsg); ok {
		panic(loxerr.Newf(expr, loxerr.Fatal, "%s", string(errorMsg)))
	}
	return result
}

func (i *Interpreter) call(location token.Position, callable loxCallable, args []loxObject) loxObject {
	i.callStack.Push(callable.CallableName(), location)
	result := callable.Call(i, args)
	i.callStack.Pop()
	return result
}

func (i *Interpreter) evalGetExpr(env environment, expr *ast.GetExpr) loxObject {
	object := i.evalExpr(env, expr.Object)
	getter, ok := object.(loxGetter)
	if !ok {
		panic(loxerr.Newf(expr, loxerr.Fatal, "property access is not valid for %m object", object.Type()))
	}
	return getter.Get(i, expr.Name)
}

func (i *Interpreter) evalUnaryExpr(env environment, expr *ast.UnaryExpr) loxObject {
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
	panic(loxerr.Newf(expr.Op, loxerr.Fatal, "%m operator cannot be used with type %m", expr.Op.Type, right.Type()))
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
		panic(loxerr.Newf(expr.Op, loxerr.Fatal, "%m operator cannot be used with types %m and %m", expr.Op.Type, left.Type(), right.Type()))
	}
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
	if expr.Left.Token.Lexeme != token.PlaceholderIdent {
		env.Assign(expr.Left, value)
	}
	return value
}

func (i *Interpreter) evalSetExpr(env environment, expr *ast.SetExpr) loxObject {
	object := i.evalExpr(env, expr.Object)
	setter, ok := object.(loxSetter)
	if !ok {
		panic(loxerr.Newf(expr, loxerr.Fatal, "property assignment is not valid for %m object", object.Type()))
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
