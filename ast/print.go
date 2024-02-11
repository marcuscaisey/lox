package ast

import (
	"fmt"
	"reflect"
	"strings"
)

// Print prints an AST Node to stdout as an indented s-expression.
func Print(n Node) {
	fmt.Println(Sprint(n))
}

// Sprint formats an AST Node as an indented s-expression.
func Sprint(n Node) string {
	return sprint(n, 0)
}

func sprint(n Node, d int) string {
	switch n := n.(type) {
	case Program:
		stmts := make([]string, len(n.Stmts))
		for i, stmt := range n.Stmts {
			stmts[i] = sprint(stmt, d+1)
		}
		return sexpr(n, d, stmts...)
	case VarDecl:
		if n.Initialiser == nil {
			return sexpr(n, d, fmt.Sprintf("%q", n.Name))
		} else {
			return sexpr(n, d, fmt.Sprintf("%q", n.Name), sprint(n.Initialiser, d+1))
		}
	case ExprStmt:
		return sexpr(n, d, sprint(n.Expr, d+1))
	case PrintStmt:
		return sexpr(n, d, sprint(n.Expr, d+1))
	case IllegalStmt:
		return sexpr(n, d)
	case GroupExpr:
		return sexpr(n, d, sprint(n.Expr, d+1))
	case LiteralExpr:
		return fmt.Sprint(n.Value)
	case VariableExpr:
		return fmt.Sprintf("%q", n.Name)
	case UnaryExpr:
		return sexpr(n, d, fmt.Sprintf("%q", n.Op), sprint(n.Right, d+1))
	case BinaryExpr:
		return sexpr(n, d, sprint(n.Left, d+1), fmt.Sprintf("%q", n.Op), sprint(n.Right, d+1))
	case TernaryExpr:
		return sexpr(n, d, sprint(n.Condition, d+1), sprint(n.Then, d+1), sprint(n.Else, d+1))
	case AssignmentExpr:
		return sexpr(n, d, fmt.Sprintf("%q", n.Left), sprint(n.Right, d+1))
	case IllegalExpr:
		return sexpr(n, d)
	default:
		panic(fmt.Sprintf("unexpected node type: %T", n))
	}
}

func sexpr(n Node, d int, children ...string) string {
	var b strings.Builder
	fmt.Fprint(&b, "(", reflect.TypeOf(n).Name())
	for _, child := range children {
		fmt.Fprint(&b, "\n", strings.Repeat("  ", d+1), child)
	}
	fmt.Fprint(&b, ")")
	return b.String()
}
