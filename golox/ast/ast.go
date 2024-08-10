// Package ast defines the types which are used to represent the abstract syntax tree of the Lox programming language.
package ast

import (
	"github.com/marcuscaisey/lox/golox/token"
)

// Node is the interface which all AST nodes implement.
type Node interface {
	Start() token.Position // Start returns the position of the first character of the node
	End() token.Position   // End returns the position of the character immediately after the node
}

// Program is the root node of the AST.
type Program struct {
	Stmts []Stmt `print:"unnamed"`
}

func (p Program) Start() token.Position { return p.Stmts[0].Start() }
func (p Program) End() token.Position   { return p.Stmts[len(p.Stmts)-1].End() }

// Stmt is the interface which all statement nodes implement.
type Stmt interface {
	Node
	isStmt()
}

type stmt struct{}

func (stmt) isStmt() {}

// VarDecl is a variable declaration, such as var a = 123 or var b.
type VarDecl struct {
	Var         token.Token
	Name        token.Token `print:"named"`
	Initialiser Expr        `print:"named"`
	Semicolon   token.Token
	stmt
}

func (d VarDecl) Start() token.Position { return d.Name.Start }
func (d VarDecl) End() token.Position   { return d.Semicolon.End }

// FunDecl is a function declaration, such as fun add(x, y) { return x + y; }.
type FunDecl struct {
	Fun        token.Token
	Name       token.Token   `print:"named"`
	Params     []token.Token `print:"named"`
	Body       []Stmt        `print:"named"`
	RightBrace token.Token
	stmt
}

func (d FunDecl) Start() token.Position { return d.Fun.Start }
func (d FunDecl) End() token.Position   { return d.RightBrace.End }

// ClassDecl is a class declaration, such as
//
//	class Foo {
//	  bar() {
//	    return "baz";
//	  }
//	}
type ClassDecl struct {
	Class           token.Token
	Name            token.Token       `print:"named"`
	InstanceMethods []MethodDecl      `print:"named"`
	ClassMethods    []ClassMethodDecl `print:"named"`
	RightBrace      token.Token
	stmt
}

func (c ClassDecl) Start() token.Position { return c.Class.Start }
func (c ClassDecl) End() token.Position   { return c.RightBrace.End }

// MethodDecl is a method declaration, such as
//
//	bar() {
//	  return "baz";
//	}
type MethodDecl struct {
	Name       token.Token   `print:"named"`
	Params     []token.Token `print:"named"`
	Body       []Stmt        `print:"named"`
	RightBrace token.Token
	stmt
}

func (m MethodDecl) Start() token.Position { return m.Name.Start }
func (m MethodDecl) End() token.Position   { return m.RightBrace.End }

// ClassMethodDecl is a class method declaration, such as
//
//	class bar() {
//	  return "baz";
//	}
type ClassMethodDecl struct {
	Class      token.Token
	Name       token.Token   `print:"named"`
	Params     []token.Token `print:"named"`
	Body       []Stmt        `print:"named"`
	RightBrace token.Token
	stmt
}

func (m ClassMethodDecl) Start() token.Position { return m.Name.Start }
func (m ClassMethodDecl) End() token.Position   { return m.RightBrace.End }

// ExprStmt is an expression statement, such as a function call.
type ExprStmt struct {
	Expr      Expr `print:"unnamed"`
	Semicolon token.Token
	stmt
}

func (s ExprStmt) Start() token.Position { return s.Expr.Start() }
func (s ExprStmt) End() token.Position   { return s.Semicolon.End }

// PrintStmt is a print statement, such as print "abc".
type PrintStmt struct {
	Print     token.Token
	Expr      Expr `print:"unnamed"`
	Semicolon token.Token
	stmt
}

func (p PrintStmt) Start() token.Position { return p.Print.Start }
func (p PrintStmt) End() token.Position   { return p.Semicolon.End }

// BlockStmt is a block statement, such as
//
//	{
//	    var a = 123;
//	    var b = 456;
//	}
type BlockStmt struct {
	LeftBrace  token.Token
	Stmts      []Stmt `print:"unnamed"`
	RightBrace token.Token
	stmt
}

func (b BlockStmt) Start() token.Position { return b.LeftBrace.Start }
func (b BlockStmt) End() token.Position   { return b.RightBrace.End }

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

func (i IfStmt) Start() token.Position { return i.If.Start }
func (i IfStmt) End() token.Position   { return i.Else.End() }

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

func (w WhileStmt) Start() token.Position { return w.While.Start }
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

func (f ForStmt) Start() token.Position { return f.For.Start }
func (f ForStmt) End() token.Position   { return f.Body.End() }

// IllegalStmt is an illegal statement, used as a placeholder when parsing fails.
type IllegalStmt struct {
	From, To token.Token
	stmt
}

func (i IllegalStmt) Start() token.Position { return i.From.Start }
func (i IllegalStmt) End() token.Position   { return i.To.End }

// BreakStmt is a break statement
type BreakStmt struct {
	Break     token.Token
	Semicolon token.Token
	stmt
}

func (b BreakStmt) Start() token.Position { return b.Break.Start }
func (b BreakStmt) End() token.Position   { return b.Semicolon.End }

// ContinueStmt is a continue statement
type ContinueStmt struct {
	Continue  token.Token
	Semicolon token.Token
	stmt
}

func (c ContinueStmt) Start() token.Position { return c.Continue.Start }
func (c ContinueStmt) End() token.Position   { return c.Semicolon.End }

// ReturnStmt is a return statement
type ReturnStmt struct {
	Return    token.Token
	Value     Expr `print:"unnamed"`
	Semicolon token.Token
	stmt
}

func (c ReturnStmt) Start() token.Position { return c.Return.Start }
func (c ReturnStmt) End() token.Position   { return c.Semicolon.End }

// Expr is the interface which all expression nodes implement.
type Expr interface {
	Node
	isExpr()
}

type expr struct{}

func (expr) isExpr() {}

// FunExpr is a function expression, such as fun(x, y) { return x + y; }.
type FunExpr struct {
	Fun        token.Token
	Params     []token.Token `print:"named"`
	Body       []Stmt        `print:"named"`
	RightBrace token.Token
	expr
}

func (d FunExpr) Start() token.Position { return d.Fun.Start }
func (d FunExpr) End() token.Position   { return d.RightBrace.End }

// GroupExpr is a group expression, such as (a + b).
type GroupExpr struct {
	LeftParen  token.Token
	Expr       Expr `print:"unnamed"`
	RightParen token.Token
	expr
}

func (g GroupExpr) Start() token.Position { return g.LeftParen.Start }
func (g GroupExpr) End() token.Position   { return g.RightParen.End }

// LiteralExpr is a literal expression, such as 123 or "abc".
type LiteralExpr struct {
	Value token.Token
	expr
}

func (l LiteralExpr) Start() token.Position { return l.Value.Start }
func (l LiteralExpr) End() token.Position   { return l.Value.End }

// VariableExpr is a variable expression, such as a or b.
type VariableExpr struct {
	Name token.Token
	expr
}

func (v VariableExpr) Start() token.Position { return v.Name.Start }
func (v VariableExpr) End() token.Position   { return v.Name.End }

// ThisExpr represents usage of the 'this' keyword.
type ThisExpr struct {
	This token.Token
	expr
}

func (t ThisExpr) Start() token.Position { return t.This.Start }
func (t ThisExpr) End() token.Position   { return t.This.End }

// CallExpr is a call expression, such as add(x, 1).
type CallExpr struct {
	Callee     Expr   `print:"named"`
	Args       []Expr `print:"named"`
	RightParen token.Token
	expr
}

func (c CallExpr) Start() token.Position { return c.Callee.Start() }
func (c CallExpr) End() token.Position   { return c.RightParen.End }

// GetExpr is a property access expression, such as a.b.
type GetExpr struct {
	Object Expr        `print:"named"`
	Name   token.Token `print:"named"`
	expr
}

func (g GetExpr) Start() token.Position { return g.Object.Start() }
func (g GetExpr) End() token.Position   { return g.Name.End }

// UnaryExpr is a unary operator expression, such as !a.
type UnaryExpr struct {
	Op    token.Token `print:"named"`
	Right Expr        `print:"named"`
	expr
}

func (u UnaryExpr) Start() token.Position { return u.Op.Start }
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

func (a AssignmentExpr) Start() token.Position { return a.Left.Start }
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
