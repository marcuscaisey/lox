// Package ast declares the types used to represent abstract syntax trees for Lox programs.
package ast

import (
	"fmt"
	"slices"
	"strings"

	"github.com/marcuscaisey/lox/golox/token"
)

// Node is the interface which all AST nodes implement.
//
//sumtype:decl
type Node interface {
	token.Range
	// IsValid reports whether the node represents a syntactically valid piece of code.
	// If a node is not valid, then it not guaranteed that calling any other methods on it will not panic.
	IsValid() bool
	isNode()
}

type node struct{}

func (node) isNode() {}

// Program is the root node of the AST.
type Program struct {
	StartPos token.Position
	Stmts    []Stmt `print:"unnamed"`
	EndPos   token.Position
	node
}

func (p *Program) Start() token.Position { return p.StartPos }
func (p *Program) End() token.Position   { return p.EndPos }
func (p *Program) IsValid() bool         { return p != nil && isValidSlice(p.Stmts) }

// Binding is the interface implemented by all nodes which may bind a name to a value in a namespace. Examples of
// namespaces are a lexical scope or an object's properties.
type Binding interface {
	Node
	// BoundIdent returns the identifier that the value is being bound to.
	BoundIdent() *Ident
}

// Ident is an identifier, such as a variable name.
type Ident struct {
	Token token.Token
	node
}

func (i *Ident) Start() token.Position { return i.Token.Start() }
func (i *Ident) End() token.Position   { return i.Token.End() }
func (i *Ident) IsValid() bool         { return i != nil && !i.Token.IsZero() }
func (i *Ident) String() string        { return i.Token.Lexeme }

// Format implements [fmt.Formatter]. All verbs have the default behaviour, except for 'm' (message) which formats the
// type for use in an error message.
func (i *Ident) Format(f fmt.State, verb rune) {
	switch verb {
	case 'm':
		fmt.Fprintf(f, "'%s'", i.String())
	default:
		fmt.Fprint(f, i.String())
	}
}

// Stmt is the interface which all statement nodes implement.
//
//sumtype:decl
type Stmt interface {
	Node
	isStmt()
}

type stmt struct {
	node
}

func (stmt) isStmt() {}

// IllegalStmt is used as a placeholder when parsing fails.
type IllegalStmt struct {
	From token.Token `print:"named"`
	To   token.Token `print:"named"`
	stmt
}

func (i *IllegalStmt) Start() token.Position { return i.From.Start() }
func (i *IllegalStmt) End() token.Position   { return i.To.End() }
func (i *IllegalStmt) IsValid() bool         { return false }

// Comment is a comment on its own line, such as
//
//	// comment
type Comment struct {
	Comment token.Token `print:"unnamed"`
	stmt
}

func (c *Comment) Start() token.Position { return c.Comment.Start() }
func (c *Comment) End() token.Position   { return c.Comment.End() }
func (c *Comment) IsValid() bool         { return c != nil && !c.Comment.IsZero() }

// CommentedStmt is a statement with a comment on the same line, such as
//
//	print 1; // *comment
type CommentedStmt struct {
	Stmt    Stmt     `print:"named"`
	Comment *Comment `print:"named"`
	stmt
}

func (i *CommentedStmt) Start() token.Position { return i.Stmt.Start() }
func (i *CommentedStmt) End() token.Position   { return last(i.Stmt, i.Comment).End() }
func (i *CommentedStmt) IsValid() bool {
	return i != nil && isValid(i.Stmt) && isValid(i.Comment)
}

// Documented is the interface implemented by all nodes which can be documented.
type Documented interface {
	Node
	// Documentation returns the node's documentation.
	Documentation() string
}

func docText(docComments []*Comment) string {
	lines := make([]string, len(docComments))
	for i, comment := range docComments {
		lines[i] = strings.TrimSpace(strings.TrimPrefix(comment.Comment.Lexeme, "//"))
	}
	return strings.Join(lines, "\n")
}

// Decl is the interface which all declaration nodes implement.
//
//sumtype:decl
type Decl interface {
	Stmt
	Binding
	isDecl()
}

type decl struct {
	stmt
}

func (decl) isDecl() {}

// VarDecl is a variable declaration, such as var a = 123 or var b.
type VarDecl struct {
	Var         token.Token
	Name        *Ident `print:"named"`
	Initialiser Expr   `print:"named"`
	Semicolon   token.Token
	decl
}

func (v *VarDecl) Start() token.Position { return v.Var.Start() }
func (v *VarDecl) End() token.Position   { return last(v.Var, v.Name, v.Initialiser, v.Semicolon).End() }
func (v *VarDecl) IsValid() bool {
	return v != nil && !v.Var.IsZero() && isValidOptional(v.Initialiser) && isValid(v.Name) && !v.Semicolon.IsZero()
}
func (v *VarDecl) BoundIdent() *Ident { return v.Name }

// FunDecl is a function declaration, such as fun add(x, y) { return x + y; }.
type FunDecl struct {
	DocComments []*Comment `print:"named"`
	Fun         token.Token
	Name        *Ident    `print:"named"`
	Function    *Function `print:"named"`
	decl
}

func (f *FunDecl) Start() token.Position { return f.Fun.Start() }
func (f *FunDecl) End() token.Position   { return last(f.Fun, f.Name, f.Function).End() }
func (f *FunDecl) IsValid() bool {
	return f != nil && isValidSlice(f.DocComments) && !f.Fun.IsZero() && isValid(f.Name) && isValid(f.Function)
}
func (f *FunDecl) BoundIdent() *Ident    { return f.Name }
func (f *FunDecl) Documentation() string { return docText(f.DocComments) }

// GetParams returns Function.Params or nil if Function is nil.
func (f *FunDecl) GetParams() []*ParamDecl {
	if f.Function == nil {
		return nil
	}
	return f.Function.Params
}

// Function is a function's parameters and body.
type Function struct {
	LeftParen token.Token
	Params    []*ParamDecl `print:"named"`
	Body      *Block       `print:"named"`
	node
}

func (f *Function) Start() token.Position { return f.LeftParen.Start() }
func (f *Function) End() token.Position   { return last(f.LeftParen, lastSlice(f.Params), f.Body).End() }
func (f *Function) IsValid() bool {
	return f != nil && !f.LeftParen.IsZero() && isValidSlice(f.Params) && isValid(f.Body)
}

// ParamDecl is a parameter declaration, such as x or y.
type ParamDecl struct {
	Name *Ident `print:"unnamed"`
	decl
}

func (p *ParamDecl) Start() token.Position { return p.Name.Start() }
func (p *ParamDecl) End() token.Position   { return p.Name.End() }
func (p *ParamDecl) IsValid() bool         { return p != nil && isValid(p.Name) }
func (p *ParamDecl) BoundIdent() *Ident    { return p.Name }

// ClassDecl is a class declaration, such as
//
//	class Foo {
//	  bar() {
//	    return "baz";
//	  }
//	}
type ClassDecl struct {
	DocComments []*Comment `print:"named"`
	Class       token.Token
	Name        *Ident `print:"named"`
	Superclass  *Ident `print:"named"`
	Body        *Block `print:"named"`
	decl
}

func (c *ClassDecl) Start() token.Position { return c.Class.Start() }
func (c *ClassDecl) End() token.Position   { return last(c.Class, c.Name, c.Body).End() }
func (c *ClassDecl) IsValid() bool {
	return c != nil && isValidSlice(c.DocComments) && !c.Class.IsZero() && isValid(c.Name) && isValid(c.Body)
}
func (c *ClassDecl) BoundIdent() *Ident    { return c.Name }
func (c *ClassDecl) Documentation() string { return docText(c.DocComments) }

// Methods returns the methods of the class.
func (c *ClassDecl) Methods() []*MethodDecl {
	if c.Body == nil {
		return nil
	}
	methods := make([]*MethodDecl, 0, len(c.Body.Stmts))
	for _, stmt := range c.Body.Stmts {
		if method, ok := stmt.(*MethodDecl); ok {
			methods = append(methods, method)
		}
	}
	return methods
}

// MethodDecl is a method declaration, such as
//
//	static bar() {
//	  return "baz";
//	}
type MethodDecl struct {
	Class       *ClassDecl
	DocComments []*Comment    `print:"named"`
	Modifiers   []token.Token `print:"named"`
	Name        *Ident        `print:"named"`
	Function    *Function     `print:"named"`
	decl
}

func (m *MethodDecl) Start() token.Position { return first(firstSlice(m.Modifiers), m.Name).Start() }
func (m *MethodDecl) End() token.Position {
	return last(lastSlice(m.Modifiers), m.Name, m.Function).End()
}
func (m *MethodDecl) IsValid() bool {
	return m != nil && isValidSlice(m.DocComments) && isValid(m.Name) && isValid(m.Function)
}
func (m *MethodDecl) BoundIdent() *Ident    { return m.Name }
func (m *MethodDecl) Documentation() string { return docText(m.DocComments) }

// GetParams returns Function.Params or nil if Function is nil.
func (m *MethodDecl) GetParams() []*ParamDecl {
	if m.Function == nil {
		return nil
	}
	return m.Function.Params
}

// hasModifier reports whether the declaration has a modifier with one of the target types.
func (m *MethodDecl) hasModifier(types ...token.Type) bool {
	if m == nil {
		return false
	}
	for _, modifier := range m.Modifiers {
		if slices.Contains(types, modifier.Type) {
			return true
		}
	}
	return false
}

// IsInit reports whether the declaration is an init method.
func (m *MethodDecl) IsInit() bool {
	return !m.IsStatic() && m.Name.IsValid() && m.Name.String() == token.IdentInit
}

// IsStatic reports whether the declaration is a static method.
func (m *MethodDecl) IsStatic() bool {
	return m.hasModifier(token.Static)
}

// IsGetter reports whether the declaration is a property getter.
func (m *MethodDecl) IsGetter() bool {
	return m.hasModifier(token.Get)
}

// IsSetter reports whether the declaration is a property setter.
func (m *MethodDecl) IsSetter() bool {
	return m.hasModifier(token.Set)
}

// IsAccessor reports whether the declaration is a property accessor.
func (m *MethodDecl) IsAccessor() bool {
	return m.IsGetter() || m.IsSetter()
}

// ExprStmt is an expression statement, such as a function call.
type ExprStmt struct {
	Expr      Expr `print:"unnamed"`
	Semicolon token.Token
	stmt
}

func (e *ExprStmt) Start() token.Position { return e.Expr.Start() }
func (e *ExprStmt) End() token.Position   { return last(e.Expr, e.Semicolon).End() }
func (e *ExprStmt) IsValid() bool {
	return e != nil && isValid(e.Expr) && !e.Semicolon.IsZero()
}

// PrintStmt is a print statement, such as print "abc".
type PrintStmt struct {
	Print     token.Token
	Expr      Expr `print:"unnamed"`
	Semicolon token.Token
	stmt
}

func (p *PrintStmt) Start() token.Position { return p.Print.Start() }
func (p *PrintStmt) End() token.Position   { return last(p.Print, p.Expr, p.Semicolon).End() }
func (p *PrintStmt) IsValid() bool {
	return p != nil && !p.Print.IsZero() && isValid(p.Expr) && !p.Semicolon.IsZero()
}

// Block is a block, such as
//
//	{
//	    var a = 123;
//	    var b = 456;
//	}
type Block struct {
	LeftBrace  token.Token
	Stmts      []Stmt `print:"unnamed"`
	RightBrace token.Token
	stmt
}

func (b *Block) Start() token.Position { return b.LeftBrace.Start() }
func (b *Block) End() token.Position {
	return last(b.LeftBrace, lastSlice(b.Stmts), b.RightBrace).End()
}
func (b *Block) IsValid() bool {
	return b != nil && !b.LeftBrace.IsZero() && isValidSlice(b.Stmts) && !b.RightBrace.IsZero()
}

// IfStmt is an if statement, such as
//
//	if (a == 123) {
//	    print "abc";
//	} else {
//
//	    print "def";
//	}
type IfStmt struct {
	If        token.Token
	Condition Expr `print:"named"`
	Then      Stmt `print:"named"`
	Else      Stmt `print:"named"`
	stmt
}

func (i *IfStmt) Start() token.Position { return i.If.Start() }
func (i *IfStmt) End() token.Position   { return last(i.If, i.Condition, i.Then, i.Else).End() }
func (i *IfStmt) IsValid() bool {
	return i != nil && !i.If.IsZero() && isValid(i.Condition) && isValid(i.Then) && isValidOptional(i.Else)
}

// WhileStmt is a while statement, such as
//
//	while (a < 10) {
//	    print a;
//	}
type WhileStmt struct {
	While     token.Token
	Condition Expr `print:"named"`
	Body      Stmt `print:"named"`
	stmt
}

func (w *WhileStmt) Start() token.Position { return w.While.Start() }
func (w *WhileStmt) End() token.Position   { return last(w.While, w.Condition, w.Body).End() }
func (w *WhileStmt) IsValid() bool {
	return w != nil && !w.While.IsZero() && isValid(w.Condition) && isValid(w.Body)
}

// ForStmt is a for statement, such as
//
//	for (var i = 0; i < 10; i = i + 1) {
//	    print i;
//	}
type ForStmt struct {
	For        token.Token
	Initialise Stmt `print:"named"`
	Condition  Expr `print:"named"`
	Update     Expr `print:"named"`
	Body       Stmt `print:"named"`
	stmt
}

func (f *ForStmt) Start() token.Position { return f.For.Start() }
func (f *ForStmt) End() token.Position {
	return last(f.For, f.Initialise, f.Condition, f.Update, f.Body).End()
}
func (f *ForStmt) IsValid() bool {
	return f != nil && isValidOptional(f.Initialise) && isValidOptional(f.Condition) && isValidOptional(f.Update) && isValid(f.Body)
}

// BreakStmt is a break statement
type BreakStmt struct {
	Break     token.Token
	Semicolon token.Token
	stmt
}

func (b *BreakStmt) Start() token.Position { return b.Break.Start() }
func (b *BreakStmt) End() token.Position   { return last(b.Break, b.Semicolon).End() }
func (b *BreakStmt) IsValid() bool         { return b != nil && !b.Break.IsZero() && !b.Semicolon.IsZero() }

// ContinueStmt is a continue statement
type ContinueStmt struct {
	Continue  token.Token
	Semicolon token.Token
	stmt
}

func (c *ContinueStmt) Start() token.Position { return c.Continue.Start() }
func (c *ContinueStmt) End() token.Position   { return last(c.Continue, c.Semicolon).End() }
func (c *ContinueStmt) IsValid() bool {
	return c != nil && !c.Continue.IsZero() && !c.Semicolon.IsZero()
}

// ReturnStmt is a return statement
type ReturnStmt struct {
	Return    token.Token
	Value     Expr `print:"unnamed"`
	Semicolon token.Token
	stmt
}

func (r *ReturnStmt) Start() token.Position { return r.Return.Start() }
func (r *ReturnStmt) End() token.Position   { return last(r.Return, r.Value, r.Semicolon).End() }
func (r *ReturnStmt) IsValid() bool {
	return r != nil && !r.Return.IsZero() && isValidOptional(r.Value) && !r.Semicolon.IsZero()
}

// Expr is the interface which all expression nodes implement.
//
//sumtype:decl
type Expr interface {
	Node
	isExpr()
}

type expr struct {
	node
}

func (expr) isExpr() {}

// LiteralExpr is a literal expression, such as 123 or "abc".
type LiteralExpr struct {
	Value token.Token
	expr
}

func (l *LiteralExpr) Start() token.Position { return l.Value.Start() }
func (l *LiteralExpr) End() token.Position   { return l.Value.End() }
func (l *LiteralExpr) IsValid() bool         { return l != nil && !l.Value.IsZero() }

// FunExpr is a function expression, such as fun(x, y) { return x + y; }.
type FunExpr struct {
	Fun      token.Token
	Function *Function `print:"unnamed"`
	expr
}

func (f *FunExpr) Start() token.Position { return f.Fun.Start() }
func (f *FunExpr) End() token.Position   { return last(f.Fun, f.Function).End() }
func (f *FunExpr) IsValid() bool         { return f != nil && !f.Fun.IsZero() && isValid(f.Function) }

// ListExpr is a list literal expression, such as [1, 2, 3].
type ListExpr struct {
	LeftBrack  token.Token
	Elements   []Expr `print:"unnamed"`
	RightBrack token.Token
	expr
}

func (l *ListExpr) Start() token.Position { return l.LeftBrack.Start() }
func (l *ListExpr) End() token.Position {
	return last(l.LeftBrack, lastSlice(l.Elements), l.RightBrack).End()
}
func (l *ListExpr) IsValid() bool {
	return l != nil && !l.LeftBrack.IsZero() && isValidSlice(l.Elements) && !l.RightBrack.IsZero()
}

// GetParams returns Function.Params or nil if Function is nil.
func (f *FunExpr) GetParams() []*ParamDecl {
	if f.Function == nil {
		return nil
	}
	return f.Function.Params
}

// IdentExpr is an identifier expression, such as a or b.
type IdentExpr struct {
	Ident *Ident
	expr
}

func (i *IdentExpr) Start() token.Position { return i.Ident.Start() }
func (i *IdentExpr) End() token.Position   { return i.Ident.End() }
func (i *IdentExpr) IsValid() bool         { return i != nil && isValid(i.Ident) }

// AssignmentExpr is an assignment expression, such as a = 2.
type AssignmentExpr struct {
	Left  *Ident `print:"named"`
	Right Expr   `print:"named"`
	expr
}

func (a *AssignmentExpr) Start() token.Position { return a.Left.Start() }
func (a *AssignmentExpr) End() token.Position   { return last(a.Left, a.Right).End() }
func (a *AssignmentExpr) IsValid() bool         { return a != nil && isValid(a.Left) && isValid(a.Right) }

// ThisExpr represents usage of the 'this' keyword.
type ThisExpr struct {
	This token.Token
	expr
}

func (t *ThisExpr) Start() token.Position { return t.This.Start() }
func (t *ThisExpr) End() token.Position   { return t.This.End() }
func (t *ThisExpr) IsValid() bool         { return t != nil && !t.This.IsZero() }

// SuperExpr is a super expression, such as super.b.
type SuperExpr struct {
	Super token.Token
	expr
}

func (s *SuperExpr) Start() token.Position { return s.Super.Start() }
func (s *SuperExpr) End() token.Position   { return s.Super.End() }
func (s *SuperExpr) IsValid() bool         { return s != nil && !s.Super.IsZero() }

// CallExpr is a call expression, such as add(x, 1).
type CallExpr struct {
	Callee     Expr `print:"named"`
	LeftParen  token.Token
	Args       []Expr `print:"named"`
	Commas     []token.Token
	RightParen token.Token
	expr
}

func (c *CallExpr) Start() token.Position { return c.Callee.Start() }
func (c *CallExpr) End() token.Position {
	return last(c.Callee, c.LeftParen, lastSlice(c.Args), lastSlice(c.Commas), c.RightParen).End()
}
func (c *CallExpr) IsValid() bool {
	return c != nil && isValid(c.Callee) && isValidSlice(c.Args) && !c.RightParen.IsZero()
}

// IndexExpr is an index expression, such as x[2].
type IndexExpr struct {
	Subject    Expr `print:"named"`
	LeftBrack  token.Token
	Index      Expr `print:"named"`
	RightBrack token.Token
	expr
}

func (i *IndexExpr) Start() token.Position { return i.Subject.Start() }
func (i *IndexExpr) End() token.Position {
	return last(i.Subject, i.LeftBrack, i.Index, i.RightBrack).End()
}
func (i *IndexExpr) IsValid() bool {
	return i != nil && i.Subject.IsValid() && !i.LeftBrack.IsZero() && i.Index.IsValid() && !i.RightBrack.IsZero()
}

// IndexSetExpr is an index assignment expression, such as x[2] = 3.
type IndexSetExpr struct {
	Subject    Expr `print:"named"`
	LeftBrack  token.Token
	Index      Expr `print:"named"`
	RightBrack token.Token
	Value      Expr `print:"named"`
	expr
}

func (i *IndexSetExpr) Start() token.Position { return i.Subject.Start() }
func (i *IndexSetExpr) End() token.Position {
	return last(i.Subject, i.LeftBrack, i.Index, i.RightBrack, i.Value).End()
}
func (i *IndexSetExpr) IsValid() bool {
	return i != nil && i.Subject.IsValid() && !i.LeftBrack.IsZero() && i.Index.IsValid() && !i.RightBrack.IsZero() && i.Value.IsValid()
}

// PropertyExpr is a property access expression, such as a.b.
type PropertyExpr struct {
	Object Expr `print:"named"`
	Dot    token.Token
	Name   *Ident `print:"named"`
	expr
}

func (p *PropertyExpr) Start() token.Position { return p.Object.Start() }
func (p *PropertyExpr) End() token.Position   { return last(p.Object, p.Dot, p.Name).End() }
func (p *PropertyExpr) IsValid() bool {
	return p != nil && isValid(p.Object) && !p.Dot.IsZero() && isValid(p.Name)
}

// PropertySetExpr is a property assignment expression, such as a.b = 2.
type PropertySetExpr struct {
	Object Expr   `print:"named"`
	Name   *Ident `print:"named"`
	Value  Expr   `print:"named"`
	expr
}

func (p *PropertySetExpr) Start() token.Position { return p.Object.Start() }
func (p *PropertySetExpr) End() token.Position   { return last(p.Object, p.Name, p.Value).End() }
func (p *PropertySetExpr) IsValid() bool {
	return p != nil && isValid(p.Object) && isValid(p.Name) && isValid(p.Value)
}
func (p *PropertySetExpr) BoundIdent() *Ident { return p.Name }

// UnaryExpr is a unary operator expression, such as !a.
type UnaryExpr struct {
	Op    token.Token `print:"named"`
	Right Expr        `print:"named"`
	expr
}

func (u *UnaryExpr) Start() token.Position { return u.Op.Start() }
func (u *UnaryExpr) End() token.Position   { return last(u.Op, u.Right).End() }
func (u *UnaryExpr) IsValid() bool         { return u != nil && !u.Op.IsZero() && isValid(u.Right) }

// BinaryExpr is a binary operator expression, such as a + b.
type BinaryExpr struct {
	Left  Expr        `print:"named"`
	Op    token.Token `print:"named"`
	Right Expr        `print:"named"`
	expr
}

func (b *BinaryExpr) Start() token.Position { return first(b.Left, b.Op, b.Right).Start() }
func (b *BinaryExpr) End() token.Position   { return last(b.Left, b.Op, b.Right).End() }
func (b *BinaryExpr) IsValid() bool {
	return b != nil && isValid(b.Left) && !b.Op.IsZero() && isValid(b.Right)
}

// TernaryExpr is a ternary operator expression, such as a ? b : c.
type TernaryExpr struct {
	Condition Expr `print:"named"`
	Then      Expr `print:"named"`
	Else      Expr `print:"named"`
	expr
}

func (t *TernaryExpr) Start() token.Position { return t.Condition.Start() }
func (t *TernaryExpr) End() token.Position   { return last(t.Condition, t.Then, t.Else).End() }
func (t *TernaryExpr) IsValid() bool {
	return t != nil && isValid(t.Condition) && isValid(t.Then) && isValid(t.Else)
}

// GroupExpr is a group expression, such as (a + b).
type GroupExpr struct {
	LeftParen  token.Token
	Expr       Expr `print:"unnamed"`
	RightParen token.Token
	expr
}

func (g *GroupExpr) Start() token.Position { return g.LeftParen.Start() }
func (g *GroupExpr) End() token.Position   { return last(g.LeftParen, g.Expr, g.RightParen).End() }
func (g *GroupExpr) IsValid() bool {
	return g != nil && !g.LeftParen.IsZero() && g.Expr != nil && isValid(g.Expr) && !g.RightParen.IsZero()
}

func first(ranges ...token.Range) token.Range {
	for _, rang := range ranges {
		switch rang := rang.(type) {
		case token.Token:
			if rang.IsZero() {
				continue
			}
		case Node:
			if isNil(rang) {
				continue
			}
		case nil:
			continue
		default:
			panic(fmt.Sprintf("unexpected range type: %T", rang))
		}
		return rang
	}
	return nil
}

func last(ranges ...token.Range) token.Range {
	ranges = slices.Clone(ranges)
	slices.Reverse(ranges)
	return first(ranges...)
}

func firstSlice[T token.Range](s []T) token.Range {
	sRangeSlice := make([]token.Range, len(s))
	for i, v := range s {
		sRangeSlice[i] = v
	}
	return first(sRangeSlice...)
}

func lastSlice[T token.Range](s []T) token.Range {
	s = slices.Clone(s)
	slices.Reverse(s)
	return firstSlice(s)
}

func isValid(n Node) bool {
	return !isNil(n) && n.IsValid()
}

func isValidOptional(n Node) bool {
	if !isNil(n) && !n.IsValid() {
		return false
	}
	return true
}

func isValidSlice[T Node](s []T) bool {
	for _, n := range s {
		if !n.IsValid() {
			return false
		}
	}
	return true
}

func isNil(node Node) bool {
	switch node := node.(type) {
	case *Program:
		return node == nil
	case *Ident:
		return node == nil
	case *IllegalStmt:
		return node == nil
	case *Comment:
		return node == nil
	case *CommentedStmt:
		return node == nil
	case *VarDecl:
		return node == nil
	case *FunDecl:
		return node == nil
	case *Function:
		return node == nil
	case *ParamDecl:
		return node == nil
	case *ClassDecl:
		return node == nil
	case *MethodDecl:
		return node == nil
	case *ExprStmt:
		return node == nil
	case *PrintStmt:
		return node == nil
	case *Block:
		return node == nil
	case *IfStmt:
		return node == nil
	case *WhileStmt:
		return node == nil
	case *ForStmt:
		return node == nil
	case *BreakStmt:
		return node == nil
	case *ContinueStmt:
		return node == nil
	case *ReturnStmt:
		return node == nil
	case *LiteralExpr:
		return node == nil
	case *FunExpr:
		return node == nil
	case *ListExpr:
		return node == nil
	case *IdentExpr:
		return node == nil
	case *AssignmentExpr:
		return node == nil
	case *ThisExpr:
		return node == nil
	case *SuperExpr:
		return node == nil
	case *CallExpr:
		return node == nil
	case *IndexExpr:
		return node == nil
	case *IndexSetExpr:
		return node == nil
	case *PropertyExpr:
		return node == nil
	case *PropertySetExpr:
		return node == nil
	case *UnaryExpr:
		return node == nil
	case *BinaryExpr:
		return node == nil
	case *TernaryExpr:
		return node == nil
	case *GroupExpr:
		return node == nil
	case nil:
		return true
	}
	return false
}
