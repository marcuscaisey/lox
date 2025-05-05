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
	printTags := parsePrintTags(nodeType)
	for i := range nodeType.NumField() {
		field := nodeType.Field(i)
		value := nodeValue.Field(i)

		tag, ok := printTags[field.Name]
		if !ok {
			continue
		}

		if field.Type.Kind() == reflect.Slice {
			prefix := ""
			extraDepth := 0
			if tag == "named" {
				children = append(children, fmt.Sprintf("(%s [", field.Name))
				if value.Len() == 0 {
					children[len(children)-1] += "])"
					continue
				}
				prefix = "  "
				extraDepth = 1
			}
			for j := range value.Len() {
				child, ok := formatValue(value.Index(j), depth+1+extraDepth)
				if !ok {
					panic(fmt.Sprintf("%s field %s element %d has unsupported type: %T", nodeType.Name(), field.Name, j, value.Index(j).Interface()))
				}
				children = append(children, prefix+child)
			}
			if tag == "named" {
				children = append(children, "]")
			}
			continue
		}

		formattedValue, ok := formatValue(value, depth+1)
		if !ok {
			panic(fmt.Sprintf("%s field %s has unsupported type: %T", nodeType.Name(), field.Name, value.Interface()))
		}

		if tag == "unnamed" {
			return fmt.Sprintf("(%s %s)", nodeType.Name(), formattedValue)
		}

		children = append(children, fmt.Sprintf("(%s %s)", field.Name, formattedValue))
	}

	return sexpr(nodeType.Name(), depth, children...)
}

func formatValue(value reflect.Value, depth int) (string, bool) {
	if (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) && value.IsNil() {
		return "nil", true
	}
	var child string
	switch value := value.Interface().(type) {
	case token.Token:
		child = value.Lexeme
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

func parsePrintTags(nodeType reflect.Type) map[string]string {
	printTags := map[string]string{}
	unnamedTagCount := 0
	for i := range nodeType.NumField() {
		field := nodeType.Field(i)
		tag, ok := parsePrintTag(nodeType.Name(), field)
		if !ok {
			continue
		}
		printTags[field.Name] = tag
		if tag == "unnamed" {
			unnamedTagCount++
		}
	}
	if len(printTags) > 1 && unnamedTagCount > 0 {
		panic(fmt.Sprintf(`%s has %d field(s) with an "unnamed" print tag and %d field(s) with a "named" print tag (%s). Only one field can have a print tag if the "unnamed" tag is used.`, nodeType.Name(), unnamedTagCount, len(printTags)-unnamedTagCount, printTags))
	}
	return printTags
}

func parsePrintTag(structName string, field reflect.StructField) (tag string, ok bool) {
	tags := strings.Split(field.Tag.Get("print"), ",")
	if len(tags) == 1 && tags[0] == "" {
		return "", false
	}

	for _, tag := range tags {
		switch tag {
		case "named", "unnamed":
			if slices.Contains(tags, "unnamed") && slices.Contains(tags, "named") {
				panic(fmt.Sprintf(`%s field %s has both "named" and "unnamed" print tags`, structName, field.Name))
			}
			return tag, true
		default:
			panic(fmt.Sprintf("%s field %s has invalid print tag: %q", structName, field.Name, tag))
		}
	}

	return "", false
}
