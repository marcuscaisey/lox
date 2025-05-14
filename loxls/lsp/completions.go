package lsp

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/token"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

type completion struct {
	Label   string
	Kind    protocol.CompletionItemKind
	Snippet string
}

var keywordCompletions = genKeywordCompletions()

func genKeywordCompletions() []*completion {
	keywords := []string{"print", "var", "true", "false", "nil", "if", "while", "for", "fun", "class"}
	completions := make([]*completion, len(keywords))
	for i, keyword := range keywords {
		completions[i] = &completion{
			Label: keyword,
			Kind:  protocol.CompletionItemKindKeyword,
		}
	}
	return completions
}

type identCompletions struct {
	completionLocations []*completionLocation
	scopeLocations      []*scopeLocation
}

type completionLocation struct {
	Position    token.Position
	ScopeDepth  int
	Completions []*completion
}

type scopeLocation struct {
	Position token.Position
	Depth    int
}

// TODO: document
func (c *identCompletions) At(pos *protocol.Position) []*completion {
	positionCmp := func(x, y *protocol.Position) int {
		if x.Line == y.Line {
			return cmp.Compare(x.Character, y.Character)
		}
		return cmp.Compare(x.Line, y.Line)
	}

	completionStartIdx, found := slices.BinarySearchFunc(c.completionLocations, pos, func(loc *completionLocation, target *protocol.Position) int {
		return positionCmp(newPosition(loc.Position), target)
	})
	if !found {
		completionStartIdx--
	}

	curScopeIdx, found := slices.BinarySearchFunc(c.scopeLocations, pos, func(loc *scopeLocation, target *protocol.Position) int {
		pos := newPosition(loc.Position)
		return positionCmp(pos, target)
	})
	if !found {
		curScopeIdx--
	}

	curScopeDepth := 0
	if curScopeIdx >= 0 {
		// TODO: Inject a scope location with depth 0 at 0:0 so that we don't have to deal with the curScopeIdx being
		// negative anymore. This would simplify the curScopeIdx logic below as well.
		curScopeDepth = c.scopeLocations[curScopeIdx].Depth
	}

	var completions []*completion
	for _, completionLoc := range slices.Backward(c.completionLocations[:completionStartIdx+1]) {
		for curScopeIdx >= 0 && c.scopeLocations[curScopeIdx].Position.Compare(completionLoc.Position) >= 1 {
			curScopeIdx--
			if curScopeIdx >= 0 {
				curScopeDepth = min(curScopeDepth, c.scopeLocations[curScopeIdx].Depth)
			} else {
				curScopeDepth = 0
			}
		}
		if completionLoc.ScopeDepth == curScopeDepth {
			completions = append(completions, completionLoc.Completions...)
		}
	}
	return completions
}

func genIdentCompletions(program *ast.Program) *identCompletions {
	g := &identCompletionsGenerator{}
	ast.Walk(program, g.walk)
	return &identCompletions{scopeLocations: g.scopeLocations, completionLocations: g.completionLocations}
}

type identCompletionsGenerator struct {
	scopeDepth          int
	completionLocations []*completionLocation
	scopeLocations      []*scopeLocation
}

func (g *identCompletionsGenerator) walk(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.VarDecl:
		ast.Walk(node.Initialiser, g.walk)

		if !node.Name.IsValid() || node.Semicolon.IsZero() {
			return false
		}
		g.completionLocations = append(g.completionLocations, &completionLocation{
			Position:   node.Semicolon.End(),
			ScopeDepth: g.scopeDepth,
			Completions: []*completion{
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
		nameCompletion := &completion{
			Label:   node.Name.Token.Lexeme,
			Kind:    protocol.CompletionItemKindFunction,
			Snippet: fmt.Sprintf("%s($1)$0", node.Name.Token.Lexeme),
		}

		localCompletions := make([]*completion, 1+len(node.Function.Params))
		localCompletions[len(localCompletions)-1] = nameCompletion
		for i, paramDecl := range node.Function.Params {
			if !paramDecl.IsValid() {
				continue
			}
			localCompletions[i] = &completion{
				Label: paramDecl.Name.Token.Lexeme,
				Kind:  protocol.CompletionItemKindVariable,
			}
		}
		g.scopeDepth++
		g.completionLocations = append(g.completionLocations, &completionLocation{
			Position:    node.Function.Body.LeftBrace.End(),
			ScopeDepth:  g.scopeDepth,
			Completions: localCompletions,
		})
		g.scopeDepth--

		ast.Walk(node.Function, g.walk)

		if node.Function.Body.RightBrace.IsZero() {
			return false
		}
		g.completionLocations = append(g.completionLocations, &completionLocation{
			Position:    node.Function.Body.RightBrace.End(),
			ScopeDepth:  g.scopeDepth,
			Completions: []*completion{nameCompletion},
		})
		return false

	case *ast.FunExpr:
		if node.Function == nil || node.Function.Body == nil || node.Function.Body.LeftBrace.IsZero() {
			return false
		}
		completions := make([]*completion, len(node.Function.Params))
		for i, paramDecl := range node.Function.Params {
			if !paramDecl.IsValid() {
				continue
			}
			completions[i] = &completion{
				Label: paramDecl.Name.Token.Lexeme,
				Kind:  protocol.CompletionItemKindVariable,
			}
		}
		g.scopeDepth++
		g.completionLocations = append(g.completionLocations, &completionLocation{
			Position:    node.Function.Body.LeftBrace.End(),
			ScopeDepth:  g.scopeDepth,
			Completions: completions,
		})
		g.scopeDepth--

		ast.Walk(node.Function, g.walk)
		return false

	case *ast.MethodDecl:
		if !node.Name.IsValid() || node.Function == nil || node.Function.Body == nil || node.Function.Body.LeftBrace.IsZero() {
			return false
		}
		completions := make([]*completion, len(node.Function.Params))
		for i, paramDecl := range node.Function.Params {
			if !paramDecl.IsValid() {
				continue
			}
			completions[i] = &completion{
				Label: paramDecl.Name.Token.Lexeme,
				Kind:  protocol.CompletionItemKindVariable,
			}
		}
		g.scopeDepth++
		g.completionLocations = append(g.completionLocations, &completionLocation{
			Position:    node.Function.Body.LeftBrace.End(),
			ScopeDepth:  g.scopeDepth,
			Completions: completions,
		})
		g.scopeDepth--

		ast.Walk(node.Function, g.walk)
		return false

	case *ast.ClassDecl:
		if !node.Name.IsValid() || node.Body == nil || node.Body.LeftBrace.IsZero() {
			return false
		}
		g.completionLocations = append(g.completionLocations, &completionLocation{
			Position:   node.Body.LeftBrace.End(),
			ScopeDepth: g.scopeDepth,
			Completions: []*completion{
				{
					Label:   node.Name.Token.Lexeme,
					Kind:    protocol.CompletionItemKindClass,
					Snippet: fmt.Sprintf("%s($1)$0", node.Name.Token.Lexeme),
				},
			},
		})
		return true

	case *ast.Block:
		g.scopeDepth++
		if !node.LeftBrace.IsZero() {
			g.scopeLocations = append(g.scopeLocations, &scopeLocation{
				Position: node.LeftBrace.End(),
				Depth:    g.scopeDepth,
			})
		}

		for _, stmt := range node.Stmts {
			ast.Walk(stmt, g.walk)
		}

		g.scopeDepth--
		if !node.RightBrace.IsZero() {
			g.scopeLocations = append(g.scopeLocations, &scopeLocation{
				Position: node.RightBrace.End(),
				Depth:    g.scopeDepth,
			})
		}

		return false

	default:
		return true
	}
}
