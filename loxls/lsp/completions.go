package lsp

import (
	"cmp"
	"fmt"
	"math"
	"slices"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/token"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

type completion struct {
	Position   token.Position
	ScopeDepth int
	Items      []*protocol.CompletionItem
}

type completions []*completion

func (c completions) At(pos *protocol.Position, log *logger) []*protocol.CompletionItem {
	var items []*protocol.CompletionItem

	startIdx, found := slices.BinarySearchFunc(c, pos, func(item *completion, target *protocol.Position) int {
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
	for i := startIdx; i >= 0; i-- {
		if c[i].ScopeDepth < curScopeDepth {
			curScopeDepth = c[i].ScopeDepth
		}
		if c[i].ScopeDepth == curScopeDepth {
			items = append(items, c[i].Items...)
		}
	}

	keywords := []string{"print", "var", "true", "false", "nil", "if", "while", "for", "fun", "class"}
	for _, keyword := range keywords {
		items = append(items, &protocol.CompletionItem{
			Label: keyword,
			Kind:  protocol.CompletionItemKindKeyword,
		})
	}

	padding := len(fmt.Sprint(len(items)))
	for i, item := range items {
		item.SortText = fmt.Sprintf("%0*d", padding, i)
	}

	return items
}

func genCompletions(program *ast.Program) completions {
	completions := &completionGenerator{}
	ast.Walk(program, completions.walk)
	return completions.completions
}

type completionGenerator struct {
	scopeDepth int

	completions completions
}

func (c *completionGenerator) walk(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.VarDecl:
		c.completions = append(c.completions, &completion{
			Position:   node.End(),
			ScopeDepth: c.scopeDepth,
			Items: []*protocol.CompletionItem{
				{
					Label: node.Name.Token.Lexeme,
					Kind:  protocol.CompletionItemKindVariable,
				},
			},
		})
		return true

	case *ast.FunDecl:
		nameItem := &protocol.CompletionItem{
			Label: node.Name.Token.Lexeme,
			Kind:  protocol.CompletionItemKindFunction,
		}

		localItems := make([]*protocol.CompletionItem, 1+len(node.Function.Params))
		localItems[len(localItems)-1] = nameItem
		for i, paramDecl := range node.Function.Params {
			localItems[i] = &protocol.CompletionItem{
				Label: paramDecl.Name.Token.Lexeme,
				Kind:  protocol.CompletionItemKindVariable,
			}
		}
		c.scopeDepth++
		c.completions = append(c.completions, &completion{
			Position:   node.Function.Body.LeftBrace.End(),
			ScopeDepth: c.scopeDepth,
			Items:      localItems,
		})
		c.scopeDepth--

		ast.Walk(node.Function, c.walk)

		c.completions = append(c.completions, &completion{
			Position:   node.End(),
			ScopeDepth: c.scopeDepth,
			Items:      []*protocol.CompletionItem{nameItem},
		})
		return false

	case *ast.FunExpr:
		items := make([]*protocol.CompletionItem, len(node.Function.Params))
		for i, paramDecl := range node.Function.Params {
			items[i] = &protocol.CompletionItem{
				Label: paramDecl.Name.Token.Lexeme,
				Kind:  protocol.CompletionItemKindVariable,
			}
		}
		c.scopeDepth++
		c.completions = append(c.completions, &completion{
			Position:   node.Function.Body.LeftBrace.End(),
			ScopeDepth: c.scopeDepth,
			Items:      items,
		})
		c.scopeDepth--

		ast.Walk(node.Function, c.walk)
		return false

	case *ast.MethodDecl:
		items := make([]*protocol.CompletionItem, len(node.Function.Params))
		for i, paramDecl := range node.Function.Params {
			items[i] = &protocol.CompletionItem{
				Label: paramDecl.Name.Token.Lexeme,
				Kind:  protocol.CompletionItemKindVariable,
			}
		}
		c.scopeDepth++
		c.completions = append(c.completions, &completion{
			Position:   node.Function.Body.LeftBrace.End(),
			ScopeDepth: c.scopeDepth,
			Items:      items,
		})
		c.scopeDepth--

		ast.Walk(node.Function, c.walk)
		return false

	case *ast.ClassDecl:
		c.completions = append(c.completions, &completion{
			Position:   node.Name.End(),
			ScopeDepth: c.scopeDepth,
			Items: []*protocol.CompletionItem{
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
