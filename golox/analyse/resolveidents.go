package analyse

import (
	"iter"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/loxerr"
	"github.com/marcuscaisey/lox/golox/stack"
	"github.com/marcuscaisey/lox/golox/token"
)

// ResolveIdents resolves the identifiers in a program to their bindings.
// It returns a map from identifier to its bindings. There will be multiple bindings associated with a single identifier
// if it's not possible to determine which binding the identifier refers to.
// If an error is returned then a possibly incomplete map will still be returned along with it. The error will be of
// type [loxerr.Errors].
// builtins is a list of builtin declarations to add to the global scope.
//
// This function also checks that identifiers are not:
//   - declared and never used
//   - declared more than once in the same scope
//   - used before they are declared (best effort for globals)
//   - used and not declared (best effort for globals)
//   - used before they are defined (best effort for globals)
//
// Some checks are best effort for global identifiers as it's not always possible to determine how they're used without
// running the program. For example, in the following example, whether the program is valid depends on whether the
// global variable x is defined before printX is called.
//
//	fun printX() {
//	    print x;
//	}
//	var x = 1;
//	printX();
//
// # Example
//
// Given the following code:
//
//	1 | class Foo {
//	2 |   method() {}
//	3 |
//	4 |   otherMethod() {
//	5 |     this.method();
//	6 |   }
//	7 | }
//	8 |
//	9 | class Bar {
//	10|   method() {}
//	11| }
//	12|
//	13| var foo = Foo();
//	14| print foo;
//	15| foo.method();
//
// The returned map is:
//
//	{
//	  1:7: Foo => [ 1:1: class Foo { ... } ],
//	  2:3: method => [ 2:3: method() {} ],
//	  4:3: otherMethod => [ 4:3: otherMethod() { ... } ],
//	  5:10: method => [ 2:3: method() {} ],
//	  9:7: Bar => [ 9:7: class Bar { ... } ],
//	  10:3: method => [ 10:3: method() {} ],
//	  13:5: foo => [ 13:1: var foo = Foo(); ],
//	  13:11: Foo => [ 1:1: class Foo { ... } ],
//	  14:7: foo => [ 13:1: var foo = Foo(); ],
//	  15:1: foo => [ 13:1: var foo = Foo(); ],
//	  15:5: method => [ 2:3: method() {}, 10:3: method() {} ],
//	}
func ResolveIdents(program *ast.Program, builtins []ast.Decl, opts ...Option) (map[*ast.Ident][]ast.Binding, error) {
	cfg := newConfig(opts)
	r := &identResolver{
		fatalOnly:              cfg.fatalOnly,
		extraFeatures:          cfg.extraFeatures,
		builtins:               builtins,
		scopes:                 stack.New[*scope](),
		forwardDeclaredGlobals: map[string]bool{},
		unresolvedPropIdentsByNameByMethodTypeByClassDecl: map[*ast.ClassDecl]map[methodType]map[string][]*ast.Ident{},
		unresolvedPropIdentsByName:                        map[string][]*ast.Ident{},
		bindingsByNameByMethodTypeByClassDecl:             map[*ast.ClassDecl]map[methodType]map[string][]ast.Binding{},
		bindingsByName:                                    map[string][]ast.Binding{},
		identBindings:                                     map[*ast.Ident][]ast.Binding{},
	}
	return r.Resolve(program)
}

type identResolver struct {
	fatalOnly     bool
	extraFeatures bool

	builtins                                          []ast.Decl
	scopes                                            *stack.Stack[*scope]
	globalScope                                       *scope
	globalDecls                                       map[string]ast.Decl
	forwardDeclaredGlobals                            map[string]bool
	inFun                                             bool
	inGlobalFun                                       bool
	funScopeLevel                                     int
	curClassDecl                                      *ast.ClassDecl
	curMethodType                                     methodType
	unresolvedPropIdentsByNameByMethodTypeByClassDecl map[*ast.ClassDecl]map[methodType]map[string][]*ast.Ident
	unresolvedPropIdentsByName                        map[string][]*ast.Ident
	bindingsByNameByMethodTypeByClassDecl             map[*ast.ClassDecl]map[methodType]map[string][]ast.Binding
	bindingsByName                                    map[string][]ast.Binding

	identBindings map[*ast.Ident][]ast.Binding
	errs          loxerr.Errors
}

func (r *identResolver) Resolve(program *ast.Program) (map[*ast.Ident][]ast.Binding, error) {
	ast.Walk(program, r.walk)
	return r.identBindings, r.errs.Err()
}

func (r *identResolver) readGlobalDecls(program *ast.Program) map[string]ast.Decl {
	decls := map[string]ast.Decl{}
	for _, stmt := range program.Stmts {
		if commentedStmt, ok := stmt.(*ast.CommentedStmt); ok {
			stmt = commentedStmt.Stmt
		}
		if decl, ok := stmt.(ast.Decl); ok {
			ident := decl.BoundIdent()
			if !ident.IsValid() {
				continue
			}
			name := ident.String()
			if _, ok := decls[name]; !ok {
				decls[name] = decl
			}
		}
	}
	return decls
}

func (r *identResolver) addErrorf(rang token.Range, typ loxerr.Type, format string, args ...any) {
	if r.fatalOnly && typ != loxerr.Fatal {
		return
	}
	r.errs.Addf(rang, typ, format, args...)
}

type declStatus int

const (
	declStatusInitialising declStatus = 1 << iota
	declStatusDefined
	declStatusUsed
)

type decl struct {
	Status declStatus
	Stmt   ast.Decl
}

// scope represents a lexical scope and keeps track of the identifiers declared in that scope
type scope struct {
	decls            map[string]*decl
	undeclaredUsages map[string][]*ast.Ident
}

func newScope() *scope {
	return &scope{
		decls:            map[string]*decl{},
		undeclaredUsages: map[string][]*ast.Ident{},
	}
}

// DeclareName marks an identifier which is not defined in code as declared in the scope.
func (s *scope) DeclareName(name string) {
	s.Declare(&ast.VarDecl{
		Name: &ast.Ident{
			Token: token.Token{Lexeme: name},
		},
	})
}

// Declare marks an identifier as declared by a statement in the scope.
func (s *scope) Declare(stmt ast.Decl) {
	decl := &decl{Stmt: stmt}
	name := stmt.BoundIdent().String()
	if _, ok := s.undeclaredUsages[name]; ok {
		decl.Status |= declStatusUsed
	}
	s.decls[name] = decl
}

// StartInitialising marks an identifier as being initialised in the scope.
func (s *scope) StartInitialising(name string) {
	s.decls[name].Status |= declStatusInitialising
}

// FinishInitialising unmarks an identifier as being initialised in the scope.
func (s *scope) FinishInitialising(name string) {
	s.decls[name].Status &= ^declStatusInitialising
}

// Declaration returns the statement which declared an identifier in the scope.
func (s *scope) Declaration(name string) ast.Decl {
	return s.decls[name].Stmt
}

// Define marks an identifier as defined in the scope.
func (s *scope) Define(name string) {
	s.decls[name].Status |= declStatusDefined
}

// Use marks an identifier as used in the scope.
func (s *scope) Use(name string) {
	s.decls[name].Status |= declStatusUsed
}

// UseUndeclared marks an undeclared identifier as used in the scope.
func (s *scope) UseUndeclared(ident *ast.Ident) {
	s.undeclaredUsages[ident.String()] = append(s.undeclaredUsages[ident.String()], ident)
}

// IsDeclared reports whether the identifier has been declared in the scope.
func (s *scope) IsDeclared(name string) bool {
	_, ok := s.decls[name]
	return ok
}

// IsInitialising reports whether the identifier is being initialised in the scope.
func (s *scope) IsInitialising(name string) bool {
	if decl, ok := s.decls[name]; ok {
		return decl.Status&declStatusInitialising != 0
	}
	return false
}

// IsDefined reports whether the identifier has been defined in the scope.
func (s *scope) IsDefined(name string) bool {
	return s.decls[name].Status&declStatusDefined != 0
}

// UnusedDeclarations returns an iterator over the declarations of names in the scope which have not been used.
func (s *scope) UnusedDeclarations() iter.Seq[ast.Decl] {
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
func (s *scope) UndeclaredUsages() iter.Seq[*ast.Ident] {
	return func(yield func(*ast.Ident) bool) {
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
		for decl := range scope.UnusedDeclarations() {
			r.addErrorf(decl.BoundIdent(), loxerr.Hint, "%s has been declared but is never used", decl.BoundIdent().String())
		}
		for ident := range scope.UndeclaredUsages() {
			if scope.IsDeclared(ident.String()) {
				r.addErrorf(ident, loxerr.Warning, "%s has been used before its declaration", ident.String())
			} else {
				r.addErrorf(ident, loxerr.Warning, "%s has not been declared", ident.String())
			}
		}
	}
}

func (r *identResolver) inGlobalScope() bool {
	return r.scopes.Len() == 1
}

func (r *identResolver) declareIdent(stmt ast.Decl) {
	ident := stmt.BoundIdent()
	if !ident.IsValid() || (r.extraFeatures && ident.String() == token.PlaceholderIdent) {
		return
	}
	if r.inGlobalScope() && r.forwardDeclaredGlobals[ident.String()] {
		if r.scopes.Peek().Declaration(ident.String()) != stmt {
			r.addErrorf(ident, loxerr.Hint, "%s has already been declared", ident.String())
		}
		return
	}
	if scope := r.scopes.Peek(); scope.IsDeclared(ident.String()) {
		typ := loxerr.Fatal
		if r.inGlobalScope() {
			typ = loxerr.Hint
		}
		r.addErrorf(ident, typ, "%s has already been declared", ident.String())
	} else {
		scope.Declare(stmt)
		r.identBindings[ident] = append(r.identBindings[ident], stmt)
	}
}

func (r *identResolver) defineIdent(ident *ast.Ident) {
	if !ident.IsValid() || (r.extraFeatures && ident.String() == token.PlaceholderIdent) {
		return
	}
	for _, scope := range r.scopes.Backward() {
		if scope.IsDeclared(ident.String()) {
			scope.Define(ident.String())
			return
		}
	}
}

func (r *identResolver) startInitialisingIdent(ident *ast.Ident) {
	if !ident.IsValid() || (r.extraFeatures && ident.String() == token.PlaceholderIdent) {
		return
	}
	for _, scope := range r.scopes.Backward() {
		if scope.IsDeclared(ident.String()) {
			scope.StartInitialising(ident.String())
			return
		}
	}
}

func (r *identResolver) finishInitialisingIdent(ident *ast.Ident) {
	if !ident.IsValid() || (r.extraFeatures && ident.String() == token.PlaceholderIdent) {
		return
	}
	for _, scope := range r.scopes.Backward() {
		if scope.IsDeclared(ident.String()) {
			scope.FinishInitialising(ident.String())
			return
		}
	}
}

type identOp int

const (
	identOpRead identOp = iota
	identOpWrite
)

func (r *identResolver) resolveIdent(ident *ast.Ident, op identOp) {
	if !ident.IsValid() || (r.extraFeatures && ident.String() == token.PlaceholderIdent) {
		return
	}
	for level, scope := range r.scopes.Backward() {
		if scope.IsDeclared(ident.String()) {
			scope.Use(ident.String())
			r.identBindings[ident] = append(r.identBindings[ident], scope.Declaration(ident.String()))
			// If we're in a function which was declared in the same or a deeper scope than the identifier was declared
			// in, then we can't definitely say that the identifier has been defined yet. It might be defined later
			// before the function is called.
			if op == identOpRead && !scope.IsDefined(ident.String()) && !(r.inFun && level <= r.funScopeLevel) { //nolint:staticcheck
				r.addErrorf(ident, loxerr.Hint, "%s has not been defined", ident.String())
			}
			return
		}
	}
	if decl, ok := r.globalDecls[ident.String()]; ok && r.inGlobalFun {
		r.globalScope.Declare(decl)
		r.identBindings[decl.BoundIdent()] = append(r.identBindings[decl.BoundIdent()], decl)
		r.globalScope.Use(ident.String())
		r.forwardDeclaredGlobals[ident.String()] = true
		r.identBindings[ident] = append(r.identBindings[ident], decl)
		return
	}
	r.scopes.Peek().UseUndeclared(ident)
}

func (r *identResolver) walk(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.Program:
		r.walkProgram(node)
	case *ast.VarDecl:
		r.walkVarDecl(node)
	case *ast.FunDecl:
		r.walkFunDecl(node)
	case *ast.Function:
		r.walkFun(node)
	case *ast.ClassDecl:
		r.walkClassDecl(node)
	case *ast.MethodDecl:
		r.walkMethodDecl(node)
	case *ast.Block:
		r.walkBlock(node)
	case *ast.ForStmt:
		r.walkForStmt(node)
	case *ast.FunExpr:
		r.walkFunExpr(node)
	case *ast.IdentExpr:
		r.resolveIdentExpr(node)
		return true
	case *ast.GetExpr:
		r.walkGetExpr(node)
	case *ast.AssignmentExpr:
		r.resolveAssignmentExpr(node)
		return true
	case *ast.SetExpr:
		r.walkSetExpr(node)
	default:
		return true
	}
	return false
}

func (r *identResolver) walkProgram(program *ast.Program) {
	endScope := r.beginScope()
	defer endScope()

	r.globalScope = r.scopes.Peek()
	for _, decl := range r.builtins {
		name := decl.BoundIdent().String()
		r.globalScope.Declare(decl)
		r.globalScope.Define(name)
		r.globalScope.Use(name)
	}
	r.globalDecls = r.readGlobalDecls(program)

	ast.WalkChildren(program, r.walk)

	for name, bindings := range r.bindingsByName {
		for _, ident := range r.unresolvedPropIdentsByName[name] {
			r.identBindings[ident] = bindings
		}
	}
	for classDecl, bindingsByNameByMethodType := range r.bindingsByNameByMethodTypeByClassDecl {
		for name, bindings := range bindingsByNameByMethodType[methodTypeStatic] {
			for _, ident := range r.unresolvedPropIdentsByNameByMethodTypeByClassDecl[classDecl][methodTypeStatic][name] {
				r.identBindings[ident] = bindings
			}
		}
	}
}

func (r *identResolver) walkVarDecl(decl *ast.VarDecl) {
	if decl.Initialiser != nil {
		if r.inGlobalScope() {
			ast.Walk(decl.Initialiser, r.walk)
			r.declareIdent(decl)
		} else {
			r.declareIdent(decl)
			r.startInitialisingIdent(decl.Name)
			ast.Walk(decl.Initialiser, r.walk)
			r.finishInitialisingIdent(decl.Name)
		}
		r.defineIdent(decl.Name)
	} else {
		r.declareIdent(decl)
	}
}

func (r *identResolver) walkFunDecl(decl *ast.FunDecl) {
	r.declareIdent(decl)
	r.defineIdent(decl.Name)
	prevFunScopeLevel := r.funScopeLevel
	r.funScopeLevel = r.scopes.Len() - 1
	defer func() { r.funScopeLevel = prevFunScopeLevel }()
	ast.WalkChildren(decl, r.walk)
}

func (r *identResolver) walkFun(fun *ast.Function) {
	endScope := r.beginScope()
	defer endScope()

	prevInFun := r.inFun
	r.inFun = true
	defer func() { r.inFun = prevInFun }()
	if r.scopes.Len() == 2 {
		r.inGlobalFun = true
		defer func() { r.inGlobalFun = false }()
	}

	for _, param := range fun.Params {
		r.declareIdent(param)
		r.defineIdent(param.Name)
	}
	// We don't walk over the body using ast.Walk(fun.Body, r.walk) because this would introduce another scope which
	// would allow redeclaration of the parameters.
	ast.WalkChildren(fun.Body, r.walk)
}

func (r *identResolver) walkClassDecl(decl *ast.ClassDecl) {
	prevCurClassDecl := r.curClassDecl
	defer func() { r.curClassDecl = prevCurClassDecl }()
	r.curClassDecl = decl
	//nolint:exhaustive
	r.unresolvedPropIdentsByNameByMethodTypeByClassDecl[decl] = map[methodType]map[string][]*ast.Ident{
		methodTypeInstance: {},
		methodTypeStatic:   {},
	}
	//nolint:exhaustive
	r.bindingsByNameByMethodTypeByClassDecl[decl] = map[methodType]map[string][]ast.Binding{
		methodTypeInstance: {},
		methodTypeStatic:   {},
	}

	r.declareIdent(decl)
	r.defineIdent(decl.Name)

	endScope := r.beginScope()
	defer endScope()

	prevFunScopeLevel := r.funScopeLevel
	r.funScopeLevel = r.scopes.Len() - 1
	defer func() { r.funScopeLevel = prevFunScopeLevel }()

	if r.scopes.Len() == 2 {
		r.inGlobalFun = true
		defer func() { r.inGlobalFun = false }()
	}

	scope := r.scopes.Peek()
	scope.DeclareName(token.CurrentInstanceIdent)
	scope.Define(token.CurrentInstanceIdent)
	scope.Use(token.CurrentInstanceIdent)

	ast.WalkChildren(decl, r.walk)

	for name, bindings := range r.bindingsByNameByMethodTypeByClassDecl[decl][methodTypeInstance] {
		for _, ident := range r.unresolvedPropIdentsByNameByMethodTypeByClassDecl[decl][methodTypeInstance][name] {
			r.identBindings[ident] = bindings
		}
	}
}

type methodType int

const (
	methodTypeNone methodType = iota
	methodTypeInstance
	methodTypeStatic
)

func (r *identResolver) walkMethodDecl(decl *ast.MethodDecl) {
	prevCurMethodType := r.curMethodType
	if decl.HasModifier(token.Static) {
		r.curMethodType = methodTypeStatic
	} else {
		r.curMethodType = methodTypeInstance
	}
	defer func() { r.curMethodType = prevCurMethodType }()

	if decl.Name.IsValid() {
		r.identBindings[decl.Name] = append(r.identBindings[decl.Name], decl)
		name := decl.Name.String()
		r.bindingsByNameByMethodTypeByClassDecl[r.curClassDecl][r.curMethodType][name] = append(
			r.bindingsByNameByMethodTypeByClassDecl[r.curClassDecl][r.curMethodType][name], decl)
		r.bindingsByName[name] = append(r.bindingsByName[name], decl)
	}

	ast.WalkChildren(decl, r.walk)
}

func (r *identResolver) walkBlock(block *ast.Block) {
	exitScope := r.beginScope()
	defer exitScope()
	ast.WalkChildren(block, r.walk)
}

func (r *identResolver) walkForStmt(stmt *ast.ForStmt) {
	endScope := r.beginScope()
	defer endScope()
	ast.WalkChildren(stmt, r.walk)
}

func (r *identResolver) walkFunExpr(expr *ast.FunExpr) {
	prevFunScopeLevel := r.funScopeLevel
	r.funScopeLevel = r.scopes.Len() - 1
	defer func() { r.funScopeLevel = prevFunScopeLevel }()
	ast.WalkChildren(expr, r.walk)
}

func (r *identResolver) resolveIdentExpr(expr *ast.IdentExpr) {
	if !r.inGlobalScope() && r.scopes.Peek().IsInitialising(expr.Ident.String()) {
		r.addErrorf(expr, loxerr.Fatal, "%s read in its own initialiser", expr.Ident.String())
		return
	}
	r.resolveIdent(expr.Ident, identOpRead)
}

func (r *identResolver) walkGetExpr(expr *ast.GetExpr) {
	ast.WalkChildren(expr, r.walk)

	if !expr.Name.IsValid() {
		return
	}
	name := expr.Name.String()

	if _, ok := expr.Object.(*ast.ThisExpr); ok && r.curClassDecl != nil && r.curMethodType != methodTypeNone {
		r.unresolvedPropIdentsByNameByMethodTypeByClassDecl[r.curClassDecl][r.curMethodType][name] = append(
			r.unresolvedPropIdentsByNameByMethodTypeByClassDecl[r.curClassDecl][r.curMethodType][name], expr.Name)
		return
	}

	if identExpr, ok := expr.Object.(*ast.IdentExpr); ok {
		if bindings, ok := r.identBindings[identExpr.Ident]; ok {
			if classDecl, ok := bindings[0].(*ast.ClassDecl); ok {
				r.unresolvedPropIdentsByNameByMethodTypeByClassDecl[classDecl][methodTypeStatic][name] = append(
					r.unresolvedPropIdentsByNameByMethodTypeByClassDecl[classDecl][methodTypeStatic][name], expr.Name)
				return
			}
		}
	}

	r.unresolvedPropIdentsByName[name] = append(r.unresolvedPropIdentsByName[name], expr.Name)
}

func (r *identResolver) resolveAssignmentExpr(expr *ast.AssignmentExpr) {
	r.resolveIdent(expr.Left, identOpWrite)
	r.defineIdent(expr.Left)
}

func (r *identResolver) walkSetExpr(expr *ast.SetExpr) {
	ast.WalkChildren(expr, r.walk)

	if !expr.Name.IsValid() {
		return
	}
	name := expr.Name.String()

	r.bindingsByName[name] = append(r.bindingsByName[name], expr)

	if _, ok := expr.Object.(*ast.ThisExpr); ok && r.curClassDecl != nil && r.curMethodType != methodTypeNone {
		r.unresolvedPropIdentsByNameByMethodTypeByClassDecl[r.curClassDecl][r.curMethodType][name] = append(
			r.unresolvedPropIdentsByNameByMethodTypeByClassDecl[r.curClassDecl][r.curMethodType][name], expr.Name)
		r.bindingsByNameByMethodTypeByClassDecl[r.curClassDecl][r.curMethodType][name] = append(
			r.bindingsByNameByMethodTypeByClassDecl[r.curClassDecl][r.curMethodType][name], expr)
		return
	}

	if identExpr, ok := expr.Object.(*ast.IdentExpr); ok {
		if bindings, ok := r.identBindings[identExpr.Ident]; ok {
			if classDecl, ok := bindings[0].(*ast.ClassDecl); ok {
				r.unresolvedPropIdentsByNameByMethodTypeByClassDecl[classDecl][methodTypeStatic][name] = append(
					r.unresolvedPropIdentsByNameByMethodTypeByClassDecl[classDecl][methodTypeStatic][name], expr.Name)
				r.bindingsByNameByMethodTypeByClassDecl[classDecl][methodTypeStatic][name] = append(
					r.bindingsByNameByMethodTypeByClassDecl[classDecl][methodTypeStatic][name], expr)
				return
			}
		}
	}

	r.unresolvedPropIdentsByName[name] = append(r.unresolvedPropIdentsByName[name], expr.Name)
}
