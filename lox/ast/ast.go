// Package ast declares the types used to represent abstract syntax trees for Lox programs.
package ast

import (
	"github.com/marcuscaisey/lox/lox/token"
)

// Node is the interface which all AST nodes implement.
//
//gosumtype:decl Node
type Node interface {
	token.CharacterRange
	isNode()
}

type node struct{}

func (node) isNode() {}

// Program is the root node of the AST.
type Program struct {
	Stmts Stmts `print:"unnamed"`
	node
}

func (p Program) Start() token.Position { return p.Stmts[0].Start() }
func (p Program) End() token.Position   { return p.Stmts[len(p.Stmts)-1].End() }

// Stmt is the interface which all statement nodes implement.
//
//gosumtype:decl Stmt
type Stmt interface {
	Node
	isStmt()
}

// Stmts is a slice of statements.
type Stmts []Stmt

func (s Stmts) Start() token.Position { return s[0].Start() }
func (s Stmts) End() token.Position   { return s[len(s)-1].End() }

type stmt struct {
	node
}

func (stmt) isStmt() {}

// CommentStmt is a comment on its own line, such as
//
//	// comment
type CommentStmt struct {
	Comment token.Token `print:"unnamed"`
	stmt
}

func (c CommentStmt) Start() token.Position { return c.Comment.StartPos }
func (c CommentStmt) End() token.Position   { return c.Comment.EndPos }

// InlineCommentStmt is a statement with a comment on the same line, such as
//
//	print 1; // comment
type InlineCommentStmt struct {
	Stmt    Stmt        `print:"unnamed"`
	Comment token.Token `print:"named"`
	stmt
}

func (s InlineCommentStmt) Start() token.Position { return s.Stmt.Start() }
func (s InlineCommentStmt) End() token.Position   { return s.Comment.EndPos }

// VarDecl is a variable declaration, such as var a = 123 or var b.
type VarDecl struct {
	Var         token.Token
	Name        token.Token `print:"named"`
	Initialiser Expr        `print:"named"`
	Semicolon   token.Token
	stmt
}

func (d VarDecl) Start() token.Position { return d.Name.StartPos }
func (d VarDecl) End() token.Position   { return d.Semicolon.EndPos }

// FunDecl is a function declaration, such as fun add(x, y) { return x + y; }.
type FunDecl struct {
	Fun      token.Token
	Name     token.Token `print:"named"`
	Function `print:"named"`
	stmt
}

func (d FunDecl) Start() token.Position { return d.Fun.StartPos }
func (d FunDecl) End() token.Position   { return d.Body.End() }

// Function is a function's parameters and body.
type Function struct {
	LeftParen token.Token
	Params    token.Tokens `print:"named"`
	Body      BlockStmt    `print:"named"`
}

func (f Function) Start() token.Position { return f.LeftParen.StartPos }
func (f Function) End() token.Position   { return f.Body.End() }

// ClassDecl is a class declaration, such as
//
//	class Foo {
//	  bar() {
//	    return "baz";
//	  }
//	}
type ClassDecl struct {
	Class      token.Token
	Name       token.Token `print:"named"`
	Body       Stmts       `print:"named"`
	RightBrace token.Token
	stmt
}

func (c ClassDecl) Start() token.Position { return c.Class.StartPos }
func (c ClassDecl) End() token.Position   { return c.RightBrace.EndPos }

// Methods returns the methods of the class.
func (c ClassDecl) Methods() []MethodDecl {
	methods := make([]MethodDecl, 0, len(c.Body))
	for _, stmt := range c.Body {
		if method, ok := stmt.(MethodDecl); ok {
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
	Modifiers []token.Token `print:"named"`
	Name      token.Token   `print:"named"`
	Function  `print:"named"`
	stmt
}

func (m MethodDecl) Start() token.Position {
	if len(m.Modifiers) > 0 {
		return m.Modifiers[0].StartPos
	}
	return m.Name.StartPos
}
func (m MethodDecl) End() token.Position { return m.Body.End() }

// HasModifier reports whether the declaration has a modifier of the target type.
func (m MethodDecl) HasModifier(target token.Type) bool {
	for _, modifier := range m.Modifiers {
		if modifier.Type == target {
			return true
		}
	}
	return false
}

// IsConstructor reports whether the declaration is a constructor.
func (m MethodDecl) IsConstructor() bool {
	return !m.HasModifier(token.Static) && m.Name.Lexeme == token.ConstructorIdent
}

// ExprStmt is an expression statement, such as a function call.
type ExprStmt struct {
	Expr      Expr `print:"unnamed"`
	Semicolon token.Token
	stmt
}

func (s ExprStmt) Start() token.Position { return s.Expr.Start() }
func (s ExprStmt) End() token.Position   { return s.Semicolon.EndPos }

// PrintStmt is a print statement, such as print "abc".
type PrintStmt struct {
	Print     token.Token
	Expr      Expr `print:"unnamed"`
	Semicolon token.Token
	stmt
}

func (p PrintStmt) Start() token.Position { return p.Print.StartPos }
func (p PrintStmt) End() token.Position   { return p.Semicolon.EndPos }

// BlockStmt is a block statement, such as
//
//	{
//	    var a = 123;
//	    var b = 456;
//	}
type BlockStmt struct {
	LeftBrace  token.Token
	Stmts      Stmts `print:"unnamed"`
	RightBrace token.Token
	stmt
}

func (b BlockStmt) Start() token.Position { return b.LeftBrace.StartPos }
func (b BlockStmt) End() token.Position   { return b.RightBrace.EndPos }

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

func (i IfStmt) Start() token.Position { return i.If.StartPos }
func (i IfStmt) End() token.Position {
	if i.Else != nil {
		return i.Else.End()
	} else {
		return i.Then.End()
	}
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

func (w WhileStmt) Start() token.Position { return w.While.StartPos }
func (w WhileStmt) End() token.Position   { return w.Body.End() }

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

func (f ForStmt) Start() token.Position { return f.For.StartPos }
func (f ForStmt) End() token.Position   { return f.Body.End() }

// IllegalStmt is an illegal statement, used as a placeholder when parsing fails.
type IllegalStmt struct {
	From, To token.Token
	stmt
}

func (i IllegalStmt) Start() token.Position { return i.From.StartPos }
func (i IllegalStmt) End() token.Position   { return i.To.EndPos }

// BreakStmt is a break statement
type BreakStmt struct {
	Break     token.Token
	Semicolon token.Token
	stmt
}

func (b BreakStmt) Start() token.Position { return b.Break.StartPos }
func (b BreakStmt) End() token.Position   { return b.Semicolon.EndPos }

// ContinueStmt is a continue statement
type ContinueStmt struct {
	Continue  token.Token
	Semicolon token.Token
	stmt
}

func (c ContinueStmt) Start() token.Position { return c.Continue.StartPos }
func (c ContinueStmt) End() token.Position   { return c.Semicolon.EndPos }

// ReturnStmt is a return statement
type ReturnStmt struct {
	Return    token.Token
	Value     Expr `print:"unnamed"`
	Semicolon token.Token
	stmt
}

func (c ReturnStmt) Start() token.Position { return c.Return.StartPos }
func (c ReturnStmt) End() token.Position   { return c.Semicolon.EndPos }

// Expr is the interface which all expression nodes implement.
//
//gosumtype:decl Expr
type Expr interface {
	Node
	isExpr()
}

// Exprs is a slice of expressions.
type Exprs []Expr

func (e Exprs) Start() token.Position { return e[0].Start() }
func (e Exprs) End() token.Position   { return e[len(e)-1].End() }

type expr struct {
	node
}

func (expr) isExpr() {}

// FunExpr is a function expression, such as fun(x, y) { return x + y; }.
type FunExpr struct {
	Fun      token.Token
	Function `print:"named"`
	expr
}

func (d FunExpr) Start() token.Position { return d.Fun.StartPos }
func (d FunExpr) End() token.Position   { return d.Body.End() }

// GroupExpr is a group expression, such as (a + b).
type GroupExpr struct {
	LeftParen  token.Token
	Expr       Expr `print:"unnamed"`
	RightParen token.Token
	expr
}

func (g GroupExpr) Start() token.Position { return g.LeftParen.StartPos }
func (g GroupExpr) End() token.Position   { return g.RightParen.EndPos }

// LiteralExpr is a literal expression, such as 123 or "abc".
type LiteralExpr struct {
	Value token.Token
	expr
}

func (l LiteralExpr) Start() token.Position { return l.Value.StartPos }
func (l LiteralExpr) End() token.Position   { return l.Value.EndPos }

// VariableExpr is a variable expression, such as a or b.
type VariableExpr struct {
	Name token.Token
	expr
}

func (v VariableExpr) Start() token.Position { return v.Name.StartPos }
func (v VariableExpr) End() token.Position   { return v.Name.EndPos }

// ThisExpr represents usage of the 'this' keyword.
type ThisExpr struct {
	This token.Token
	expr
}

func (t ThisExpr) Start() token.Position { return t.This.StartPos }
func (t ThisExpr) End() token.Position   { return t.This.EndPos }

// CallExpr is a call expression, such as add(x, 1).
type CallExpr struct {
	Callee     Expr  `print:"named"`
	Args       Exprs `print:"named"`
	RightParen token.Token
	expr
}

func (c CallExpr) Start() token.Position { return c.Callee.Start() }
func (c CallExpr) End() token.Position   { return c.RightParen.EndPos }

// GetExpr is a property access expression, such as a.b.
type GetExpr struct {
	Object Expr        `print:"named"`
	Name   token.Token `print:"named"`
	expr
}

func (g GetExpr) Start() token.Position { return g.Object.Start() }
func (g GetExpr) End() token.Position   { return g.Name.EndPos }

// UnaryExpr is a unary operator expression, such as !a.
type UnaryExpr struct {
	Op    token.Token `print:"named"`
	Right Expr        `print:"named"`
	expr
}

func (u UnaryExpr) Start() token.Position { return u.Op.StartPos }
func (u UnaryExpr) End() token.Position   { return u.Right.End() }

// BinaryExpr is a binary operator expression, such as a + b.
type BinaryExpr struct {
	Left  Expr        `print:"named"`
	Op    token.Token `print:"named"`
	Right Expr        `print:"named"`
	expr
}

func (b BinaryExpr) Start() token.Position { return b.Left.Start() }
func (b BinaryExpr) End() token.Position   { return b.Right.End() }

// TernaryExpr is a ternary operator expression, such as a ? b : c.
type TernaryExpr struct {
	Condition Expr `print:"named"`
	Then      Expr `print:"named"`
	Else      Expr `print:"named"`
	expr
}

func (t TernaryExpr) Start() token.Position { return t.Condition.Start() }
func (t TernaryExpr) End() token.Position   { return t.Else.End() }

// AssignmentExpr is an assignment expression, such as a = 2.
type AssignmentExpr struct {
	Left  token.Token `print:"named"`
	Right Expr        `print:"named"`
	expr
}

func (a AssignmentExpr) Start() token.Position { return a.Left.StartPos }
func (a AssignmentExpr) End() token.Position   { return a.Right.End() }

// SetExpr is a property assignment expression, such as a.b = 2.
type SetExpr struct {
	Object Expr        `print:"named"`
	Name   token.Token `print:"named"`
	Value  Expr        `print:"named"`
	expr
}

func (s SetExpr) Start() token.Position { return s.Object.Start() }
func (s SetExpr) End() token.Position   { return s.Value.End() }
