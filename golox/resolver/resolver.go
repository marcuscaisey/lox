// Package resolver implements the resolution of identifier tokens in a Lox program.
package resolver

import (
	"fmt"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/loxerror"
	"github.com/marcuscaisey/lox/golox/token"
)

// Resolve resolves the identifier tokens in a program to the declarations that they refer to.
// It returns a map from identifier tokens to the distance to the declaration of the identifier that they refer.
// to. A distance of 0 means that the identifier was declared in the current scope, 1 means it was declared in the
// parent scope, and so on.
// If a token is not present in the map, then the identifier that it refers to was either declared globally or not at
// all.
func Resolve(program ast.Program) (map[token.Token]int, error) {
	r := newResolver()
	return r.Resolve(program)
}

type identStatus int

const (
	undeclared identStatus = iota
	declared
	defined
)

// scope represents a lexical scope and keeps track of which identifiers have been declared and defined in that scope.
type scope map[string]identStatus

// Declare marks an identifier as declared in the scope.
func (s scope) Declare(ident string) {
	s[ident] = declared
}

// Define marks an identifier as defined in the scope.
func (s scope) Define(ident string) {
	s[ident] = defined
}

// IsDeclared returns true if the identifier has been declared in the scope.
func (s scope) IsDeclared(ident string) bool {
	return s[ident] != undeclared
}

// IsDefined returns true if the identifier has been defined in the scope.
func (s scope) IsDefined(ident string) bool {
	return s[ident] == defined
}

type resolver struct {
	// stack of lexical scopes where each scope maps identifiers to their status in that scope
	scopes *stack[scope]
	// whether we're currently resolving a variable declaration
	inVarDecl bool

	// declDistancesByTok maps identifier tokens to the distance to the declaration of the identifier that they refer to
	declDistancesByTok map[token.Token]int

	errs loxerror.LoxErrors
}

func newResolver() *resolver {
	return &resolver{
		scopes:             newStack[scope](),
		declDistancesByTok: map[token.Token]int{},
	}
}

func (r *resolver) Resolve(program ast.Program) (map[token.Token]int, error) {
	r.resolveProgram(program)
	if err := r.errs.Err(); err != nil {
		return nil, err
	}
	return r.declDistancesByTok, nil
}

func (r *resolver) beginScope() func() {
	r.scopes.Push(scope{})
	return func() {
		r.scopes.Pop()
	}
}

func (r *resolver) declareIdent(tok token.Token) {
	if r.scopes.Len() == 0 {
		return
	}
	if scope := r.scopes.Peek(); scope.IsDeclared(tok.Literal) {
		r.errs.AddFromToken(tok, "%s has already been declared", tok.Literal)
	} else {
		scope.Declare(tok.Literal)
	}
}

func (r *resolver) defineIdent(tok token.Token) {
	for i := r.scopes.Len() - 1; i >= 0; i-- {
		if scope := r.scopes.Index(i); scope.IsDeclared(tok.Literal) {
			scope.Define(tok.Literal)
			return
		}
	}
	// The identifier will either be declared globally later in the program or not at all
}

type identOp int

const (
	read identOp = iota
	write
)

func (r *resolver) resolveIdent(tok token.Token, op identOp) {
	for i := r.scopes.Len() - 1; i >= 0; i-- {
		if scope := r.scopes.Index(i); scope.IsDeclared(tok.Literal) {
			if !scope.IsDefined(tok.Literal) && op == read {
				r.errs.AddFromToken(tok, "%s has not been defined", tok.Literal)
			} else {
				r.declDistancesByTok[tok] = r.scopes.Len() - 1 - i
			}
			return
		}
	}
	// The identifier will either be declared globally later in the program or not at all
}

func (r *resolver) resolveProgram(program ast.Program) {
	for _, stmt := range program.Stmts {
		r.resolveStmt(stmt)
	}
}

func (r *resolver) resolveStmt(stmt ast.Stmt) {
	switch stmt := stmt.(type) {
	case ast.VarDecl:
		r.resolveVarDecl(stmt)
	case ast.FunDecl:
		r.resolveFunDecl(stmt)
	case ast.ExprStmt:
		r.resolveExprStmt(stmt)
	case ast.PrintStmt:
		r.resolvePrintStmt(stmt)
	case ast.BlockStmt:
		r.resolveBlockStmt(stmt)
	case ast.IfStmt:
		r.resolveIfStmt(stmt)
	case ast.WhileStmt:
		r.resolveWhileStmt(stmt)
	case ast.ForStmt:
		r.resolveForStmt(stmt)
	case ast.BreakStmt:
		r.resolveBreakStmt()
	case ast.ContinueStmt:
		r.resolveContinueStmt()
	case ast.ReturnStmt:
		r.resolveReturnStmt(stmt)
	default:
		panic(fmt.Sprintf("unexpected statement type: %T", stmt))
	}
}

func (r *resolver) resolveVarDecl(stmt ast.VarDecl) {
	r.declareIdent(stmt.Name)
	if stmt.Initialiser != nil {
		r.inVarDecl = true
		defer func() { r.inVarDecl = false }()
		r.resolveExpr(stmt.Initialiser)
		r.defineIdent(stmt.Name)
	}
}

func (r *resolver) resolveFunDecl(stmt ast.FunDecl) {
	r.declareIdent(stmt.Name)
	r.defineIdent(stmt.Name)
	r.resolveFun(stmt.Params, stmt.Body)
}

func (r *resolver) resolveFun(params []token.Token, body []ast.Stmt) {
	endScope := r.beginScope()
	defer endScope()
	for _, param := range params {
		r.declareIdent(param)
		r.defineIdent(param)
	}
	for _, stmt := range body {
		r.resolveStmt(stmt)
	}
}

func (r *resolver) resolveExprStmt(stmt ast.ExprStmt) {
	r.resolveExpr(stmt.Expr)
}

func (r *resolver) resolvePrintStmt(stmt ast.PrintStmt) {
	r.resolveExpr(stmt.Expr)
}

func (r *resolver) resolveBlockStmt(stmt ast.BlockStmt) {
	exitScope := r.beginScope()
	defer exitScope()
	for _, stmt := range stmt.Stmts {
		r.resolveStmt(stmt)
	}
}

func (r *resolver) resolveIfStmt(stmt ast.IfStmt) {
	r.resolveExpr(stmt.Condition)
	r.resolveStmt(stmt.Then)
	if stmt.Else != nil {
		r.resolveStmt(stmt.Else)
	}
}

func (r *resolver) resolveWhileStmt(stmt ast.WhileStmt) {
	r.resolveExpr(stmt.Condition)
	r.resolveStmt(stmt.Body)
}

func (r *resolver) resolveForStmt(stmt ast.ForStmt) {
	endScope := r.beginScope()
	defer endScope()
	if stmt.Initialise != nil {
		r.resolveStmt(stmt.Initialise)
	}
	if stmt.Condition != nil {
		r.resolveExpr(stmt.Condition)
	}
	if stmt.Update != nil {
		r.resolveExpr(stmt.Update)
	}
	r.resolveStmt(stmt.Body)
}

func (r *resolver) resolveBreakStmt() {
	// Nothing to resolve
}

func (r *resolver) resolveContinueStmt() {
	// Nothing to resolve
}

func (r *resolver) resolveReturnStmt(stmt ast.ReturnStmt) {
	if stmt.Value != nil {
		r.resolveExpr(stmt.Value)
	}
}

func (r *resolver) resolveExpr(expr ast.Expr) {
	switch expr := expr.(type) {
	case ast.FunExpr:
		r.resolveFunExpr(expr)
	case ast.GroupExpr:
		r.resolveGroupExpr(expr)
	case ast.LiteralExpr:
		r.resolveLiteralExpr(expr)
	case ast.VariableExpr:
		r.resolveVariableExpr(expr)
	case ast.CallExpr:
		r.resolveCallExpr(expr)
	case ast.UnaryExpr:
		r.resolveUnaryExpr(expr)
	case ast.BinaryExpr:
		r.resolveBinaryExpr(expr)
	case ast.TernaryExpr:
		r.resolveTernaryExpr(expr)
	case ast.AssignmentExpr:
		r.resolveAssignmentExpr(expr)
	default:
		panic(fmt.Sprintf("unexpected expression type: %T", expr))
	}
}

func (r *resolver) resolveFunExpr(expr ast.FunExpr) {
	r.resolveFun(expr.Params, expr.Body)
}

func (r *resolver) resolveGroupExpr(expr ast.GroupExpr) {
	r.resolveExpr(expr.Expr)
}

func (r *resolver) resolveLiteralExpr(ast.LiteralExpr) {
	// Nothing to resolve
}

func (r *resolver) resolveVariableExpr(expr ast.VariableExpr) {
	if r.inVarDecl && r.scopes.Len() > 0 && r.scopes.Peek().IsDeclared(expr.Name.Literal) {
		r.errs.AddFromToken(expr.Name, "variable definition cannot refer to itself")
	} else {
		r.resolveIdent(expr.Name, read)
	}
}

func (r *resolver) resolveBinaryExpr(expr ast.BinaryExpr) {
	r.resolveExpr(expr.Left)
	r.resolveExpr(expr.Right)
}

func (r *resolver) resolveTernaryExpr(expr ast.TernaryExpr) {
	r.resolveExpr(expr.Condition)
	r.resolveExpr(expr.Then)
	r.resolveExpr(expr.Else)
}

func (r *resolver) resolveCallExpr(expr ast.CallExpr) {
	r.resolveExpr(expr.Callee)
	for _, arg := range expr.Args {
		r.resolveExpr(arg)
	}
}

func (r *resolver) resolveUnaryExpr(expr ast.UnaryExpr) {
	r.resolveExpr(expr.Right)
}

func (r *resolver) resolveAssignmentExpr(expr ast.AssignmentExpr) {
	r.resolveExpr(expr.Right)
	r.resolveIdent(expr.Left, write)
	r.defineIdent(expr.Left)
}
