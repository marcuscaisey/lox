package analysis

import (
	"iter"

	"github.com/marcuscaisey/lox/lox"
	"github.com/marcuscaisey/lox/lox/ast"
	"github.com/marcuscaisey/lox/lox/stack"
	"github.com/marcuscaisey/lox/lox/token"
)

// ResolveIdentsOption can be passed to [ResolveIdents] to configure the resolving behaviour.
type ResolveIdentsOption func(*identResolver)

// WithREPLMode configures identifiers to be resolved in REPL mode.
// In REPL mode, the following identifier checks are disabled:
//   - declared and never used
//   - declared more than once in the same scope
//   - used before they are declared
func WithREPLMode() ResolveIdentsOption {
	return func(i *identResolver) {
		i.replMode = true
	}
}

// ResolveIdents resolves the identifiers in a program to their declarations.
// It returns a map from identifier to the statement which declared it. If an error is returned then a possibly
// incomplete map will still be returned along with it.
//
// For example, given the following code:
//
//	1| var a = 1;
//	2|
//	3| print a;
//	4| print a + 1;
//
// The returned map is:
//
//	{
//	  1:5: a => 1:5: var a = 1;,
//	  3:7: a => 1:5: var a = 1;,
//	  4:7: a => 1:5: var a = 1;,
//	}
//
// This function also checks that identifiers are not:
//   - declared and never used
//   - declared more than once in the same scope
//   - used before they are declared (best effort for globals)
//   - used and not declared (best effort for globals)
//   - used before they are defined (best effort for globals)
//
// Some checks are best effort for global identifiers as it's not always possible to (easily) determine how they're used
// without running the program. For example, in the following example, whether the program is valid depends on whether
// the global variable x is defined before printX is called.
//
//	fun printX() {
//	    print x;
//	}
//	var x = 1;
//	printX();
func ResolveIdents(program ast.Program, opts ...ResolveIdentsOption) (map[ast.Ident]ast.Decl, lox.Errors) {
	r := newIdentResolver(program, opts...)
	return r.Resolve()
}

type identResolver struct {
	program ast.Program

	scopes                 *stack.Stack[scope]
	globalScope            scope
	globalDecls            map[string]ast.Decl
	forwardDeclaredGlobals map[string]bool
	inFun                  bool
	funScopeLevel          int

	identDecls map[ast.Ident]ast.Decl
	errs       lox.Errors

	replMode bool
}

func newIdentResolver(program ast.Program, opts ...ResolveIdentsOption) *identResolver {
	r := &identResolver{
		program:                program,
		scopes:                 stack.New[scope](),
		forwardDeclaredGlobals: map[string]bool{},
		identDecls:             map[ast.Ident]ast.Decl{},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *identResolver) Resolve() (map[ast.Ident]ast.Decl, lox.Errors) {
	r.resolve()
	return r.identDecls, r.errs
}

func (r *identResolver) resolve() {
	endScope := r.beginScope()
	defer endScope()
	r.globalScope = r.scopes.Peek()
	r.declareBuiltins(r.globalScope)
	r.globalDecls = r.readGlobalDecls(r.program)
	ast.Walk(r.program, r.walk)
}

func (r *identResolver) readGlobalDecls(program ast.Program) map[string]ast.Decl {
	decls := map[string]ast.Decl{}
	for _, stmt := range program.Stmts {
		if inlineCommentStmt, ok := stmt.(ast.InlineCommentStmt); ok {
			stmt = inlineCommentStmt.Stmt
		}
		if decl, ok := stmt.(ast.Decl); ok {
			decls[decl.Ident().Token.Lexeme] = decl
		}
	}
	return decls
}

func (r *identResolver) declareBuiltins(scope scope) {
	for _, name := range lox.AllBuiltins {
		scope.DeclareName(name)
		scope.Define(name)
		scope.Use(name) // We don't want to raise unused declaration errors for builtins.
	}
}

type declStatus int

const (
	declStatusDefined declStatus = 1 << iota
	declStatusUsed
)

type decl struct {
	Status declStatus
	Stmt   ast.Decl
}

// scope represents a lexical scope and keeps track of the identifiers declared in that scope
type scope struct {
	decls            map[string]*decl
	undeclaredUsages map[string][]ast.Ident
}

func newScope() scope {
	return scope{
		decls:            map[string]*decl{},
		undeclaredUsages: map[string][]ast.Ident{},
	}
}

// DeclareName marks an identifier which is not defined in code as declared in the scope.
func (s scope) DeclareName(name string) {
	// TODO: This makes builtins appear as if they're variables (i.e. when hovered over in the editor). Come up with a
	// better way to handle these.
	s.Declare(ast.VarDecl{
		Name: ast.Ident{
			Token: token.Token{Lexeme: name},
		},
	})
}

// Declare marks an identifier as declared by a statement in the scope.
func (s scope) Declare(stmt ast.Decl) {
	decl := &decl{Stmt: stmt}
	name := stmt.Ident().Token.Lexeme
	if _, ok := s.undeclaredUsages[name]; ok {
		decl.Status |= declStatusUsed
	}
	s.decls[name] = decl
}

// Declaration returns the statement which declared an identifier in the scope.
func (s scope) Declaration(name string) ast.Decl {
	return s.decls[name].Stmt
}

// Define marks an identifier as defined in the scope.
func (s scope) Define(name string) {
	s.decls[name].Status |= declStatusDefined
}

// Use marks an identifier as used in the scope.
func (s scope) Use(name string) {
	s.decls[name].Status |= declStatusUsed
}

// UseUndeclared marks an undeclared identifier as used in the scope.
func (s scope) UseUndeclared(ident ast.Ident) {
	s.undeclaredUsages[ident.Token.Lexeme] = append(s.undeclaredUsages[ident.Token.Lexeme], ident)
}

// IsDeclared reports whether the identifier has been declared in the scope.
func (s scope) IsDeclared(name string) bool {
	_, ok := s.decls[name]
	return ok
}

// IsDefined reports whether the identifier has been defined in the scope.
func (s scope) IsDefined(name string) bool {
	return s.decls[name].Status&declStatusDefined != 0
}

// UnusedDeclarations returns an iterator over the declarations of names in the scope which have not been used.
func (s scope) UnusedDeclarations() iter.Seq[ast.Decl] {
	return func(yield func(ast.Decl) bool) {
		for _, decl := range s.decls {
			if decl.Status&declStatusUsed == 0 {
				if !yield(decl.Stmt) {
					return
				}
			}
		}
	}
}

// UndeclaredIdents returns an iterator over the identifiers in the scope that were used before they were declared.
func (s scope) UndeclaredUsages() iter.Seq[ast.Ident] {
	return func(yield func(ast.Ident) bool) {
		for _, idents := range s.undeclaredUsages {
			for _, ident := range idents {
				if !yield(ident) {
					return
				}
			}
		}
	}
}

// beginScope creates a new scope and returns a function that ends the scope.
func (r *identResolver) beginScope() func() {
	r.scopes.Push(newScope())
	return func() {
		scope := r.scopes.Pop()
		if r.replMode {
			return
		}
		for decl := range scope.UnusedDeclarations() {
			r.errs.Addf(decl.Ident(), "%s has been declared but is never used", decl.Ident().Token.Lexeme)
		}
		for ident := range scope.UndeclaredUsages() {
			if scope.IsDeclared(ident.Token.Lexeme) {
				r.errs.Addf(ident, "%s has been used before its declaration", ident.Token.Lexeme)
			} else {
				r.errs.Addf(ident, "%s has not been declared", ident.Token.Lexeme)
			}
		}
	}
}

func (r *identResolver) declareIdent(stmt ast.Decl) {
	ident := stmt.Ident()
	if ident.Token.Lexeme == token.PlaceholderIdent {
		return
	}
	if r.scopes.Len() == 1 && r.forwardDeclaredGlobals[ident.Token.Lexeme] {
		return
	}
	if scope := r.scopes.Peek(); scope.IsDeclared(ident.Token.Lexeme) {
		r.errs.Addf(ident, "%s has already been declared", ident.Token.Lexeme)
	} else {
		scope.Declare(stmt)
		r.identDecls[ident] = stmt
	}
}

func (r *identResolver) defineIdent(ident ast.Ident) {
	if ident.Token.Lexeme == token.PlaceholderIdent {
		return
	}
	for _, scope := range r.scopes.Backward() {
		if scope.IsDeclared(ident.Token.Lexeme) {
			scope.Define(ident.Token.Lexeme)
			return
		}
	}
}

type identOp int

const (
	identOpRead identOp = iota
	identOpWrite
)

func (r *identResolver) resolveIdent(ident ast.Ident, op identOp) {
	if ident.Token.Lexeme == token.PlaceholderIdent {
		return
	}
	for level, scope := range r.scopes.Backward() {
		if scope.IsDeclared(ident.Token.Lexeme) {
			scope.Use(ident.Token.Lexeme)
			r.identDecls[ident] = scope.Declaration(ident.Token.Lexeme)
			// If we're in a function which was declared in the same or a deeper scope than the identifier was declared
			// in, then we can't definitely say that the identifier has been defined yet. It might be defined later
			// before the function is called.
			if op == identOpRead && !scope.IsDefined(ident.Token.Lexeme) && !(r.inFun && level <= r.funScopeLevel) { //nolint:staticcheck
				r.errs.Addf(ident, "%s has not been defined", ident.Token.Lexeme)
			}
			return
		}
	}
	if decl, ok := r.globalDecls[ident.Token.Lexeme]; ok && r.inFun {
		r.globalScope.Declare(decl)
		r.globalScope.Use(ident.Token.Lexeme)
		r.forwardDeclaredGlobals[ident.Token.Lexeme] = true
		r.identDecls[ident] = decl
		return
	}
	r.scopes.Peek().UseUndeclared(ident)
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
	case ast.IdentExpr:
		r.resolveIdentExpr(node)
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
		r.declareIdent(decl)
		r.defineIdent(decl.Name)
	} else {
		r.declareIdent(decl)
	}
}

func (r *identResolver) walkFunDecl(decl ast.FunDecl) {
	r.declareIdent(decl)
	r.defineIdent(decl.Name)
	prevFunScopeLevel := r.funScopeLevel
	r.funScopeLevel = r.scopes.Len() - 1
	defer func() { r.funScopeLevel = prevFunScopeLevel }()
	r.walkFun(decl.Function)
}

func (r *identResolver) walkFun(fun ast.Function) {
	endScope := r.beginScope()
	defer endScope()

	prevInFun := r.inFun
	r.inFun = true
	defer func() { r.inFun = prevInFun }()

	for _, param := range fun.Params {
		r.declareIdent(param)
		r.defineIdent(param.Name)
	}
	for _, stmt := range fun.Body.Stmts {
		ast.Walk(stmt, r.walk)
	}
}

func (r *identResolver) walkClassDecl(decl ast.ClassDecl) {
	r.declareIdent(decl)
	r.defineIdent(decl.Name)
	endScope := r.beginScope()
	defer endScope()
	scope := r.scopes.Peek()
	scope.DeclareName(token.CurrentInstanceIdent)
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

func (r *identResolver) resolveIdentExpr(expr ast.IdentExpr) {
	r.resolveIdent(expr.Ident, identOpRead)
}

func (r *identResolver) walkAssignmentExpr(expr ast.AssignmentExpr) {
	ast.Walk(expr.Right, r.walk)
	r.resolveIdent(expr.Left, identOpWrite)
	r.defineIdent(expr.Left)
}
