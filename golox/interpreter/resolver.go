package interpreter

import (
	"fmt"
	"iter"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/lox"
	"github.com/marcuscaisey/lox/golox/token"
)

// resolve resolves the identifier tokens in a program to the declarations that they refer to.
// It returns a map from identifier tokens to the distance to the declaration of the identifier that they refer to.
// A distance of 0 means that the identifier was declared in the current scope, 1 means it was declared in the
// parent scope, and so on.
// If a token is not present in the map, then the identifier that it refers to was either declared globally or not at
// all.
func resolve(program ast.Program) (map[token.Token]int, error) {
	r := newResolver()
	return r.Resolve(program)
}

type resolver struct {
	scopes     *stack[scope]
	inLoop     bool
	curFunType funType

	// map of identifier tokens to the distance to the declaration of the identifier that they refer to
	declDistancesByTok map[token.Token]int

	errs lox.Errors
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
func (r *resolver) beginScope() func() {
	r.scopes.Push(scope{})
	return func() {
		scope := r.scopes.Pop()
		for _, tok := range scope.UnusedIdents() {
			r.errs.AddFromToken(tok, "%s has been declared but is never used", tok.Lexeme)
		}
	}
}

func (r *resolver) declareIdent(tok token.Token) {
	if r.scopes.Len() == 0 {
		return
	}
	if scope := r.scopes.Peek(); scope.IsDeclared(tok.Lexeme) {
		r.errs.AddFromToken(tok, "%s has already been declared", tok.Lexeme)
	} else {
		scope.DeclareFromToken(tok)
	}
}

func (r *resolver) defineIdent(tok token.Token) {
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

func (r *resolver) resolveIdent(tok token.Token, op identOp) {
	for i, scope := range r.scopes.Backward() {
		if scope.IsDeclared(tok.Lexeme) {
			scope.Use(tok.Lexeme)
			if !scope.IsDefined(tok.Lexeme) && op == identOpRead {
				r.errs.AddFromToken(tok, "%s has not been defined", tok.Lexeme)
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
	case ast.ClassDecl:
		r.resolveClassDecl(stmt)
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
		r.resolveBreakStmt(stmt)
	case ast.ContinueStmt:
		r.resolveContinueStmt(stmt)
	case ast.ReturnStmt:
		r.resolveReturnStmt(stmt)
	default:
		panic(fmt.Sprintf("unexpected statement type: %T", stmt))
	}
}

func (r *resolver) resolveVarDecl(stmt ast.VarDecl) {
	if stmt.Initialiser != nil {
		r.resolveExpr(stmt.Initialiser)
		r.declareIdent(stmt.Name)
		r.defineIdent(stmt.Name)
	} else {
		r.declareIdent(stmt.Name)
	}
}

func (r *resolver) resolveFunDecl(stmt ast.FunDecl) {
	r.declareIdent(stmt.Name)
	r.defineIdent(stmt.Name)
	r.resolveFun(stmt.Params, stmt.Body, funTypeFunction)
}

func (r *resolver) resolveFun(params []token.Token, body []ast.Stmt, funType funType) {
	// Break and continue are not allowed to jump out of a function so reset the loop depth to catch any invalid uses.
	prevInLoop := r.inLoop
	r.inLoop = false
	defer func() { r.inLoop = prevInLoop }()

	prevFunType := r.curFunType
	r.curFunType = funType
	defer func() { r.curFunType = prevFunType }()

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

func (r *resolver) resolveClassDecl(stmt ast.ClassDecl) {
	r.declareIdent(stmt.Name)
	r.defineIdent(stmt.Name)
	endScope := r.beginScope()
	defer endScope()
	scope := r.scopes.Peek()
	scope.Declare(token.CurrentInstanceIdent)
	scope.Define(token.CurrentInstanceIdent)
	scope.Use(token.CurrentInstanceIdent)
	for _, methodDecl := range stmt.Methods {
		r.resolveFun(methodDecl.Params, methodDecl.Body, methodFunType(methodDecl))
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
	endLoop := r.beginLoop()
	defer endLoop()
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
	endLoop := r.beginLoop()
	defer endLoop()
	r.resolveStmt(stmt.Body)
}

// beginLoop sets the inLoop flag to true and returns a function which resets it to its previous value
func (r *resolver) beginLoop() func() {
	prev := r.inLoop
	r.inLoop = true
	return func() { r.inLoop = prev }
}

func (r *resolver) resolveBreakStmt(stmt ast.BreakStmt) {
	if !r.inLoop {
		r.errs.AddFromNode(stmt, "%m can only be used inside a loop", token.Break)
	}
}

func (r *resolver) resolveContinueStmt(stmt ast.ContinueStmt) {
	if !r.inLoop {
		r.errs.AddFromNode(stmt, "%m can only be used inside a loop", token.Continue)
	}
}

func (r *resolver) resolveReturnStmt(stmt ast.ReturnStmt) {
	if r.curFunType == funTypeNone {
		r.errs.AddFromNode(stmt, "%m can only be used inside a function definition", token.Return)
	}
	if stmt.Value != nil {
		if r.curFunType.IsConstructor() {
			r.errs.AddFromNode(stmt, "%s() cannot return a value", token.ConstructorIdent)
		}
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
		// Nothing to resolve
	case ast.VariableExpr:
		r.resolveVariableExpr(expr)
	case ast.ThisExpr:
		r.resolveThisExpr(expr)
	case ast.CallExpr:
		r.resolveCallExpr(expr)
	case ast.GetExpr:
		r.resolveGetExpr(expr)
	case ast.UnaryExpr:
		r.resolveUnaryExpr(expr)
	case ast.BinaryExpr:
		r.resolveBinaryExpr(expr)
	case ast.TernaryExpr:
		r.resolveTernaryExpr(expr)
	case ast.AssignmentExpr:
		r.resolveAssignmentExpr(expr)
	case ast.SetExpr:
		r.resolveSetExpr(expr)
	default:
		panic(fmt.Sprintf("unexpected expression type: %T", expr))
	}
}

func (r *resolver) resolveFunExpr(expr ast.FunExpr) {
	r.resolveFun(expr.Params, expr.Body, funTypeFunction)
}

func (r *resolver) resolveGroupExpr(expr ast.GroupExpr) {
	r.resolveExpr(expr.Expr)
}

func (r *resolver) resolveVariableExpr(expr ast.VariableExpr) {
	if expr.Name.Lexeme == token.PlaceholderIdent {
		r.errs.AddFromToken(expr.Name, "identifier %s cannot be used in a non-assignment expression", token.PlaceholderIdent)
	} else {
		r.resolveIdent(expr.Name, identOpRead)
	}
}

func (r *resolver) resolveThisExpr(expr ast.ThisExpr) {
	if !r.curFunType.IsMethod() {
		r.errs.AddFromNode(expr, "%m can only be used inside a method definition", token.This)
	}
	r.resolveIdent(expr.This, identOpRead)
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

func (r *resolver) resolveGetExpr(expr ast.GetExpr) {
	r.resolveExpr(expr.Object)
}

func (r *resolver) resolveUnaryExpr(expr ast.UnaryExpr) {
	r.resolveExpr(expr.Right)
}

func (r *resolver) resolveAssignmentExpr(expr ast.AssignmentExpr) {
	r.resolveExpr(expr.Right)
	r.resolveIdent(expr.Left, identOpWrite)
	r.defineIdent(expr.Left)
}

func (r *resolver) resolveSetExpr(expr ast.SetExpr) {
	r.resolveExpr(expr.Value)
	r.resolveExpr(expr.Object)
}

type stack[E any] []E

func newStack[E any]() *stack[E] {
	return &stack[E]{}
}

func (s *stack[E]) Push(v E) {
	*s = append(*s, v)
}

func (s *stack[E]) Pop() E {
	if len(*s) == 0 {
		panic("pop from empty stack")
	}
	v := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return v
}

func (s *stack[E]) Peek() E {
	if len(*s) == 0 {
		panic("peek of empty stack")
	}
	return (*s)[len(*s)-1]
}

func (s *stack[E]) Len() int {
	return len(*s)
}

func (s *stack[E]) Backward() iter.Seq2[int, E] {
	return func(yield func(int, E) bool) {
		for i := s.Len() - 1; i >= 0; i-- {
			if !yield(i, (*s)[i]) {
				return
			}
		}
	}
}
