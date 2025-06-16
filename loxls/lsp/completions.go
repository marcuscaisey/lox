package lsp

import (
	"cmp"
	"fmt"
	"maps"
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

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_completion
func (h *Handler) textDocumentCompletion(params *protocol.CompletionParams) (*protocol.CompletionItemSliceOrCompletionList, error) {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return nil, err
	}

	replaceRange := &protocol.Range{Start: params.Position, End: params.Position}
	if containingIdentRange, ok := containingIdentRange(doc.Program, params.Position); ok {
		replaceRange = containingIdentRange
	}
	insertRange := &protocol.Range{Start: replaceRange.Start, End: params.Position}

	var itemDefaults *protocol.CompletionListItemDefaults
	if slices.Contains(h.capabilities.GetTextDocument().GetCompletion().GetCompletionList().GetItemDefaults(), "editRange") {
		itemDefaults = &protocol.CompletionListItemDefaults{EditRange: &protocol.RangeOrInsertReplaceRange{}}
		if h.capabilities.GetTextDocument().GetCompletion().GetCompletionItem().GetInsertReplaceSupport() {
			itemDefaults.EditRange.Value = &protocol.InsertReplaceRange{Insert: insertRange, Replace: replaceRange}
		} else {
			itemDefaults.EditRange.Value = insertRange
		}
	}

	var completions []*completion
	if getExpr, ok := ast.Find(doc.Program, func(getExpr *ast.GetExpr) bool { return inRangeOrFollows(params.Position, getExpr) }); ok {
		if _, ok := getExpr.Object.(*ast.ThisExpr); ok {
			completions = doc.ThisPropertyCompletor.Complete(params.Position)
		} else {
			completions = doc.PropertyCompletor.Complete(params.Position)
		}
	} else {
		completions = slices.Concat(
			doc.IdentCompletor.Complete(params.Position),
			doc.KeywordCompletor.Complete(params.Position),
			h.builtinCompletions,
		)
	}

	padding := len(fmt.Sprint(len(completions)))
	items := make([]*protocol.CompletionItem, len(completions))
	for i, completion := range completions {
		var documentation *protocol.StringOrMarkupContent
		if completion.Documentation != "" {
			kind := protocol.MarkupKindPlainText
			if len(h.capabilities.GetTextDocument().GetCompletion().GetCompletionItem().GetDocumentationFormat()) > 0 {
				kind = h.capabilities.GetTextDocument().GetCompletion().GetCompletionItem().GetDocumentationFormat()[0]
			}
			documentation = &protocol.StringOrMarkupContent{
				Value: &protocol.MarkupContent{
					Kind:  kind,
					Value: completion.Documentation,
				},
			}
		}

		var insertTextFormat protocol.InsertTextFormat
		var snippet string
		var textEditText string
		if completion.Snippet != "" && h.capabilities.GetTextDocument().GetCompletion().GetCompletionItem().GetSnippetSupport() {
			insertTextFormat = protocol.InsertTextFormatSnippet
			snippet = completion.Snippet
			if itemDefaults != nil {
				textEditText = snippet
			}
		}

		var textEdit *protocol.TextEditOrInsertReplaceEdit
		if itemDefaults == nil {
			newText := completion.Label
			if snippet != "" {
				newText = snippet
			}
			textEdit = &protocol.TextEditOrInsertReplaceEdit{}
			if h.capabilities.GetTextDocument().GetCompletion().GetCompletionItem().GetInsertReplaceSupport() {
				textEdit.Value = &protocol.InsertReplaceEdit{NewText: newText, Insert: insertRange, Replace: replaceRange}
			} else {
				textEdit.Value = &protocol.TextEdit{Range: insertRange, NewText: newText}
			}
		}

		items[i] = &protocol.CompletionItem{
			Label:            completion.Label,
			Kind:             completion.Kind,
			Detail:           completion.Detail,
			Documentation:    documentation,
			InsertTextFormat: insertTextFormat,
			TextEdit:         textEdit,
			TextEditText:     textEditText,
			SortText:         fmt.Sprintf("%0*d", padding, i),
		}
	}

	return &protocol.CompletionItemSliceOrCompletionList{
		Value: &protocol.CompletionList{
			ItemDefaults: itemDefaults,
			Items:        items,
		},
	}, nil
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
	startChar := len(utf16.Encode(line[:startIdx]))

	if startChar == pos.Character {
		return nil, false
	}

	endIdx := posIdx
	for endIdx < len(line) && isAlphaNumeric(line[endIdx]) {
		endIdx++
	}
	endChar := len(utf16.Encode(line[:endIdx]))

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

// completion contains a subset of [protocol.CompletionItem] fields plus some others which can be used to populate a
// [protocol.CompletionItem] fully depending on the capabilities of the client.
type completion struct {
	// Label is the same as [protocol.CompletionItem] Label.
	Label string
	// Kind is the same as [protocol.CompletionItem] Kind.
	Kind protocol.CompletionItemKind
	// Detail is the same as [protocol.CompletionItem] Detail.
	Detail string

	// Snippet is the text that should be inserted if the client supports snippets.
	Snippet string
	// Documentation is the documentation that will be shown. If the client supports it, this will be displayed as
	// markdown.
	Documentation string
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
			Label: keyword,
			Kind:  protocol.CompletionItemKindKeyword,
		}
	}

	if c.validStatementPosition(pos) {
		for _, keywordSnippet := range statementKeywordSnippets {
			compls = append(compls, &completion{
				Label:   keywordSnippet.Keyword,
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
		compl, ok := declCompletion(decl)
		if !ok {
			continue
		}
		locs = append(locs, &completionLocation{
			Position:    decl.Start(),
			Completions: []*completion{compl},
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
	compl, ok := varCompletion(decl)
	if !ok {
		return
	}
	if decl.Semicolon.IsZero() {
		return
	}
	g.curScope.complLocs = append(g.curScope.complLocs, &completionLocation{
		Position:    decl.Semicolon.End(),
		Completions: []*completion{compl},
	})
}

func (g *identCompletionGenerator) walkFunDecl(decl *ast.FunDecl) {
	funCompl, ok := funCompletion(decl)
	if !ok {
		return
	}

	extraCompls := []*completion{funCompl}
	if g.curScope == g.globalScope {
		forwardDeclaredCompls := g.globalCompletionsAfter(decl.Start())
		extraCompls = append(extraCompls, forwardDeclaredCompls...)
	}
	g.walkFun(decl.Function, extraCompls...)

	if decl.Function != nil && decl.Function.Body != nil && !decl.Function.Body.RightBrace.IsZero() {
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
	classCompl, ok := classCompletion(decl)
	if !ok {
		return
	}

	thisCompl := &completion{Label: "this", Kind: protocol.CompletionItemKindKeyword}
	extraMethodCompls := []*completion{thisCompl, classCompl}
	if g.curScope == g.globalScope {
		forwardDeclaredCompls := g.globalCompletionsAfter(decl.Start())
		extraMethodCompls = append(extraMethodCompls, forwardDeclaredCompls...)
	}
	for _, methodDecl := range decl.Methods() {
		g.walkFun(methodDecl.Function, extraMethodCompls...)
	}

	if decl.Body != nil && !decl.Body.RightBrace.IsZero() {
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
		if paramDecl.Name.IsValid() {
			paramCompls = append(paramCompls, &completion{Label: paramDecl.Name.Token.Lexeme, Kind: protocol.CompletionItemKindVariable})
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

type propertyCompletor struct{}

func newPropertyCompletor(program *ast.Program) *propertyCompletor {
	return &propertyCompletor{}
}

func (c *propertyCompletor) Complete(pos *protocol.Position) []*completion {
	return nil
}

type thisPropertyCompletor struct {
	program                *ast.Program
	completionsByClassDecl map[*ast.ClassDecl][]*completion
}

func newThisPropertyCompletor(program *ast.Program) *thisPropertyCompletor {
	g := newThisPropertyCompletionGenerator()
	completionsByClassDecl := g.Generate(program)
	return &thisPropertyCompletor{program: program, completionsByClassDecl: completionsByClassDecl}
}

func (c *thisPropertyCompletor) Complete(pos *protocol.Position) []*completion {
	classDecl, ok := ast.Find(c.program, func(classDecl *ast.ClassDecl) bool {
		if !inRange(pos, classDecl) {
			return false
		}
		_, ok := ast.Find(classDecl.Body, func(classDecl *ast.ClassDecl) bool { return inRange(pos, classDecl) })
		return !ok
	})
	if !ok {
		return nil
	}
	return c.completionsByClassDecl[classDecl]
}

type thisPropertyCompletionGenerator struct {
	curClassDecl *ast.ClassDecl

	complsByLabelByKindByClassDecl map[*ast.ClassDecl]map[protocol.CompletionItemKind]map[string]*completion
}

func newThisPropertyCompletionGenerator() *thisPropertyCompletionGenerator {
	return &thisPropertyCompletionGenerator{complsByLabelByKindByClassDecl: map[*ast.ClassDecl]map[protocol.CompletionItemKind]map[string]*completion{}}
}

func (g *thisPropertyCompletionGenerator) Generate(program *ast.Program) map[*ast.ClassDecl][]*completion {
	ast.Walk(program, g.walk)

	kindOrder := []protocol.CompletionItemKind{
		protocol.CompletionItemKindField,
		protocol.CompletionItemKindProperty,
		protocol.CompletionItemKindMethod,
	}
	complsByClassDecl := map[*ast.ClassDecl][]*completion{}
	for classDecl, complsByLabelByKind := range g.complsByLabelByKindByClassDecl {
		for label := range complsByLabelByKind[protocol.CompletionItemKindProperty] {
			delete(complsByLabelByKind[protocol.CompletionItemKindField], label)
		}
		for _, kind := range kindOrder {
			sortedCompls := slices.SortedFunc(maps.Values(complsByLabelByKind[kind]), func(x, y *completion) int {
				return cmp.Compare(x.Label, y.Label)
			})
			complsByClassDecl[classDecl] = append(complsByClassDecl[classDecl], sortedCompls...)
		}
	}

	return complsByClassDecl
}

func (g *thisPropertyCompletionGenerator) walk(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.ClassDecl:
		g.walkClassDecl(node)
		return false
	case *ast.MethodDecl:
		g.walkMethodDecl(node)
	case *ast.SetExpr:
		g.walkSetExpr(node)
	}
	return true
}

func (g *thisPropertyCompletionGenerator) walkClassDecl(decl *ast.ClassDecl) {
	prevCurClassDecl := g.curClassDecl
	defer func() { g.curClassDecl = prevCurClassDecl }()
	g.curClassDecl = decl
	ast.Walk(decl.Body, g.walk)
}

func (g *thisPropertyCompletionGenerator) walkMethodDecl(decl *ast.MethodDecl) {
	compl, ok := methodCompletion(decl)
	if !ok {
		return
	}
	g.add(g.curClassDecl, compl)
}

func (g *thisPropertyCompletionGenerator) walkSetExpr(expr *ast.SetExpr) {
	if g.curClassDecl == nil || expr.Object == nil {
		return
	}
	if _, ok := expr.Object.(*ast.ThisExpr); !ok {
		return
	}
	compl, ok := fieldCompletion(expr.Name)
	if !ok {
		return
	}
	g.add(g.curClassDecl, compl)
}

func (g *thisPropertyCompletionGenerator) add(classDecl *ast.ClassDecl, compl *completion) {
	complsByLabelByKind, ok := g.complsByLabelByKindByClassDecl[classDecl]
	if !ok {
		complsByLabelByKind = map[protocol.CompletionItemKind]map[string]*completion{}
		g.complsByLabelByKindByClassDecl[classDecl] = complsByLabelByKind
	}
	complsByLabel, ok := complsByLabelByKind[compl.Kind]
	if !ok {
		complsByLabel = map[string]*completion{}
		complsByLabelByKind[compl.Kind] = complsByLabel
	}
	complsByLabel[compl.Label] = compl
}

func varCompletion(decl *ast.VarDecl) (*completion, bool) {
	if !decl.Name.IsValid() {
		return nil, false
	}
	return &completion{
		Label: decl.Name.Token.Lexeme,
		Kind:  protocol.CompletionItemKindVariable,
	}, true
}

func funCompletion(decl *ast.FunDecl) (*completion, bool) {
	if !decl.Name.IsValid() {
		return nil, false
	}
	return &completion{
		Label:         decl.Name.Token.Lexeme,
		Kind:          protocol.CompletionItemKindFunction,
		Detail:        funDetail(decl.Function),
		Snippet:       callSnippet(decl.Name.Token.Lexeme),
		Documentation: commentsText(decl.Doc),
	}, true
}

func classCompletion(decl *ast.ClassDecl) (*completion, bool) {
	if !decl.Name.IsValid() {
		return nil, false
	}
	return &completion{
		Label:         decl.Name.Token.Lexeme,
		Kind:          protocol.CompletionItemKindClass,
		Detail:        classDetail(decl),
		Snippet:       callSnippet(decl.Name.Token.Lexeme),
		Documentation: commentsText(decl.Doc),
	}, true
}

func fieldCompletion(ident *ast.Ident) (*completion, bool) {
	if !ident.IsValid() {
		return nil, false
	}
	return &completion{
		Label: ident.Token.Lexeme,
		Kind:  protocol.CompletionItemKindField,
	}, true
}

func methodCompletion(decl *ast.MethodDecl) (*completion, bool) {
	if !decl.Name.IsValid() {
		return nil, false
	}
	var kind protocol.CompletionItemKind
	var detail string
	var snippet string
	var documentation string
	if decl.HasModifier(token.Get, token.Set) {
		kind = protocol.CompletionItemKindProperty
	} else {
		kind = protocol.CompletionItemKindMethod
		detail = funDetail(decl.Function)
		snippet = callSnippet(decl.Name.Token.Lexeme)
		documentation = commentsText(decl.Doc)
	}
	return &completion{
		Label:         decl.Name.Token.Lexeme,
		Kind:          kind,
		Detail:        detail,
		Snippet:       snippet,
		Documentation: documentation,
	}, true
}

func callSnippet(name string) string {
	return fmt.Sprintf("%s($1)$0", name)
}

func declCompletion(decl ast.Decl) (*completion, bool) {
	switch decl := decl.(type) {
	case *ast.VarDecl:
		return varCompletion(decl)
	case *ast.FunDecl:
		return funCompletion(decl)
	case *ast.ClassDecl:
		return classCompletion(decl)
	case *ast.MethodDecl, *ast.ParamDecl:
		panic(fmt.Sprintf("unexpected declaration type: %T", decl))
	}
	panic("unreachable")
}

func declCompletions(decls []ast.Decl) []*completion {
	compls := make([]*completion, len(decls))
	for i, decl := range decls {
		if compl, ok := declCompletion(decl); ok {
			compls[i] = compl
		}
	}
	return compls
}
