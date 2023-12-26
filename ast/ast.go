// Package ast defines the types which are used to represent the abstract syntax tree of the Lox programming language.
package ast

import "github.com/marcuscaisey/golox/token"

// Node is the interface which all AST nodes implement.
type Node interface {
	node()
}

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
)

func (GroupExpr) node()   {}
func (LiteralExpr) node() {}
func (UnaryExpr) node()   {}
func (BinaryExpr) node()  {}
func (TernaryExpr) node() {}

func (GroupExpr) exprNode()   {}
func (LiteralExpr) exprNode() {}
func (UnaryExpr) exprNode()   {}
func (BinaryExpr) exprNode()  {}
func (TernaryExpr) exprNode() {}
