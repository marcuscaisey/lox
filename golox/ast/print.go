package ast

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/marcuscaisey/lox/golox/token"
)

// Print prints an AST Node to stdout as an indented s-expression.
func Print(node Node) {
	fmt.Println(Sprint(node))
}

// Sprint formats an AST Node as an indented s-expression.
func Sprint(node Node) string {
	return sprint(node, 0)
}

func sprint(node Node, depth int) string {
	switch node := node.(type) {
	case *LiteralExpr:
		return node.Value.Lexeme
	case *IdentExpr:
		return node.Ident.Token.Lexeme
	default:
	}

	nodeType := reflect.TypeOf(node)
	nodeValue := reflect.ValueOf(node)
	if nodeType.Kind() == reflect.Pointer {
		nodeType = nodeType.Elem()
		nodeValue = nodeValue.Elem()
	}

	var children []string
	for i := range nodeType.NumField() {
		field := nodeType.Field(i)
		value := nodeValue.Field(i)

		named, ok := parsePrintTag(nodeType.Name(), field)
		if !ok {
			continue
		}

		if field.Type.Kind() == reflect.Slice {
			prefix := ""
			extraDepth := 0
			if named {
				children = append(children, field.Name+": [")
				if value.Len() == 0 {
					children[len(children)-1] += "]"
					continue
				}
				prefix = "  "
				extraDepth = 1
			}
			for j := range value.Len() {
				child, ok := childString(value.Index(j), depth+1+extraDepth)
				if !ok {
					panic(fmt.Sprintf("%s field %s element %d has unsupported type: %T", nodeType.Name(), field.Name, j, value.Index(j).Interface()))
				}
				children = append(children, prefix+child)
			}
			if named {
				children = append(children, "]")
			}
			continue
		}

		prefix := ""
		if named {
			prefix = field.Name + ": "
		}

		child, ok := childString(value, depth+1)
		if !ok {
			panic(fmt.Sprintf("%s field %s has unsupported type: %T", nodeType.Name(), field.Name, value.Interface()))
		}
		children = append(children, prefix+child)
	}

	return sexpr(nodeType.Name(), depth, children...)
}

func childString(value reflect.Value, depth int) (string, bool) {
	if (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) && value.IsNil() {
		return "nil", true
	}
	var child string
	switch value := value.Interface().(type) {
	case token.Token:
		if value.Type == token.EOF {
			child = "EOF"
		} else {
			child = value.Lexeme
		}
	case *Ident:
		child = value.Token.Lexeme
	case Node:
		child = sprint(value, depth)
	case bool:
		child = fmt.Sprint(value)
	default:
		return "", false
	}
	return child, true
}

func sexpr(name string, depth int, children ...string) string {
	var b strings.Builder
	fmt.Fprint(&b, "(", name)
	for _, child := range children {
		fmt.Fprint(&b, "\n", strings.Repeat("  ", depth+1), child)
	}
	fmt.Fprint(&b, ")")
	return b.String()
}

func parsePrintTag(structName string, field reflect.StructField) (named bool, ok bool) {
	tags := strings.Split(field.Tag.Get("print"), ",")
	if len(tags) == 1 && tags[0] == "" {
		return false, false
	}

	for _, tag := range tags {
		switch tag {
		case "named", "unnamed":
			if slices.Contains(tags, "unnamed") && slices.Contains(tags, "named") {
				panic(fmt.Sprintf(`%s field %s has both "named" and "unnamed" print tags`, structName, field.Name))
			}
			return tag == "named", true
		default:
			panic(fmt.Sprintf("%s field %s has invalid print tag: %q", structName, field.Name, tag))
		}
	}

	return false, false
}
