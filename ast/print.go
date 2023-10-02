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
	case BinaryExpr:
		return sexpr(n, d, sprint(n.Left, d+1), fmt.Sprintf("%q", n.Op), sprint(n.Right, d+1))
	case LiteralExpr:
		return fmt.Sprintf("%#v", n.Value)
	case GroupExpr:
		return sexpr(n, d, sprint(n.Expr, d+1))
	case UnaryExpr:
		return sexpr(n, d, fmt.Sprintf("%q", n.Op), sprint(n.Right, d+1))
	case nil:
		return "<nil>"
	default:
		panic(fmt.Sprintf("ast: cannot print node of type %T", n))
	}
}

func sexpr(n Node, d int, children ...string) string {
	var sb strings.Builder
	sb.WriteString("(")
	sb.WriteString(reflect.TypeOf(n).Name())
	for _, child := range children {
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("  ", d+1))
		sb.WriteString(child)
	}
	sb.WriteString(")")
	return sb.String()
}
