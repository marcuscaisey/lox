package lsp

import (
	"cmp"
	"math"
	"slices"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/token"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

type completion struct {
	Label string
	Kind  protocol.CompletionItemKind
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

type identCompletions []*completionLocation

type completionLocation struct {
	Position    token.Position
	ScopeDepth  int
	Completions []*completion
}

// TODO: document
func (c identCompletions) At(pos *protocol.Position) []*completion {
	var completions []*completion

	startIdx, found := slices.BinarySearchFunc(c, pos, func(item *completionLocation, target *protocol.Position) int {
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
			completions = append(completions, completion.Completions...)
		}
	}

	return completions
}

func genIdentCompletions(program *ast.Program) identCompletions {
	icg := &identCompletionGenerator{}
	ast.Walk(program, icg.walk)
	return icg.completionLocations
}

type identCompletionGenerator struct {
	scopeDepth          int
	completionLocations []*completionLocation
}

func (c *identCompletionGenerator) walk(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.VarDecl:
		ast.Walk(node.Initialiser, c.walk)

		if !node.Name.IsValid() || node.Semicolon.IsZero() {
			return false
		}
		c.completionLocations = append(c.completionLocations, &completionLocation{
			Position:   node.Semicolon.End(),
			ScopeDepth: c.scopeDepth,
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
			Label: node.Name.Token.Lexeme,
			Kind:  protocol.CompletionItemKindFunction,
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
		c.scopeDepth++
		c.completionLocations = append(c.completionLocations, &completionLocation{
			Position:    node.Function.Body.LeftBrace.End(),
			ScopeDepth:  c.scopeDepth,
			Completions: localCompletions,
		})
		c.scopeDepth--

		ast.Walk(node.Function, c.walk)

		if node.Function.Body.RightBrace.IsZero() {
			return false
		}
		c.completionLocations = append(c.completionLocations, &completionLocation{
			Position:    node.Function.Body.RightBrace.End(),
			ScopeDepth:  c.scopeDepth,
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
		c.scopeDepth++
		c.completionLocations = append(c.completionLocations, &completionLocation{
			Position:    node.Function.Body.LeftBrace.End(),
			ScopeDepth:  c.scopeDepth,
			Completions: completions,
		})
		c.scopeDepth--

		ast.Walk(node.Function, c.walk)
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
		c.scopeDepth++
		c.completionLocations = append(c.completionLocations, &completionLocation{
			Position:    node.Function.Body.LeftBrace.End(),
			ScopeDepth:  c.scopeDepth,
			Completions: completions,
		})
		c.scopeDepth--

		ast.Walk(node.Function, c.walk)
		return false

	case *ast.ClassDecl:
		if !node.Name.IsValid() || node.Body == nil || node.Body.LeftBrace.IsZero() {
			return false
		}
		c.completionLocations = append(c.completionLocations, &completionLocation{
			Position:   node.Body.LeftBrace.End(),
			ScopeDepth: c.scopeDepth,
			Completions: []*completion{
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
