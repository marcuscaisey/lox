package ast

import (
	"fmt"
)

// Walk traverses an AST in depth-first order: It starts by calling f(node); node must not be nil. If f returns true,
// Walk invokes f recursively for each of the non-nil children of node.
func Walk(node Node, f func(Node) bool) {
	if !f(node) {
		return
	}
	switch node := node.(type) {
	case Program:
		walkSlice(node.Stmts, f)
	case CommentStmt:
	case InlineCommentStmt:
		Walk(node.Stmt, f)
	case VarDecl:
		if node.Initialiser != nil {
			Walk(node.Initialiser, f)
		}
	case FunDecl:
		walkSlice(node.Body.Stmts, f)
	case ClassDecl:
		walkSlice(node.Body, f)
	case MethodDecl:
		walkSlice(node.Body.Stmts, f)
	case ExprStmt:
		Walk(node.Expr, f)
	case PrintStmt:
		Walk(node.Expr, f)
	case BlockStmt:
		walkSlice(node.Stmts, f)
	case IfStmt:
		Walk(node.Condition, f)
		Walk(node.Then, f)
		if node.Else != nil {
			Walk(node.Else, f)
		}
	case WhileStmt:
		Walk(node.Condition, f)
		Walk(node.Body, f)
	case ForStmt:
		if node.Initialise != nil {
			Walk(node.Initialise, f)
		}
		if node.Condition != nil {
			Walk(node.Condition, f)
		}
		if node.Update != nil {
			Walk(node.Update, f)
		}
		Walk(node.Body, f)
	case IllegalStmt:
	case BreakStmt:
	case ContinueStmt:
	case ReturnStmt:
		if node.Value != nil {
			Walk(node.Value, f)
		}
	case FunExpr:
		walkSlice(node.Body.Stmts, f)
	case GroupExpr:
		Walk(node.Expr, f)
	case LiteralExpr:
	case VariableExpr:
	case ThisExpr:
	case CallExpr:
		Walk(node.Callee, f)
		walkSlice(node.Args, f)
	case GetExpr:
		Walk(node.Object, f)
	case UnaryExpr:
		Walk(node.Right, f)
	case BinaryExpr:
		Walk(node.Left, f)
		Walk(node.Right, f)
	case TernaryExpr:
		Walk(node.Condition, f)
		Walk(node.Then, f)
		Walk(node.Else, f)
	case AssignmentExpr:
		Walk(node.Right, f)
	case SetExpr:
		Walk(node.Object, f)
		Walk(node.Value, f)
	default:
		panic(fmt.Sprintf("unexpected node type: %T", node))
	}
}

func walkSlice[T Node](nodes []T, f func(Node) bool) {
	for _, node := range nodes {
		Walk(node, f)
	}
}
