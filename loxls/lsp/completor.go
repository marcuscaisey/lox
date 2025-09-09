package lsp

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"
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

// completor provides completions of text that is being typed in a program.
type completor struct {
	program               *ast.Program
	identCompletor        *identCompletor
	keywordCompletor      *keywordCompletor
	builtinCompls         []*completion
	thisPropertyCompletor *thisPropertyCompletor
	propertyCompls        []*completion
}

// newCompletor returns a [completor] which provides completions inside the given program.
// builtins is a list of built-in declarations which are available in the global scope.
func newCompletor(program *ast.Program, builtins []ast.Decl) *completor {
	return &completor{
		program:               program,
		identCompletor:        newIdentCompletor(program),
		keywordCompletor:      newKeywordCompletor(program),
		builtinCompls:         declCompletions(builtins),
		thisPropertyCompletor: newThisPropertyCompletor(program),
		propertyCompls:        genPropertyCompletions(program),
	}
}

// Complete returns the completions which should be suggested at a position.
func (c *completor) Complete(pos *protocol.Position) []*completion {
	var getSetExprObject ast.Expr
	if getExpr, ok := ast.Find(c.program, inRangeOrFollowsPredicate[*ast.GetExpr](pos)); ok {
		getSetExprObject = getExpr.Object
	} else if setExpr, ok := ast.Find(c.program, func(setExpr *ast.SetExpr) bool {
		return inRangeOrFollows(pos, setExpr) && inRangeOrFollows(pos, setExpr.Name)
	}); ok {
		getSetExprObject = setExpr.Object
	}
	if getSetExprObject != nil {
		if _, ok := getSetExprObject.(*ast.ThisExpr); ok {
			return c.thisPropertyCompletor.Complete(pos)
		} else {
			return c.propertyCompls
		}
	} else {
		return slices.Concat(
			c.identCompletor.Complete(pos),
			c.keywordCompletor.Complete(pos),
			c.builtinCompls,
		)
	}
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
	ast.Walk(c.program, func(n ast.Stmt) bool {
		if block, ok := n.(*ast.Block); ok && !block.LeftBrace.IsZero() && equalPositions(prevCharEnd, block.LeftBrace.End()) {
			result = true
			return false
		}
		if n.IsValid() && equalPositions(prevCharEnd, n.End()) {
			result = true
			return false
		}
		return true
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

func newIdentCompletor(program *ast.Program) *identCompletor {
	globalScope := genIdentCompletions(program)
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

func genIdentCompletions(program *ast.Program) *completionScope {
	g := &identCompletionGenerator{}
	return g.Generate(program)
}

type identCompletionGenerator struct {
	globalComplLocs []*completionLocation
	curScope        *completionScope

	globalScope *completionScope
}

func (g *identCompletionGenerator) Generate(program *ast.Program) *completionScope {
	globalScope := &completionScope{start: program.Start(), end: program.End()}
	g.globalScope = globalScope
	g.curScope = globalScope
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

// genPropertyCompletions generates completions for all properties of all classes in the given program.
func genPropertyCompletions(program *ast.Program) []*completion {
	complsByClassDecl := genClassPropertyCompletions(program)
	var compls []*completion
	for classDecl, classCompls := range complsByClassDecl {
		for _, compl := range classCompls {
			b := &strings.Builder{}
			fmt.Fprintf(b, "_From class: %s_", classDecl.Name.Token.Lexeme)
			if compl.Documentation != "" {
				fmt.Fprintf(b, "\n\n%s", compl.Documentation)
			}
			compl.Documentation = b.String()
			compls = append(compls, compl)
		}
		_ = classDecl
	}
	return compls
}

// thisPropertyCompletor provides completions of properties for get and set expressions where the object is a this
// expression.
type thisPropertyCompletor struct {
	program           *ast.Program
	complsByClassDecl map[*ast.ClassDecl][]*completion
}

func newThisPropertyCompletor(program *ast.Program) *thisPropertyCompletor {
	complsByClassDecl := genClassPropertyCompletions(program)
	return &thisPropertyCompletor{program: program, complsByClassDecl: complsByClassDecl}
}

func (c *thisPropertyCompletor) Complete(pos *protocol.Position) []*completion {
	classDecl, ok := ast.FindLast(c.program, inRangePredicate[*ast.ClassDecl](pos))
	if !ok {
		return nil
	}
	return c.complsByClassDecl[classDecl]
}

func genClassPropertyCompletions(program *ast.Program) map[*ast.ClassDecl][]*completion {
	g := &propertyCompletionGenerator{complsByLabelByKindByClassDecl: map[*ast.ClassDecl]map[protocol.CompletionItemKind]map[string]*completion{}}
	return g.Generate(program)
}

type propertyCompletionGenerator struct {
	curClassDecl *ast.ClassDecl

	complsByLabelByKindByClassDecl map[*ast.ClassDecl]map[protocol.CompletionItemKind]map[string]*completion
}

func (g *propertyCompletionGenerator) Generate(program *ast.Program) map[*ast.ClassDecl][]*completion {
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
				xUnderscore := strings.HasPrefix(x.Label, "_")
				yUnderscore := strings.HasPrefix(y.Label, "_")
				if xUnderscore && !yUnderscore {
					return 1
				} else if !xUnderscore && yUnderscore {
					return -1
				} else {
					return cmp.Compare(x.Label, y.Label)
				}
			})
			complsByClassDecl[classDecl] = append(complsByClassDecl[classDecl], sortedCompls...)
		}
	}

	return complsByClassDecl
}

func (g *propertyCompletionGenerator) walk(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.ClassDecl:
		g.walkClassDecl(node)
		return false
	case *ast.MethodDecl:
		g.walkMethodDecl(node)
	case *ast.SetExpr:
		g.walkSetExpr(node)
	default:
	}
	return true
}

func (g *propertyCompletionGenerator) walkClassDecl(decl *ast.ClassDecl) {
	prevCurClassDecl := g.curClassDecl
	defer func() { g.curClassDecl = prevCurClassDecl }()
	g.curClassDecl = decl
	ast.Walk(decl.Body, g.walk)
}

func (g *propertyCompletionGenerator) walkMethodDecl(decl *ast.MethodDecl) {
	compl, ok := methodCompletion(decl)
	if !ok {
		return
	}
	g.add(g.curClassDecl, compl)
}

func (g *propertyCompletionGenerator) walkSetExpr(expr *ast.SetExpr) {
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

func (g *propertyCompletionGenerator) add(classDecl *ast.ClassDecl, compl *completion) {
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

// declCompletions returns completions for all of the provided declarations.
func declCompletions(decls []ast.Decl) []*completion {
	compls := make([]*completion, len(decls))
	for i, decl := range decls {
		if compl, ok := declCompletion(decl); ok {
			compls[i] = compl
		}
	}
	return compls
}
