// Package resolver implements the resolution of identifiers in a Lox program.
package resolver

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/token"
)

// Resolve resolves the identifiers in the given program.
// It returns a map from identifiers which were declared locally to the distance from their lexical scope to the one
// where they were declared. A distance of 0 means the identifier was declared in its current scope, 1 means it was
// declared in the parent scope, and so on.
// If an identifier is not present in the map, then it was declared in the global scope.
func Resolve(program ast.Program) (map[token.Token]int, error) {
	r := newResolver()
	return r.Resolve(program)
}

type resolver struct {
	// scopes is a stack of lexical scopes where each scope maps identifiers to whether they've been defined
	scopes *stack[map[string]bool]
	// localDeclDistancesByIdent maps identifiers which were declared locally to the distance from their current lexical
	// scope to the one where they were declared
	localDeclDistancesByIdent map[token.Token]int
}

func newResolver() *resolver {
	return &resolver{
		scopes:                    newStack[map[string]bool](),
		localDeclDistancesByIdent: map[token.Token]int{},
	}
}

func (r *resolver) beginScope() func() {
	r.scopes.Push(map[string]bool{})
	return func() {
		r.scopes.Pop()
	}
}

func (r *resolver) declareIdent(ident string) {
	if r.scopes.Len() == 0 {
		return
	}
	r.scopes.Peek()[ident] = false
}

func (r *resolver) defineIdent(ident string) {
	if r.scopes.Len() == 0 {
		return
	}
	r.scopes.Peek()[ident] = true
}

func (r *resolver) resolveIdent(ident token.Token) {
	for i := r.scopes.Len() - 1; i >= 0; i-- {
		// TODO: Check if the identifier has been defined and if not, error?
		if _, ok := r.scopes.Index(i)[ident.Literal]; ok {
			r.localDeclDistancesByIdent[ident] = r.scopes.Len() - 1 - i
			return
		}
	}
	// If the identifier can't be found in any scope, then it must be a global variable
}

func (r *resolver) Resolve(program ast.Program) (m map[token.Token]int, err error) {
	defer func() {
		if r := recover(); r != nil {
			if resolverErr, ok := r.(*resolverError); ok {
				err = resolverErr
			} else {
				panic(r)
			}
		}
	}()
	r.resolveProgram(program)
	return r.localDeclDistancesByIdent, nil
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
	r.declareIdent(stmt.Name.Literal)
	if stmt.Initialiser != nil {
		r.resolveExpr(stmt.Initialiser)
	}
	// TODO: Move this into initialiser if block? The variable is only defined its got an initialiser
	r.defineIdent(stmt.Name.Literal)
}

func (r *resolver) resolveFunDecl(stmt ast.FunDecl) {
	r.defineIdent(stmt.Name.Literal)
	endScope := r.beginScope()
	defer endScope()
	for _, param := range stmt.Params {
		r.defineIdent(param.Literal)
	}
	for _, stmt := range stmt.Body {
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
	endScope := r.beginScope()
	defer endScope()
	for _, param := range expr.Params {
		r.defineIdent(param.Literal)
	}
	for _, stmt := range expr.Body {
		r.resolveStmt(stmt)
	}
}

func (r *resolver) resolveGroupExpr(expr ast.GroupExpr) {
	r.resolveExpr(expr.Expr)
}

func (r *resolver) resolveLiteralExpr(ast.LiteralExpr) {
	// Nothing to resolve
}

func (r *resolver) resolveVariableExpr(expr ast.VariableExpr) {
	// TODO: this is a bit awkward
	if r.scopes.Len() > 0 {
		if defined, ok := r.scopes.Peek()[expr.Name.Literal]; ok && !defined {
			panic(newTokenResolverErrorf(expr.Name, "variable definition cannot refer to itself"))
		}
	}
	r.resolveIdent(expr.Name)
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
	r.resolveIdent(expr.Left)
}

type resolverError struct {
	start token.Position
	end   token.Position
	msg   string
}

func newResolverErrorf(start token.Position, end token.Position, format string, args ...interface{}) *resolverError {
	return &resolverError{
		start: start,
		end:   end,
		msg:   fmt.Sprintf(format, args...),
	}
}

func newTokenResolverErrorf(tok token.Token, format string, args ...interface{}) *resolverError {
	return newResolverErrorf(tok.Start, tok.End, format, args...)
}

func (e *resolverError) Error() string {
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
