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
	compls := make([]*completion, len(keywords))
	for i, keyword := range keywords {
		compls[i] = &completion{
			Label: keyword,
			Kind:  protocol.CompletionItemKindKeyword,
		}
	}
	return compls
}

type identCompletions struct {
	complLocs []*completionLocation
	scopeLocs []*scopeLocation
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

	complLocsStartIdx, found := slices.BinarySearchFunc(c.complLocs, pos, func(loc *completionLocation, target *protocol.Position) int {
		return positionCmp(newPosition(loc.Position), target)
	})
	if !found {
		complLocsStartIdx--
	}

	scopeLocsIdx, found := slices.BinarySearchFunc(c.scopeLocs, pos, func(loc *scopeLocation, target *protocol.Position) int {
		pos := newPosition(loc.Position)
		return positionCmp(pos, target)
	})
	if !found {
		scopeLocsIdx--
	}

	var compls []*completion
	curScopeDepth := c.scopeLocs[scopeLocsIdx].Depth
	seenLabels := map[string]bool{}
	for _, complLoc := range slices.Backward(c.complLocs[:complLocsStartIdx+1]) {
		for ; curScopeDepth > 0 && c.scopeLocs[scopeLocsIdx].Position.Compare(complLoc.Position) >= 1; scopeLocsIdx-- {
			curScopeDepth = min(curScopeDepth, c.scopeLocs[scopeLocsIdx-1].Depth)
		}
		if complLoc.ScopeDepth <= curScopeDepth {
			for _, compl := range complLoc.Completions {
				if !seenLabels[compl.Label] {
					compls = append(compls, compl)
					seenLabels[compl.Label] = true
				}
			}
		}
	}
	return compls
}

func genIdentCompletions(program *ast.Program) *identCompletions {
	g := newIdentCompletionsGenerator(program.Start())
	return g.Generate(program)
}

type identCompletionsGenerator struct {
	scopeDepth                    int
	globalComplLocs               []*completionLocation
	curClassCompl                 *completion
	curClassForwardDeclaredCompls []*completion

	complLocs []*completionLocation
	scopeLocs []*scopeLocation
}

func newIdentCompletionsGenerator(programStart token.Position) *identCompletionsGenerator {
	return &identCompletionsGenerator{
		scopeLocs: []*scopeLocation{{
			Position: programStart,
			Depth:    0,
		}},
	}
}

func (g *identCompletionsGenerator) Generate(program *ast.Program) *identCompletions {
	g.globalComplLocs = g.readGlobalCompletionLocations(program)
	ast.Walk(program, g.walk)
	return &identCompletions{complLocs: g.complLocs, scopeLocs: g.scopeLocs}
}

func (g *identCompletionsGenerator) readGlobalCompletionLocations(program *ast.Program) []*completionLocation {
	var locs []*completionLocation
	for _, stmt := range program.Stmts {
		if commentedStmt, ok := stmt.(*ast.CommentedStmt); ok {
			stmt = commentedStmt.Stmt
		}
		decl, ok := stmt.(ast.Decl)
		if !ok {
			continue
		}
		ident := decl.Ident()
		if !ident.IsValid() {
			continue
		}

		var compl *completion
		switch decl.(type) {
		case *ast.VarDecl:
			compl = varCompl(ident.Token.Lexeme)
		case *ast.FunDecl:
			compl = funCompl(ident.Token.Lexeme)
		case *ast.ClassDecl:
			compl = classCompl(ident.Token.Lexeme)
		case *ast.MethodDecl, *ast.ParamDecl:
			panic(fmt.Sprintf("unexpected declaration type: %T", stmt))
		}
		locs = append(locs, &completionLocation{
			Position:    decl.Start(),
			Completions: []*completion{compl},
		})
	}
	return locs
}

func (g *identCompletionsGenerator) walk(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.VarDecl:
		ast.Walk(node.Initialiser, g.walk)

		if !node.Name.IsValid() || node.Semicolon.IsZero() {
			return false
		}
		g.complLocs = append(g.complLocs, &completionLocation{
			Position:    node.Semicolon.End(),
			ScopeDepth:  g.scopeDepth,
			Completions: []*completion{varCompl(node.Name.Token.Lexeme)},
		})
		return false

	case *ast.FunDecl:
		if !node.Name.IsValid() || node.Function == nil || node.Function.Body == nil || node.Function.Body.LeftBrace.IsZero() {
			return false
		}

		funCompl := funCompl(node.Name.Token.Lexeme)

		var forwardDeclaredCompls []*completion
		if g.scopeDepth == 0 {
			forwardDeclaredCompls = g.globalCompletionsAfter(node.Start())
		}
		bodyCompls := make([]*completion, len(node.Function.Params)+1+len(forwardDeclaredCompls))
		for i, paramDecl := range node.Function.Params {
			// FIXME: if the param declaration is not valid, then a nil completion will be returned
			if !paramDecl.IsValid() {
				continue
			}
			bodyCompls[i] = &completion{
				Label: paramDecl.Name.Token.Lexeme,
				Kind:  protocol.CompletionItemKindVariable,
			}
		}
		bodyCompls[len(node.Function.Params)] = funCompl
		copy(bodyCompls[len(node.Function.Params)+1:], forwardDeclaredCompls)
		g.complLocs = append(g.complLocs, &completionLocation{
			Position:    node.Function.Body.LeftBrace.End(),
			ScopeDepth:  g.scopeDepth + 1,
			Completions: bodyCompls,
		})

		ast.Walk(node.Function, g.walk)

		if node.Function.Body.RightBrace.IsZero() {
			return false
		}
		g.complLocs = append(g.complLocs, &completionLocation{
			Position:    node.Function.Body.RightBrace.End(),
			ScopeDepth:  g.scopeDepth,
			Completions: []*completion{funCompl},
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
		g.complLocs = append(g.complLocs, &completionLocation{
			Position:    node.Function.Body.LeftBrace.End(),
			ScopeDepth:  g.scopeDepth + 1,
			Completions: completions,
		})

		ast.Walk(node.Function, g.walk)
		return false

	case *ast.ClassDecl:
		if !node.Name.IsValid() || node.Body == nil {
			return false
		}

		classCompl := classCompl(node.Name.Token.Lexeme)

		g.curClassCompl = classCompl
		if g.scopeDepth == 0 {
			g.curClassForwardDeclaredCompls = g.globalCompletionsAfter(node.Start())
		}

		ast.Walk(node.Body, g.walk)

		if node.Body.RightBrace.IsZero() {
			return false
		}
		g.complLocs = append(g.complLocs, &completionLocation{
			Position:    node.Body.RightBrace.End(),
			ScopeDepth:  g.scopeDepth,
			Completions: []*completion{classCompl},
		})

		return false

	case *ast.MethodDecl:
		if !node.Name.IsValid() || node.Function == nil || node.Function.Body == nil || node.Function.Body.LeftBrace.IsZero() {
			return false
		}
		compls := make([]*completion, len(node.Function.Params)+1+len(g.curClassForwardDeclaredCompls))
		for i, paramDecl := range node.Function.Params {
			if !paramDecl.IsValid() {
				continue
			}
			compls[i] = &completion{
				Label: paramDecl.Name.Token.Lexeme,
				Kind:  protocol.CompletionItemKindVariable,
			}
		}
		compls[len(node.Function.Params)] = g.curClassCompl
		copy(compls[len(node.Function.Params)+1:], g.curClassForwardDeclaredCompls)
		g.complLocs = append(g.complLocs, &completionLocation{
			Position:    node.Function.Body.LeftBrace.End(),
			ScopeDepth:  g.scopeDepth + 1,
			Completions: compls,
		})

		ast.Walk(node.Function, g.walk)
		return false

	case *ast.Block:
		g.scopeDepth++
		if !node.LeftBrace.IsZero() {
			g.scopeLocs = append(g.scopeLocs, &scopeLocation{
				Position: node.LeftBrace.End(),
				Depth:    g.scopeDepth,
			})
		}

		for _, stmt := range node.Stmts {
			ast.Walk(stmt, g.walk)
		}

		g.scopeDepth--
		if !node.RightBrace.IsZero() {
			g.scopeLocs = append(g.scopeLocs, &scopeLocation{
				Position: node.RightBrace.End(),
				Depth:    g.scopeDepth,
			})
		}

		return false

	default:
		return true
	}
}

func varCompl(name string) *completion {
	return &completion{
		Label: name,
		Kind:  protocol.CompletionItemKindVariable,
	}
}

func funCompl(name string) *completion {
	return &completion{
		Label:   name,
		Kind:    protocol.CompletionItemKindFunction,
		Snippet: callSnippet(name),
	}
}

func classCompl(name string) *completion {
	return &completion{
		Label:   name,
		Kind:    protocol.CompletionItemKindClass,
		Snippet: callSnippet(name),
	}
}

func callSnippet(name string) string {
	return fmt.Sprintf("%s($1)$0", name)
}

func (g *identCompletionsGenerator) globalCompletionsAfter(pos token.Position) []*completion {
	startIdx, found := slices.BinarySearchFunc(g.globalComplLocs, pos, func(loc *completionLocation, target token.Position) int {
		return loc.Position.Compare(target)
	})
	if found {
		startIdx++
	}
	compls := make([]*completion, len(g.globalComplLocs)-startIdx)
	for i, loc := range g.globalComplLocs[startIdx:] {
		for _, compl := range loc.Completions {
			compls[i] = compl
		}
	}
	return compls
}
