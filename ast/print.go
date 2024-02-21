package ast

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/marcuscaisey/golox/token"
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
	case LiteralExpr:
		return n.Value.String()
	case VariableExpr:
		return n.Name.String()
	}

	nType := reflect.TypeOf(n)
	nValue := reflect.ValueOf(n)

	var children []string
	for i := 0; i < nType.NumField(); i++ {
		field := nType.Field(i)
		value := nValue.Field(i)
		tag, ok := field.Tag.Lookup("print")
		if !ok {
			continue
		}

		if tag == "repeat" {
			if value.Kind() != reflect.Slice {
				panic(fmt.Sprintf("%s field %s has repeat tag but is not a slice", nType.Name(), field.Name))
			}
			for j := 0; j < value.Len(); j++ {
				element := value.Index(j).Interface().(Node)
				child := sprint(element, d+1)
				children = append(children, child)
			}
			continue
		}

		prefix := ""
		switch tag {
		case "named":
			prefix = field.Name + ": "
		case "unnamed":
		default:
			panic(fmt.Sprintf("%s field %s has invalid print tag: %q", nType.Name(), field.Name, tag))
		}

		var child string
		switch value := value.Interface().(type) {
		case Node:
			child = sprint(value, d+1)
		case nil:
			continue
		case token.Token:
			child = value.String()
		default:
			panic(fmt.Sprintf("%s field %s has unsupported type: %T", nType.Name(), field.Name, value))
		}
		children = append(children, prefix+child)
	}
	return sexpr(nType.Name(), d, children...)
}

func sexpr(name string, d int, children ...string) string {
	var b strings.Builder
	fmt.Fprint(&b, "(", name)
	for _, child := range children {
		fmt.Fprint(&b, "\n", strings.Repeat("  ", d+1), child)
	}
	fmt.Fprint(&b, ")")
	return b.String()
}
