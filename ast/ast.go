// Package ast defines the types which are used to represent the abstract syntax tree of the Lox programming language.
package ast

import "github.com/marcuscaisey/golox/token"

// Node is the interface which all AST nodes implement.
type Node interface {
	Start() token.Position // Start returns the position of the first character of the node
	End() token.Position   // End returns the position of the character immediately after the node
}

// Program is the root node of the AST.
type Program struct {
	Stmts []Stmt `print:"repeat"`
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
	Stmts      []Stmt `print:"repeat"`
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

// IllegalStmt is an illegal statement, used as a placeholder when parsing fails.
type IllegalStmt struct {
	From, To token.Token
	stmt
}

func (i IllegalStmt) Start() token.Position { return i.From.Start }
func (i IllegalStmt) End() token.Position   { return i.To.End }

// Expr is the interface which all expression nodes implement.
type Expr interface {
	Node
	isExpr()
}

type expr struct{}

func (expr) isExpr() {}

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
	Value token.Token `print:"unnamed"`
	expr
}

func (l LiteralExpr) Start() token.Position { return l.Value.Start }
func (l LiteralExpr) End() token.Position   { return l.Value.End }

// VariableExpr is a variable expression, such as a or b.
type VariableExpr struct {
	Name token.Token `print:"named"`
	expr
}

func (v VariableExpr) Start() token.Position { return v.Name.Start }
func (v VariableExpr) End() token.Position   { return v.Name.End }

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

// IllegalExpr is an illegal expression, used as a placeholder when parsing fails.
type IllegalExpr struct {
	From, To token.Token
	expr
}

func (i IllegalExpr) Start() token.Position { return i.From.Start }
func (i IllegalExpr) End() token.Position   { return i.To.End }
