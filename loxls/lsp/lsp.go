// Package lsp implements a JSON-RPC 2.0 server handler which implements the Language Server Protocol.
package lsp

import (
	"fmt"
	"strings"
	"unicode/utf16"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/token"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

var log *logger

func varDetail(name *ast.Ident) (string, bool) {
	if !name.IsValid() {
		return "", false
	}
	return fmt.Sprintf("var %s", name), true
}

func funDetail(decl *ast.FunDecl) (string, bool) {
	prefix, ok := funDetailPrefix(decl)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%s(%s)", prefix, formatParams(decl.GetParams())), true
}

func funDetailPrefix(decl *ast.FunDecl) (string, bool) {
	if !decl.Name.IsValid() {
		return "", false
	}
	return fmt.Sprintf("fun %s", decl.Name), true
}

func funSignature(params []*ast.ParamDecl) string {
	return fmt.Sprintf("fun(%s)", formatParams(params))
}

func classDetail(decl *ast.ClassDecl) (string, bool) {
	if !decl.Name.IsValid() {
		return "", false
	}
	return fmt.Sprintf("class %s", decl.Name), true
}

func methodDetail(methodDecl *ast.MethodDecl) (string, bool) {
	if methodDecl.IsSetter() {
		return "", false
	}
	if methodDecl.IsGetter() {
		static := ""
		if methodDecl.IsStatic() {
			static = "static "
		}
		return fmt.Sprintf("(property) %s%s.%s", static, methodDecl.Class.Name, methodDecl.Name), true
	}
	prefix, ok := methodDetailPrefix(methodDecl)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%s(%s)", prefix, formatParams(methodDecl.GetParams())), true
}

func methodDetailPrefix(methodDecl *ast.MethodDecl) (string, bool) {
	name, ok := formatMethodName(methodDecl)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("(method) %s", name), true
}

func formatMethodName(decl *ast.MethodDecl) (string, bool) {
	if !decl.Name.IsValid() || decl.Class == nil || !decl.Class.Name.IsValid() {
		return "", false
	}
	return fmt.Sprintf("%s%s.%s", formatMethodModifiers(decl.Modifiers), decl.Class.Name, decl.Name), true
}

func formatMethodModifiers(modifiers []token.Token) string {
	b := new(strings.Builder)
	for _, modifier := range modifiers {
		fmt.Fprintf(b, "%s ", modifier.Lexeme)
	}
	return b.String()
}

func formatParams(params []*ast.ParamDecl) string {
	b := &strings.Builder{}
	for i, param := range params {
		fmt.Fprint(b, param.Name.String())
		if i < len(params)-1 {
			fmt.Fprint(b, ", ")
		}
	}
	return b.String()
}

// containingIdentRange returns the range of the identifier containing the given position and whether one exists.
func containingIdentRange(program *ast.Program, pos *protocol.Position) (*protocol.Range, bool) {
	file := program.Start().File
	line := []rune(string(file.Line(pos.Line + 1)))
	posIdx := len(utf16.Decode(utf16.Encode(line)[:pos.Character]))

	startIdx := posIdx
startIdxLoop:
	for startIdx > 0 {
		switch {
		case isAlpha(line[startIdx-1]):
			startIdx--
		// Identifiers can't start with a digit so if the previous character is a digit, we need to find an alphabetic
		// character which proceeds it before we can accept the digit.
		case isDigit(line[startIdx-1]):
			for i := startIdx - 2; i >= 0 && isAlphaNumeric(line[i]); i-- {
				if isAlpha(line[i]) {
					startIdx = i
					continue startIdxLoop
				}
			}
			break startIdxLoop
		default:
			break startIdxLoop
		}
	}
	startChar := utf16RunesLen(line[:startIdx])

	if startChar == pos.Character {
		return nil, false
	}

	endIdx := posIdx
	for endIdx < len(line) && isAlphaNumeric(line[endIdx]) {
		endIdx++
	}
	endChar := utf16RunesLen(line[:endIdx])

	return &protocol.Range{
		Start: &protocol.Position{Line: pos.Line, Character: startChar},
		End:   &protocol.Position{Line: pos.Line, Character: endChar},
	}, true
}

func isDigit(r rune) bool {
	return '0' <= r && r <= '9'
}

func isAlpha(r rune) bool {
	return ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || r == '_'
}

func isAlphaNumeric(r rune) bool {
	return isAlpha(r) || isDigit(r)
}

// outermostNodeAt returns the outermost node of a [*ast.Program] which has type T and contains a [*protocol.Position].
func outermostNodeAt[T ast.Node](program *ast.Program, pos *protocol.Position) (T, bool) {
	return ast.Find(program, func(node T) bool {
		return inRange(pos, node)
	})
}

// outermostNodeAtOrBefore returns the outermost node of a [*ast.Program] which has type T and contains or precedes a
// [*protocol.Position].
func outermostNodeAtOrBefore[T ast.Node](node ast.Node, pos *protocol.Position) (T, bool) {
	return ast.Find(node, func(node T) bool {
		return inRangeOrFollows(pos, node)
	})
}

// innermostNodeAt returns the innermost node of a [*ast.Program] which has type T and contains a [*protocol.Position].
func innermostNodeAt[T ast.Node](node ast.Node, pos *protocol.Position) (T, bool) {
	return ast.FindLast(node, func(node T) bool {
		return inRange(pos, node)
	})
}

func utf16RunesLen(s []rune) int {
	return len(utf16.Encode(s))
}

func utf16StringLen(s string) int {
	return utf16RunesLen([]rune(s))
}

func utf16BytesLen(b []byte) int {
	return utf16StringLen(string(b))
}
