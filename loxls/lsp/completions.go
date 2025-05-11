package lsp

import (
	"cmp"
	"math"
	"slices"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/token"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

type completionItem struct {
	Label string
	Kind  protocol.CompletionItemKind
}

var keywordCompletions = genKeywordCompletions()

func genKeywordCompletions() []*completionItem {
	keywords := []string{"print", "var", "true", "false", "nil", "if", "while", "for", "fun", "class"}
	items := make([]*completionItem, len(keywords))
	for i, keyword := range keywords {
		items[i] = &completionItem{
			Label: keyword,
			Kind:  protocol.CompletionItemKindKeyword,
		}
	}
	return items
}

type identCompletions []*identCompletion

type identCompletion struct {
	Position   token.Position
	ScopeDepth int
	Items      []*completionItem
}

// TODO: document
func (c identCompletions) At(pos *protocol.Position) []*completionItem {
	var items []*completionItem

	startIdx, found := slices.BinarySearchFunc(c, pos, func(item *identCompletion, target *protocol.Position) int {
		protocolPos := newPosition(item.Position)
		if protocolPos.Line == target.Line {
			return cmp.Compare(protocolPos.Character, target.Character)
		}
		return cmp.Compare(protocolPos.Line, target.Line)
	})
	if !found {
		startIdx--
	}

	curScopeDepth := math.MaxInt
	for _, completion := range slices.Backward(c[:startIdx+1]) {
		if completion.ScopeDepth < curScopeDepth {
			curScopeDepth = completion.ScopeDepth
		}
		if completion.ScopeDepth == curScopeDepth {
			items = append(items, completion.Items...)
		}
	}

	return items
}

func genIdentCompletions(program *ast.Program) identCompletions {
	icg := &identCompletionGenerator{}
	ast.Walk(program, icg.walk)
	return icg.completions
}

type identCompletionGenerator struct {
	scopeDepth  int
	completions identCompletions
}

func (c *identCompletionGenerator) walk(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.VarDecl:
		ast.Walk(node.Initialiser, c.walk)

		if !node.Name.IsValid() || node.Semicolon.IsZero() {
			return false
		}
		c.completions = append(c.completions, &identCompletion{
			Position:   node.Semicolon.End(),
			ScopeDepth: c.scopeDepth,
			Items: []*completionItem{
				{
					Label: node.Name.Token.Lexeme,
					Kind:  protocol.CompletionItemKindVariable,
				},
			},
		})
		return false

	case *ast.FunDecl:
		if !node.Name.IsValid() || node.Function == nil || node.Function.Body == nil || node.Function.Body.LeftBrace.IsZero() {
			return false
		}
		nameItem := &completionItem{
			Label: node.Name.Token.Lexeme,
			Kind:  protocol.CompletionItemKindFunction,
		}

		localItems := make([]*completionItem, 1+len(node.Function.Params))
		localItems[len(localItems)-1] = nameItem
		for i, paramDecl := range node.Function.Params {
			if !paramDecl.IsValid() {
				continue
			}
			localItems[i] = &completionItem{
				Label: paramDecl.Name.Token.Lexeme,
				Kind:  protocol.CompletionItemKindVariable,
			}
		}
		c.scopeDepth++
		c.completions = append(c.completions, &identCompletion{
			Position:   node.Function.Body.LeftBrace.End(),
			ScopeDepth: c.scopeDepth,
			Items:      localItems,
		})
		c.scopeDepth--

		ast.Walk(node.Function, c.walk)

		if node.Function.Body.RightBrace.IsZero() {
			return false
		}
		c.completions = append(c.completions, &identCompletion{
			Position:   node.Function.Body.RightBrace.End(),
			ScopeDepth: c.scopeDepth,
			Items:      []*completionItem{nameItem},
		})
		return false

	case *ast.FunExpr:
		if node.Function == nil || node.Function.Body == nil || node.Function.Body.LeftBrace.IsZero() {
			return false
		}
		items := make([]*completionItem, len(node.Function.Params))
		for i, paramDecl := range node.Function.Params {
			if !paramDecl.IsValid() {
				continue
			}
			items[i] = &completionItem{
				Label: paramDecl.Name.Token.Lexeme,
				Kind:  protocol.CompletionItemKindVariable,
			}
		}
		c.scopeDepth++
		c.completions = append(c.completions, &identCompletion{
			Position:   node.Function.Body.LeftBrace.End(),
			ScopeDepth: c.scopeDepth,
			Items:      items,
		})
		c.scopeDepth--

		ast.Walk(node.Function, c.walk)
		return false

	case *ast.MethodDecl:
		if !node.Name.IsValid() || node.Function == nil || node.Function.Body == nil || node.Function.Body.LeftBrace.IsZero() {
			return false
		}
		items := make([]*completionItem, len(node.Function.Params))
		for i, paramDecl := range node.Function.Params {
			if !paramDecl.IsValid() {
				continue
			}
			items[i] = &completionItem{
				Label: paramDecl.Name.Token.Lexeme,
				Kind:  protocol.CompletionItemKindVariable,
			}
		}
		c.scopeDepth++
		c.completions = append(c.completions, &identCompletion{
			Position:   node.Function.Body.LeftBrace.End(),
			ScopeDepth: c.scopeDepth,
			Items:      items,
		})
		c.scopeDepth--

		ast.Walk(node.Function, c.walk)
		return false

	case *ast.ClassDecl:
		if !node.Name.IsValid() || node.Body == nil || node.Body.LeftBrace.IsZero() {
			return false
		}
		c.completions = append(c.completions, &identCompletion{
			Position:   node.Body.LeftBrace.End(),
			ScopeDepth: c.scopeDepth,
			Items: []*completionItem{
				{
					Label: node.Name.Token.Lexeme,
					Kind:  protocol.CompletionItemKindClass,
				},
			},
		})
		return true

	case *ast.Block:
		c.scopeDepth++
		for _, stmt := range node.Stmts {
			ast.Walk(stmt, c.walk)
		}
		c.scopeDepth--
		return false

	default:
		return true
	}
}
