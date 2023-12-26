// Package ast defines the types which are used to represent the abstract syntax tree of the Lox programming language.
package ast

import "github.com/marcuscaisey/golox/token"

// Node is the interface which all AST nodes implement.
type Node interface {
	node()
}

// Expr is the interface which all expression nodes implement.
type Expr interface {
	Node
	exprNode()
}

// Expression nodes
type (
	// BinaryExpr is a binary operator expression, such as a + b.
	BinaryExpr struct {
		Left  Expr
		Op    token.Type
		Right Expr
	}

	// GroupExpr is a group expression, such as (a + b).
	GroupExpr struct {
		Expr Expr
	}

	// LiteralExpr is a literal expression, such as 123 or "abc".
	LiteralExpr struct {
		Value any
	}

	// TernaryExpr is a ternary operator expression, such as a ? b : c.
	TernaryExpr struct {
		Condition Expr
		Then      Expr
		Else      Expr
	}

	// UnaryExpr is a unary operator expression, such as !a.
	UnaryExpr struct {
		Op    token.Type
		Right Expr
	}
)

func (BinaryExpr) node()  {}
func (GroupExpr) node()   {}
func (LiteralExpr) node() {}
func (TernaryExpr) node() {}
func (UnaryExpr) node()   {}

func (BinaryExpr) exprNode()  {}
func (GroupExpr) exprNode()   {}
func (LiteralExpr) exprNode() {}
func (TernaryExpr) exprNode() {}
func (UnaryExpr) exprNode()   {}
