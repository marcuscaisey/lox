package lsp

import (
	"fmt"
	"slices"
	"unicode"
	"unicode/utf16"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/token"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

var (
	expressionKeywords       = []string{"true", "false", "nil"}
	statementKeywordSnippets = []keywordSnippet{
		{"print", "print $0;"},
		{"var", "var $0;"},
		{"if", "if ($1) {\n    $0\n}"},
		{"while", "while ($1) {\n    $0\n}"},
		{"for", "for ($1;$2;$3) {\n    $0\n}"},
		{"fun", "fun $1($2) {\n    $0\n}"},
		{"class", "class $1 {\n    $0\n}"},
	}
)

// completion represents a candidate piece of text that can be suggested to complete text that is being typed.
type completion struct {
	// Text is the text that is being suggested. It will be shown in the completion menu and is the text that will be
	// inserted if Snippet is empty or the client doesn't support snippets.
	Text string
	// Kind is the "kind" value that should be used for the resulting CompletionItem.
	Kind protocol.CompletionItemKind
	// Snippet is the snippet that should be inserted instead of Text if the client supports snippets.
	Snippet string
}

// keywordCompletor provides completions of keywords.
type keywordCompletor struct {
	program *ast.Program
}

// newKeywordCompletor returns a [keywordCompletor] which completes keywords inside the given program.
func newKeywordCompletor(program *ast.Program) *keywordCompletor {
	return &keywordCompletor{program: program}
}

type keywordSnippet struct {
	Keyword string
	Snippet string
}

// Complete returns completions for keywords which are valid at the given position.
func (c *keywordCompletor) Complete(pos *protocol.Position) []*completion {
	compls := make([]*completion, len(expressionKeywords))
	for i, keyword := range expressionKeywords {
		compls[i] = &completion{
			Text: keyword,
			Kind: protocol.CompletionItemKindKeyword,
		}
	}

	if c.validStatementPosition(pos) {
		for _, keywordSnippet := range statementKeywordSnippets {
			compls = append(compls, &completion{
				Text:    keywordSnippet.Keyword,
				Kind:    protocol.CompletionItemKindKeyword,
				Snippet: keywordSnippet.Snippet,
			})
		}
	}

	return compls
}

// validStatementPosition reports whether it's valid to suggest a statement at the given position. This is when either:
//  1. Only whitespace precedes it.
//  2. It's immediately preceded by a valid statement.
//  3. It's immediately preceded by the opening of a block.
//
// If the position is contained by an identifier, then the above conditions are applied to the start position of the
// identifier.
func (c *keywordCompletor) validStatementPosition(pos *protocol.Position) bool {
	startPos := pos
	if containingIdentRange, ok := containingIdentRange(c.program, pos); ok {
		startPos = containingIdentRange.Start
	}

	prevCharEnd, ok := c.previousCharacterEnd(startPos)
	if !ok {
		return true
	}

	result := false
	ast.Walk(c.program, func(n ast.Node) bool {
		switch n.(type) {
		case ast.Stmt:
			if block, ok := n.(*ast.Block); ok && !block.LeftBrace.IsZero() && equalPositions(prevCharEnd, block.LeftBrace.End()) {
				result = true
				return false
			}
			if n.IsValid() && equalPositions(prevCharEnd, n.End()) {
				result = true
				return false
			}
			return true
		default:
			return true
		}
	})

	return result
}

// previousCharacterEnd returns the end position of the previous non-whitespace character and whether one exists.
func (c *keywordCompletor) previousCharacterEnd(pos *protocol.Position) (*protocol.Position, bool) {
	lastCharEnd := func(line []rune) (int, bool) {
		for i, rune := range slices.Backward(line) {
			if !unicode.IsSpace(rune) {
				return len(utf16.Encode(line[:i+1])), true
			}
		}
		return 0, false
	}

	file := c.program.Start().File

	curLineRunes := []rune(string(file.Line(pos.Line + 1)))
	curLineUTF16 := utf16.Encode(curLineRunes)
	runesBeforeCurChar := utf16.Decode(curLineUTF16[:pos.Character])
	if character, ok := lastCharEnd(runesBeforeCurChar); ok {
		return &protocol.Position{Line: pos.Line, Character: character}, true
	}

	line := pos.Line - 1
	for ; line >= 0; line-- {
		lineRunes := []rune(string(file.Line(line + 1)))
		if character, ok := lastCharEnd(lineRunes); ok {
			return &protocol.Position{Line: line, Character: character}, true
		}
	}

	return nil, false
}

// identCompletor provides completions of identifiers based on their lexical scope.
type identCompletor struct {
	globalScope *completionScope
}

// newIdentCompletor returns an [identCompletor] which completes identifiers inside the given program.
func newIdentCompletor(program *ast.Program) *identCompletor {
	g := newIdentCompletionGenerator(program.Start(), program.End())
	globalScope := g.Generate(program)
	return &identCompletor{globalScope: globalScope}
}

// Complete returns completions for all identifiers in scope at the given position.
func (c *identCompletor) Complete(pos *protocol.Position) []*completion {
	return c.globalScope.Complete(pos)
}

// completionScope represents a lexical scope.
type completionScope struct {
	start     token.Position        // Position of the first character of the scope.
	end       token.Position        // Position of the character immediately after the scope.
	complLocs []*completionLocation // Locations where completions can be suggested.
	children  []*completionScope    // Child scopes nested inside this one.
}

// completionLocation represents a position after which some completions can be suggested in a scope.
type completionLocation struct {
	Position    token.Position // The earliest position in the scope that these completions can be suggested.
	Completions []*completion  // Completions which can be suggested.
}

// Complete returns completions for all identifiers in scope at the given position.
func (s *completionScope) Complete(pos *protocol.Position) []*completion {
	var compls []*completion

	for _, child := range s.children {
		if inRangePositions(pos, child.start, child.end) {
			compls = append(compls, child.Complete(pos)...)
			break
		}
	}

	for _, loc := range slices.Backward(s.complLocs) {
		locPos := newPosition(loc.Position)
		if pos.Line > locPos.Line || (pos.Line == locPos.Line && pos.Character >= locPos.Character) {
			compls = append(compls, loc.Completions...)
		}
	}

	return compls
}

type identCompletionGenerator struct {
	globalComplLocs []*completionLocation
	curScope        *completionScope

	globalScope *completionScope
}

func newIdentCompletionGenerator(programStart token.Position, programEnd token.Position) *identCompletionGenerator {
	globalScope := &completionScope{start: programStart, end: programEnd}
	return &identCompletionGenerator{globalScope: globalScope, curScope: globalScope}
}

func (g *identCompletionGenerator) Generate(program *ast.Program) *completionScope {
	g.globalComplLocs = g.readGlobalCompletionLocations(program)
	ast.Walk(program, g.walk)
	return g.globalScope
}

func (g *identCompletionGenerator) readGlobalCompletionLocations(program *ast.Program) []*completionLocation {
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
		locs = append(locs, &completionLocation{
			Position:    decl.Start(),
			Completions: []*completion{genCompletion(decl)},
		})
	}
	return locs
}

func (g *identCompletionGenerator) walk(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.VarDecl:
		g.walkVarDecl(node)
	case *ast.FunDecl:
		g.walkFunDecl(node)
	case *ast.FunExpr:
		g.walkFunExpr(node)
	case *ast.ClassDecl:
		g.walkClassDecl(node)
	case *ast.Block:
		g.walkBlock(node)
	default:
		return true
	}
	return false
}

func (g *identCompletionGenerator) walkVarDecl(decl *ast.VarDecl) {
	ast.Walk(decl.Initialiser, g.walk)
	if decl.Name.IsValid() && !decl.Semicolon.IsZero() {
		g.curScope.complLocs = append(g.curScope.complLocs, &completionLocation{
			Position:    decl.Semicolon.End(),
			Completions: []*completion{varCompletion(decl.Name.Token.Lexeme)},
		})
	}
}

func (g *identCompletionGenerator) walkFunDecl(decl *ast.FunDecl) {
	if !decl.Name.IsValid() || decl.Function == nil || decl.Function.Body == nil {
		return
	}

	funCompl := funCompletion(decl.Name.Token.Lexeme)

	extraCompls := []*completion{funCompl}
	if g.curScope == g.globalScope {
		forwardDeclaredCompls := g.globalCompletionsAfter(decl.Start())
		extraCompls = append(extraCompls, forwardDeclaredCompls...)
	}
	g.walkFun(decl.Function, extraCompls...)

	if !decl.Function.Body.RightBrace.IsZero() {
		g.curScope.complLocs = append(g.curScope.complLocs, &completionLocation{
			Position:    decl.Function.Body.RightBrace.End(),
			Completions: []*completion{funCompl},
		})
	}
}

func (g *identCompletionGenerator) walkFunExpr(expr *ast.FunExpr) {
	g.walkFun(expr.Function)
}

func (g *identCompletionGenerator) walkClassDecl(decl *ast.ClassDecl) {
	if !decl.Name.IsValid() || decl.Body == nil {
		return
	}

	classCompl := classCompletion(decl.Name.Token.Lexeme)

	extraMethodCompls := []*completion{classCompl}
	if g.curScope == g.globalScope {
		forwardDeclaredCompls := g.globalCompletionsAfter(decl.Start())
		extraMethodCompls = append(extraMethodCompls, forwardDeclaredCompls...)
	}
	for _, methodDecl := range decl.Methods() {
		g.walkFun(methodDecl.Function, extraMethodCompls...)
	}

	if !decl.Body.RightBrace.IsZero() {
		g.curScope.complLocs = append(g.curScope.complLocs, &completionLocation{
			Position:    decl.Body.RightBrace.End(),
			Completions: []*completion{classCompl},
		})
	}
}

func (g *identCompletionGenerator) walkBlock(block *ast.Block) {
	_, endScope := g.beginScope(block)
	for _, stmt := range block.Stmts {
		ast.Walk(stmt, g.walk)
	}
	endScope()
}

func (g *identCompletionGenerator) walkFun(fun *ast.Function, extraCompls ...*completion) {
	if fun == nil || fun.Body == nil {
		return
	}

	paramCompls := make([]*completion, 0, len(fun.Params))
	for _, paramDecl := range fun.Params {
		if paramDecl.IsValid() {
			paramCompls = append(paramCompls, &completion{Text: paramDecl.Name.Token.Lexeme, Kind: protocol.CompletionItemKindVariable})
		}
	}

	bodyScope, endBodyScope := g.beginScope(fun.Body)
	bodyScope.complLocs = append(bodyScope.complLocs, &completionLocation{
		Position:    bodyScope.start,
		Completions: slices.Concat(paramCompls, extraCompls),
	})
	for _, stmt := range fun.Body.Stmts {
		ast.Walk(stmt, g.walk)
	}
	endBodyScope()
}

func (g *identCompletionGenerator) beginScope(block *ast.Block) (*completionScope, func()) {
	childScope := &completionScope{
		start: g.curScope.start,
		end:   g.curScope.end,
	}
	if !block.LeftBrace.IsZero() {
		childScope.start = block.LeftBrace.End()
	}
	if !block.RightBrace.IsZero() {
		childScope.end = block.RightBrace.End()
	}
	g.curScope.children = append(g.curScope.children, childScope)

	prevCurScope := g.curScope
	g.curScope = childScope

	return childScope, func() {
		g.curScope = prevCurScope
	}
}

func (g *identCompletionGenerator) globalCompletionsAfter(pos token.Position) []*completion {
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

func varCompletion(name string) *completion {
	return &completion{
		Text: name,
		Kind: protocol.CompletionItemKindVariable,
	}
}

func funCompletion(name string) *completion {
	return &completion{
		Text:    name,
		Kind:    protocol.CompletionItemKindFunction,
		Snippet: callSnippet(name),
	}
}

func classCompletion(name string) *completion {
	return &completion{
		Text:    name,
		Kind:    protocol.CompletionItemKindClass,
		Snippet: callSnippet(name),
	}
}

func callSnippet(name string) string {
	return fmt.Sprintf("%s($1)$0", name)
}

func genCompletion(decl ast.Decl) *completion {
	switch decl.(type) {
	case *ast.VarDecl:
		return varCompletion(decl.Ident().Token.Lexeme)
	case *ast.FunDecl:
		return funCompletion(decl.Ident().Token.Lexeme)
	case *ast.ClassDecl:
		return classCompletion(decl.Ident().Token.Lexeme)
	case *ast.MethodDecl, *ast.ParamDecl:
		panic(fmt.Sprintf("unexpected declaration type: %T", decl))
	}
	panic("unreachable")
}

func genCompletions(decls []ast.Decl) []*completion {
	compls := make([]*completion, len(decls))
	for i, decl := range decls {
		compls[i] = genCompletion(decl)
	}
	return compls
}
