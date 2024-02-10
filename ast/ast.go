// Package ast defines the types which are used to represent the abstract syntax tree of the Lox programming language.
package ast

import "github.com/marcuscaisey/golox/token"

// Node is the interface which all AST nodes implement.
type Node interface {
	isNode()
}

type node struct{}

func (node) isNode() {}

// Program is the root node of the AST.
type Program struct {
	Stmts []Stmt
	node
}

// Statement nodes
type (
	// Stmt is the interface which all statement nodes implement.
	Stmt interface {
		Node
		isStmt()
	}

	// VarDecl is a variable declaration, such as var a = 123 or var b.
	VarDecl struct {
		Name        token.Token
		Initialiser Expr
		stmt
	}

	// ExprStmt is an expression statement, such as a function call.
	ExprStmt struct {
		Expr Expr
		stmt
	}

	// PrintStmt is a print statement, such as print "abc".
	PrintStmt struct {
		Expr Expr
		stmt
	}

	// IllegalStmt is an illegal statement, used as a placeholder when parsing fails.
	IllegalStmt struct {
		stmt
	}
)

type stmt struct {
	node
}

func (stmt) isStmt() {}

// Expression nodes
type (
	// Expr is the interface which all expression nodes implement.
	Expr interface {
		Node
		isExpr()
	}

	// GroupExpr is a group expression, such as (a + b).
	GroupExpr struct {
		Expr Expr
		expr
	}

	// LiteralExpr is a literal expression, such as 123 or "abc".
	LiteralExpr struct {
		Value any
		expr
	}

	// VariableExpr is a variable expression, such as a or b.
	VariableExpr struct {
		Name token.Token
		expr
	}

	// UnaryExpr is a unary operator expression, such as !a.
	UnaryExpr struct {
		Op    token.Token
		Right Expr
		expr
	}

	// BinaryExpr is a binary operator expression, such as a + b.
	BinaryExpr struct {
		Left  Expr
		Op    token.Token
		Right Expr
		expr
	}

	// TernaryExpr is a ternary operator expression, such as a ? b : c.
	TernaryExpr struct {
		Condition Expr
		Then      Expr
		Else      Expr
		expr
	}

	// AssignmentExpr is an assignment expression, such as a = 2.
	AssignmentExpr struct {
		Left  token.Token
		Right Expr
		expr
	}

	// IllegalExpr is an illegal expression, used as a placeholder when parsing fails.
	IllegalExpr struct {
		expr
	}
)

type expr struct {
	node
}

func (expr) isExpr() {}
