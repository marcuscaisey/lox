// Package ast defines the types which are used to represent the abstract syntax tree of the Lox programming language.
package ast

import "github.com/marcuscaisey/golox/token"

// Node is the interface which all AST nodes implement.
type Node interface {
	node()
}

// Program is the root node of the AST.
type Program struct {
	Stmts []Stmt
}

func (Program) node() {}

// Statement nodes
type (
	// Stmt is the interface which all statement nodes implement.
	Stmt interface {
		Node
		stmtNode()
	}

	// ExprStmt is an expression statement, such as a function call.
	ExprStmt struct {
		Expr Expr
	}

	// PrintStmt is a print statement, such as print "abc".
	PrintStmt struct {
		Expr Expr
	}

	// IllegalStmt is an illegal statement, used as a placeholder when parsing fails.
	IllegalStmt struct{}
)

func (ExprStmt) node()    {}
func (PrintStmt) node()   {}
func (IllegalStmt) node() {}

func (ExprStmt) stmtNode()    {}
func (PrintStmt) stmtNode()   {}
func (IllegalStmt) stmtNode() {}

// Expression nodes
type (
	// Expr is the interface which all expression nodes implement.
	Expr interface {
		Node
		exprNode()
	}

	// GroupExpr is a group expression, such as (a + b).
	GroupExpr struct {
		Expr Expr
	}

	// LiteralExpr is a literal expression, such as 123 or "abc".
	LiteralExpr struct {
		Value any
	}

	// UnaryExpr is a unary operator expression, such as !a.
	UnaryExpr struct {
		Op    token.Token
		Right Expr
	}

	// BinaryExpr is a binary operator expression, such as a + b.
	BinaryExpr struct {
		Left  Expr
		Op    token.Token
		Right Expr
	}

	// TernaryExpr is a ternary operator expression, such as a ? b : c.
	TernaryExpr struct {
		Condition Expr
		Then      Expr
		Else      Expr
	}

	// IllegalExpr is an illegal expression, used as a placeholder when parsing fails.
	IllegalExpr struct{}
)

func (GroupExpr) node()   {}
func (LiteralExpr) node() {}
func (UnaryExpr) node()   {}
func (BinaryExpr) node()  {}
func (TernaryExpr) node() {}
func (IllegalExpr) node() {}

func (GroupExpr) exprNode()   {}
func (LiteralExpr) exprNode() {}
func (UnaryExpr) exprNode()   {}
func (BinaryExpr) exprNode()  {}
func (TernaryExpr) exprNode() {}
func (IllegalExpr) exprNode() {}
