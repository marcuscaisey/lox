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
	case GroupExpr:
		return sexpr(n, d, sprint(n.Expr, d+1))
	case LiteralExpr:
		return fmt.Sprintf("%#v", n.Value)
	case UnaryExpr:
		return sexpr(n, d, fmt.Sprintf("%q", n.Op), sprint(n.Right, d+1))
	case BinaryExpr:
		return sexpr(n, d, sprint(n.Left, d+1), fmt.Sprintf("%q", n.Op), sprint(n.Right, d+1))
	case TernaryExpr:
		return sexpr(n, d, sprint(n.Condition, d+1), sprint(n.Then, d+1), sprint(n.Else, d+1))
	case nil:
		return "nil"
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
