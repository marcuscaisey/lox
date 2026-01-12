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

func commentsText(doc []*ast.Comment) string {
	lines := make([]string, len(doc))
	for i, comment := range doc {
		lines[i] = strings.TrimSpace(strings.TrimPrefix(comment.Comment.Lexeme, "//"))
	}
	return strings.Join(lines, "\n")
}

func varDetail(name *ast.Ident) string {
	return fmt.Sprintf("var %s", name)
}

func funDetail(decl *ast.FunDecl) string {
	if !decl.Name.IsValid() {
		return ""
	}
	return fmt.Sprintf("%s(%s)", funDetailPrefix(decl), formatParams(decl.GetParams()))
}

func funDetailPrefix(decl *ast.FunDecl) string {
	return fmt.Sprintf("fun %s", decl.Name)
}

func funSignature(params []*ast.ParamDecl) string {
	return fmt.Sprintf("fun(%s)", formatParams(params))
}

func classDetail(decl *ast.ClassDecl) string {
	if !decl.Name.IsValid() {
		return ""
	}
	return fmt.Sprintf("class %s", decl.Name)
}

func methodDetail(methodDecl *ast.MethodDecl, classDecl *ast.ClassDecl) string {
	return fmt.Sprintf("%s(%s)", methodDetailPrefix(methodDecl, classDecl), formatParams(methodDecl.GetParams()))
}

func methodDetailPrefix(methodDecl *ast.MethodDecl, classDecl *ast.ClassDecl) string {
	return fmt.Sprintf("(method) %s", formatMethodName(methodDecl, classDecl))
}

func formatMethodName(methodDecl *ast.MethodDecl, classDecl *ast.ClassDecl) string {
	return fmt.Sprintf("%s%s.%s", formatMethodModifiers(methodDecl.Modifiers), classDecl.Name, methodDecl.Name)
}

func formatMethodModifiers(modifiers []token.Token) string {
	b := &strings.Builder{}
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
