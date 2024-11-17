package interpreter

import (
	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/lox"
	"github.com/marcuscaisey/lox/golox/token"
)

// resolveIdents resolves the identifier tokens in a program to the declarations that they refer to.
// It returns a map from identifier tokens to the distance to the declaration of the identifier that they refer to.
// A distance of 0 means that the identifier was declared in the current scope, 1 means it was declared in the
// parent scope, and so on.
// If a token is not present in the map, then the identifier that it refers to was either declared globally or not at
// all.
// Addtionally, for local variables, checks that they are not:
//   - declared and never used
//   - declared more than once in the same scope
//   - used before they are defined
func resolveIdents(program ast.Program) (map[token.Token]int, lox.Errors) {
	r := newIdentResolver()
	return r.Resolve(program)
}

type identResolver struct {
	scopes *stack[scope]
	// map of identifier tokens to the distance to the declaration of the identifier that they refer to
	declDistancesByTok map[token.Token]int

	errs lox.Errors
}

func newIdentResolver() *identResolver {
	return &identResolver{
		scopes:             newStack[scope](),
		declDistancesByTok: map[token.Token]int{},
	}
}

func (r *identResolver) Resolve(program ast.Program) (map[token.Token]int, lox.Errors) {
	ast.Walk(program, r.walk)
	return r.declDistancesByTok, r.errs
}

type identStatus int

const (
	identStatusDeclared identStatus = iota
	identStatusDefined              = 1 << (iota - 1)
	identStatusUsed
)

type ident struct {
	Status identStatus
	Token  token.Token
}

// scope represents a lexical scope and keeps track of the identifiers declared in that scope
type scope map[string]*ident

// Declare marks an identifier as declared in the scope, unless it's [token.PlaceholderIdent].
func (s scope) Declare(name string) {
	s.DeclareFromToken(token.Token{Lexeme: name})
}

// DeclareFromToken marks an identifier as declared in the scope, unless it's [token.PlaceholderIdent].
func (s scope) DeclareFromToken(tok token.Token) {
	if tok.Lexeme == token.PlaceholderIdent {
		return
	}
	s[tok.Lexeme] = &ident{Token: tok}
}

// Define marks an identifier as defined in the scope.
func (s scope) Define(name string) {
	s[name].Status |= identStatusDefined
}

// Use marks an identifier as used in the scope.
func (s scope) Use(name string) {
	s[name].Status |= identStatusUsed
}

// IsDeclared reports whether the identifier has been declared in the scope.
func (s scope) IsDeclared(name string) bool {
	_, ok := s[name]
	return ok
}

// IsDefined reports whether the identifier has been defined in the scope.
func (s scope) IsDefined(name string) bool {
	return s[name].Status&identStatusDefined != 0
}

// UnusedIdents returns the identifier tokens in the scope that have been declared but not used.
func (s scope) UnusedIdents() []token.Token {
	var unused []token.Token
	for _, info := range s {
		if info.Status&identStatusUsed == 0 {
			unused = append(unused, info.Token)
		}
	}
	return unused
}

// beginScope creates a new scope and returns a function that ends the scope
func (r *identResolver) beginScope() func() {
	r.scopes.Push(scope{})
	return func() {
		scope := r.scopes.Pop()
		for _, tok := range scope.UnusedIdents() {
			r.errs.Addf(tok, "%s has been declared but is never used", tok.Lexeme)
		}
	}
}

func (r *identResolver) declareIdent(tok token.Token) {
	if r.scopes.Len() == 0 {
		return
	}
	if scope := r.scopes.Peek(); scope.IsDeclared(tok.Lexeme) {
		r.errs.Addf(tok, "%s has already been declared", tok.Lexeme)
	} else {
		scope.DeclareFromToken(tok)
	}
}

func (r *identResolver) defineIdent(tok token.Token) {
	for _, scope := range r.scopes.Backward() {
		if scope.IsDeclared(tok.Lexeme) {
			scope.Define(tok.Lexeme)
			return
		}
	}
	// The identifier will either be declared globally later in the program or not at all
}

type identOp int

const (
	identOpRead identOp = iota
	identOpWrite
)

func (r *identResolver) resolveIdent(tok token.Token, op identOp) {
	for i, scope := range r.scopes.Backward() {
		if scope.IsDeclared(tok.Lexeme) {
			scope.Use(tok.Lexeme)
			if !scope.IsDefined(tok.Lexeme) && op == identOpRead {
				r.errs.Addf(tok, "%s has not been defined", tok.Lexeme)
			} else {
				r.declDistancesByTok[tok] = r.scopes.Len() - 1 - i
			}
			return
		}
	}
	// The identifier will either be declared globally later in the program or not at all
}

func (r *identResolver) walk(node ast.Node) bool {
	switch node := node.(type) {
	case ast.VarDecl:
		r.walkVarDecl(node)
	case ast.FunDecl:
		r.walkFunDecl(node)
	case ast.ClassDecl:
		r.walkClassDecl(node)
	case ast.BlockStmt:
		r.walkBlockStmt(node)
	case ast.ForStmt:
		r.walkForStmt(node)
	case ast.FunExpr:
		r.walkFunExpr(node)
	case ast.VariableExpr:
		r.resolveVariableExpr(node)
	case ast.ThisExpr:
		r.resolveThisExpr(node)
	case ast.AssignmentExpr:
		r.walkAssignmentExpr(node)
	default:
		return true
	}
	return false
}

func (r *identResolver) walkVarDecl(decl ast.VarDecl) {
	if decl.Initialiser != nil {
		ast.Walk(decl.Initialiser, r.walk)
		r.declareIdent(decl.Name)
		r.defineIdent(decl.Name)
	} else {
		r.declareIdent(decl.Name)
	}
}

func (r *identResolver) walkFunDecl(decl ast.FunDecl) {
	r.declareIdent(decl.Name)
	r.defineIdent(decl.Name)
	r.walkFun(decl.Function)
}

func (r *identResolver) walkFun(fun ast.Function) {
	endScope := r.beginScope()
	defer endScope()
	for _, param := range fun.Params {
		r.declareIdent(param)
		r.defineIdent(param)
	}
	for _, stmt := range fun.Body.Stmts {
		ast.Walk(stmt, r.walk)
	}
}

func (r *identResolver) walkClassDecl(decl ast.ClassDecl) {
	r.declareIdent(decl.Name)
	r.defineIdent(decl.Name)
	endScope := r.beginScope()
	defer endScope()
	scope := r.scopes.Peek()
	scope.Declare(token.CurrentInstanceIdent)
	scope.Define(token.CurrentInstanceIdent)
	scope.Use(token.CurrentInstanceIdent)
	for _, methodDecl := range decl.Methods() {
		r.walkFun(methodDecl.Function)
	}
}

func (r *identResolver) walkBlockStmt(block ast.BlockStmt) {
	exitScope := r.beginScope()
	defer exitScope()
	for _, stmt := range block.Stmts {
		ast.Walk(stmt, r.walk)
	}
}

func (r *identResolver) walkForStmt(stmt ast.ForStmt) {
	endScope := r.beginScope()
	defer endScope()
	if stmt.Initialise != nil {
		ast.Walk(stmt.Initialise, r.walk)
	}
	if stmt.Condition != nil {
		ast.Walk(stmt.Condition, r.walk)
	}
	if stmt.Update != nil {
		ast.Walk(stmt.Update, r.walk)
	}
	ast.Walk(stmt.Body, r.walk)
}

func (r *identResolver) walkFunExpr(expr ast.FunExpr) {
	r.walkFun(expr.Function)
}

func (r *identResolver) resolveVariableExpr(expr ast.VariableExpr) {
	if expr.Name.Lexeme != token.PlaceholderIdent {
		r.resolveIdent(expr.Name, identOpRead)
	}
}

func (r *identResolver) resolveThisExpr(expr ast.ThisExpr) {
	r.resolveIdent(expr.This, identOpRead)
}

func (r *identResolver) walkAssignmentExpr(expr ast.AssignmentExpr) {
	ast.Walk(expr.Right, r.walk)
	r.resolveIdent(expr.Left, identOpWrite)
	r.defineIdent(expr.Left)
}
