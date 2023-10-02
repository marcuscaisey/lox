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
	// BinaryExpr is a binary operation, such as + or *.
	BinaryExpr struct {
		Left  Expr
		Op    token.Type
		Right Expr
	}

	// GroupExpr is a parenthesised expression, such as (1 + 2).
	GroupExpr struct {
		Expr Expr
	}

	// LiteralExpr is a literal value, such as a number or string.
	LiteralExpr struct {
		Value any
	}

	// UnaryExpr is a unary operator expression, such as ! or -.
	UnaryExpr struct {
		Op    token.Type
		Right Expr
	}
)

func (BinaryExpr) node() {}
func (GroupExpr) node()    {}
func (LiteralExpr) node()  {}
func (UnaryExpr) node()  {}

func (BinaryExpr) exprNode() {}
func (GroupExpr) exprNode()    {}
func (LiteralExpr) exprNode()  {}
func (UnaryExpr) exprNode()  {}
