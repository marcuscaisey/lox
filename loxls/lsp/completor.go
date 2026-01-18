package lsp

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf16"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/token"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

var (
	expressionKeywords = []string{"true", "false", "nil"}
	statementKeywords  = []string{"print", "var", "if", "else", "while", "for", "break", "continue", "fun", "return", "class"}
	statementSnippets  = []snippet{
		{"var", "var ${1:name} = ${2:value};$0", "Snippet for a variable"},
		{"if", "if ($1) {\n  $0\n}", "Snippet for an if statement"},
		{"while", "while ($1) {\n  $0\n}", "Snippet for a while loop"},
		{"for", "for (var ${1:i} = ${2:0}; $1 < ${3:n}; $1 = $1 + 1) {\n  $0\n}", "Snippet for a for loop"},
		{"fun", "fun ${1:name}($2) {\n  $0\n}", "Snippet for a function"},
		{"class", "class ${1:name} {\n  $0\n}", "Snippet for a class"},
	}
	classBodySnippets = []snippet{
		{"init", "init($1) {\n  $0\n}", "Snippet for a constructor"},
		{"method", "${1:name}($2) {\n  $0\n}", "Snippet for a method"},
		{"get", "get ${1:name}() {\n  $0\n}", "Snippet for a property getter"},
		{"set", "set ${1:name}(${2:value}) {\n  $0\n}", "Snippet for a property setter"},
		{"static", "static ${1:name}($2) {\n  $0\n}", "Snippet for a class method"},
		{"staticget", "static get ${1:name}() {\n  $0\n}", "Snippet for a class property getter"},
		{"staticset", "static set ${1:name}(${2:value}) {\n  $0\n}", "Snippet for a class property setter"},
	}
)

// completor provides completions of text that is being typed in a program.
type completor struct {
	program            *ast.Program
	classBodyCompletor *classBodyCompletor
	identCompletor     *identCompletor
	keywordCompletor   *keywordCompletor
	builtinCompls      []*completion
	propertyCompletor  *propertyCompletor
}

// newCompletor returns a [completor] which provides completions inside the given program.
// builtins is a list of built-in declarations which are available in the global scope.
func newCompletor(program *ast.Program, identBindings map[*ast.Ident][]ast.Binding, builtins []ast.Decl) *completor {
	return &completor{
		program:            program,
		classBodyCompletor: newClassBodyCompletor(program),
		identCompletor:     newIdentCompletor(program),
		keywordCompletor:   newKeywordCompletor(program),
		builtinCompls:      declCompletions(builtins),
		propertyCompletor:  newPropertyCompletor(program, identBindings),
	}
}

// Complete returns the completions which should be suggested at a position.
func (c *completor) Complete(pos *protocol.Position) (compls []*completion, isIncomplete bool) {
	if compls, isIncomplete, ok := c.classBodyCompletor.Complete(pos); ok {
		return compls, isIncomplete
	}
	if compls, ok := c.propertyCompletor.Complete(pos); ok {
		return compls, false
	}
	return slices.Concat(
		c.identCompletor.Complete(pos),
		c.keywordCompletor.Complete(pos),
		c.builtinCompls,
	), false
}

type classBodyCompletor struct {
	program *ast.Program
}

func newClassBodyCompletor(program *ast.Program) *classBodyCompletor {
	return &classBodyCompletor{program: program}
}

func (c *classBodyCompletor) Complete(pos *protocol.Position) (compls []*completion, isIncomplete bool, ok bool) {
	classDecl, ok := c.inClassBody(pos)
	if !ok {
		return nil, false, false
	}
	initDefined := false
	for _, methodDecl := range classDecl.Methods() {
		if methodDecl.Name.IsValid() && methodDecl.Name.String() == "init" {
			initDefined = true
			break
		}
	}
	compls = make([]*completion, 0, len(classBodySnippets))
	ident, inIdent := outermostNodeAtOrBefore[*ast.Ident](classDecl, pos)
	for _, snippet := range classBodySnippets {
		if snippet.label == "init" && initDefined {
			continue
		}
		compl := snippet.ToCompletion()
		if compl.Label == "method" && inIdent {
			compl.Label = ident.String()
			compl.Snippet = strings.ReplaceAll(compl.Snippet, "${1:name}", ident.String())
		}
		compls = append(compls, compl)
	}
	return compls, true, true
}

func (c *classBodyCompletor) inClassBody(pos *protocol.Position) (*ast.ClassDecl, bool) {
	classDecl, ok := innermostNodeAt[*ast.ClassDecl](c.program, pos)
	if !ok {
		return nil, false
	}
	if classDecl.Body == nil || classDecl.Body.LeftBrace.IsZero() || classDecl.Body.RightBrace.IsZero() {
		return nil, false
	}
	if !inRangePositions(pos, classDecl.Body.LeftBrace.End(), classDecl.Body.RightBrace.End()) {
		return nil, false
	}
	for _, methodDecl := range classDecl.Methods() {
		if methodDecl.Function != nil && inRange(pos, methodDecl.Function) {
			return nil, false
		}
	}
	return classDecl, true
}

type snippet struct {
	label   string
	content string
	doc     string
}

func (s snippet) ToCompletion() *completion {
	return &completion{
		Label:         s.label,
		Kind:          protocol.CompletionItemKindSnippet,
		Snippet:       s.content,
		Documentation: s.doc,
	}
}

// completion contains a subset of [protocol.CompletionItem] fields plus some others which can be used to populate a
// [protocol.CompletionItem] fully depending on the capabilities of the client.
type completion struct {
	// Label is the same as [protocol.CompletionItem] Label.
	Label string
	// LabelDetails is the same as [protocol.CompletionItem] LabelDetails.
	LabelDetails *protocol.CompletionItemLabelDetails
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

// Complete returns completions for keywords which are valid at the given position.
func (c *keywordCompletor) Complete(pos *protocol.Position) []*completion {
	compls := make([]*completion, 0, len(expressionKeywords))

	if c.validStatementPosition(pos) {
		for _, snippet := range statementSnippets {
			compls = append(compls, snippet.ToCompletion())
		}
		for _, keyword := range statementKeywords {
			compls = append(compls, &completion{
				Label: keyword,
				Kind:  protocol.CompletionItemKindKeyword,
			})
		}
	}

	for _, keyword := range expressionKeywords {
		compls = append(compls, &completion{
			Label: keyword,
			Kind:  protocol.CompletionItemKindKeyword,
		})
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

// previousCharacterEnd returns the end position of the previous non-whitespace character which isn't part of a comment
// and whether one exists.
func (c *keywordCompletor) previousCharacterEnd(pos *protocol.Position) (*protocol.Position, bool) {
	lastCharEnd := func(line []rune) (int, bool) {
		if len(line) == 0 {
			return 0, false
		}
		commentIdx := len(line)
		for i := range line[:len(line)-1] {
			if line[i] == '/' && line[i+1] == '/' {
				commentIdx = i
				break
			}
		}
		for i, rune := range slices.Backward(line[:commentIdx]) {
			if !unicode.IsSpace(rune) {
				return utf16RunesLen(line[:i+1]), true
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
	compl, ok := varCompletion(decl.Name)
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
	defer endScope()
	ast.WalkChildren(block, g.walk)
}

func (g *identCompletionGenerator) walkFun(fun *ast.Function, extraCompls ...*completion) {
	if fun == nil {
		return
	}

	paramCompls := make([]*completion, 0, len(fun.Params))
	for _, paramDecl := range fun.Params {
		compl, ok := varCompletion(paramDecl.Name)
		if !ok {
			break
		}
		paramCompls = append(paramCompls, compl)
	}

	bodyScope, endBodyScope := g.beginScope(fun.Body)
	defer endBodyScope()
	bodyScope.complLocs = append(bodyScope.complLocs, &completionLocation{
		Position:    bodyScope.start,
		Completions: slices.Concat(paramCompls, extraCompls),
	})
	ast.WalkChildren(fun, g.walk)
}

func (g *identCompletionGenerator) beginScope(block *ast.Block) (*completionScope, func()) {
	childScope := &completionScope{
		start: g.curScope.start,
		end:   g.curScope.end,
	}
	if block != nil {
		if !block.LeftBrace.IsZero() {
			childScope.start = block.LeftBrace.End()
		}
		if !block.RightBrace.IsZero() {
			childScope.end = block.RightBrace.End()
		}
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

type propertyType int

const (
	propertyTypeInstance propertyType = iota
	propertyTypeStatic
)

// propertyCompletor provides completions of properties for get and set expressions.
type propertyCompletor struct {
	program                     *ast.Program
	identBindings               map[*ast.Ident][]ast.Binding
	compls                      []*completion
	complsByPropTypeByClassDecl map[*ast.ClassDecl]map[propertyType][]*completion
}

func newPropertyCompletor(program *ast.Program, identBindings map[*ast.Ident][]ast.Binding) *propertyCompletor {
	complsByPropTypeByClassDecl := genPropertyCompletions(program, identBindings)
	seenCompls := map[*completion]bool{}
	var allCompls []*completion
	for _, complsByPropType := range complsByPropTypeByClassDecl {
		for _, compls := range complsByPropType {
			sortPropertyCompletions(compls)
			for _, compl := range compls {
				if seenCompls[compl] {
					continue
				}
				seenCompls[compl] = true
				allCompls = append(allCompls, compl)
			}
		}
	}
	sortPropertyCompletions(allCompls)
	return &propertyCompletor{
		program:                     program,
		identBindings:               identBindings,
		compls:                      allCompls,
		complsByPropTypeByClassDecl: complsByPropTypeByClassDecl,
	}
}

func (c *propertyCompletor) Complete(pos *protocol.Position) ([]*completion, bool) {
	var object ast.Expr
	inRangeOrFollowsName := func(setExpr *ast.SetExpr) bool { return setExpr.Name.IsValid() && inRangeOrFollows(pos, setExpr.Name) }
	if getExpr, ok := outermostNodeAtOrBefore[*ast.GetExpr](c.program, pos); ok {
		object = getExpr.Object
	} else if setExpr, ok := ast.Find(c.program, inRangeOrFollowsName); ok {
		object = setExpr.Object
	} else {
		return nil, false
	}

	if _, ok := object.(*ast.ThisExpr); ok {
		classDecl, ok := innermostNodeAt[*ast.ClassDecl](c.program, pos)
		if !ok {
			return nil, true
		}
		methodDecl, ok := innermostNodeAt[*ast.MethodDecl](classDecl, pos)
		if !ok {
			return nil, true
		}
		propType := propertyTypeInstance
		if methodDecl.HasModifier(token.Static) {
			propType = propertyTypeStatic
		}
		return c.complsByPropTypeByClassDecl[classDecl][propType], true
	}

	if _, ok := object.(*ast.SuperExpr); ok {
		classDecl, ok := innermostNodeAt[*ast.ClassDecl](c.program, pos)
		if !ok {
			return nil, true
		}
		methodDecl, ok := innermostNodeAt[*ast.MethodDecl](classDecl, pos)
		if !ok {
			return nil, true
		}
		superclassBindings, ok := c.identBindings[classDecl.Superclass]
		if !ok {
			return nil, true
		}
		superclassDecl, ok := superclassBindings[0].(*ast.ClassDecl)
		if !ok {
			return nil, true
		}
		propType := propertyTypeInstance
		if methodDecl.HasModifier(token.Static) {
			propType = propertyTypeStatic
		}
		compls := make([]*completion, 0, len(c.complsByPropTypeByClassDecl[superclassDecl][propType]))
		for _, compl := range c.complsByPropTypeByClassDecl[superclassDecl][propType] {
			if compl.Kind == protocol.CompletionItemKindMethod {
				compls = append(compls, compl)
			}
		}
		return compls, true
	}

	if identExpr, ok := object.(*ast.IdentExpr); ok {
		if bindings, ok := c.identBindings[identExpr.Ident]; ok {
			if classDecl, ok := bindings[0].(*ast.ClassDecl); ok {
				return c.complsByPropTypeByClassDecl[classDecl][propertyTypeStatic], true
			}
		}
	}

	return c.compls, true
}

func genPropertyCompletions(program *ast.Program, identBindings map[*ast.Ident][]ast.Binding) map[*ast.ClassDecl]map[propertyType][]*completion {
	g := &propertyCompletionGenerator{
		complLabelsByPropTypeByClassDecl: map[*ast.ClassDecl]map[propertyType]map[string]bool{},
		identBindings:                    identBindings,
		complsByPropTypeByClassDecl:      map[*ast.ClassDecl]map[propertyType][]*completion{},
	}
	return g.Generate(program)
}

type propertyCompletionGenerator struct {
	curClassDecl                     *ast.ClassDecl
	curMethodDecl                    *ast.MethodDecl
	complLabelsByPropTypeByClassDecl map[*ast.ClassDecl]map[propertyType]map[string]bool
	identBindings                    map[*ast.Ident][]ast.Binding

	complsByPropTypeByClassDecl map[*ast.ClassDecl]map[propertyType][]*completion
}

func (g *propertyCompletionGenerator) Generate(program *ast.Program) map[*ast.ClassDecl]map[propertyType][]*completion {
	ast.Walk(program, g.walk)
	return g.complsByPropTypeByClassDecl
}

func (g *propertyCompletionGenerator) walk(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.ClassDecl:
		g.walkClassDecl(node)
		return false
	case *ast.MethodDecl:
		g.walkMethodDecl(node)
		return false
	case *ast.SetExpr:
		g.addFieldCompletion(node)
		return true
	default:
		return true
	}
}

func (g *propertyCompletionGenerator) walkClassDecl(decl *ast.ClassDecl) {
	prevCurClassDecl := g.curClassDecl
	defer func() { g.curClassDecl = prevCurClassDecl }()
	g.curClassDecl = decl

	g.complLabelsByPropTypeByClassDecl[decl] = map[propertyType]map[string]bool{
		propertyTypeInstance: {},
		propertyTypeStatic:   {},
	}
	g.complsByPropTypeByClassDecl[decl] = map[propertyType][]*completion{
		propertyTypeInstance: nil,
		propertyTypeStatic:   nil,
	}

	// Add completions for all methods before walking any of their bodies so that we can skip adding completions for
	// fields which already have a property completion.
	for _, methodDecl := range decl.Methods() {
		g.addCompletionForMethod(methodDecl)
	}

	ast.WalkChildren(decl, g.walk)

	superclassBindings, ok := g.identBindings[decl.Superclass]
	if !ok {
		return
	}
	superclassDecl, ok := superclassBindings[0].(*ast.ClassDecl)
	if !ok {
		return
	}
	for propType, compls := range g.complsByPropTypeByClassDecl[superclassDecl] {
		for _, compl := range compls {
			if !g.complLabelsByPropTypeByClassDecl[decl][propType][compl.Label] {
				g.complLabelsByPropTypeByClassDecl[decl][propType][compl.Label] = true
				g.complsByPropTypeByClassDecl[decl][propType] = append(g.complsByPropTypeByClassDecl[decl][propType], compl)
			}
		}
	}
}

func (g *propertyCompletionGenerator) walkMethodDecl(decl *ast.MethodDecl) {
	prevCurMethodDecl := g.curMethodDecl
	defer func() { g.curMethodDecl = prevCurMethodDecl }()
	g.curMethodDecl = decl
	ast.WalkChildren(decl, g.walk)
}

func (g *propertyCompletionGenerator) addCompletionForMethod(decl *ast.MethodDecl) {
	if !decl.Name.IsValid() || decl.IsConstructor() || g.curClassDecl == nil || !g.curClassDecl.Name.IsValid() {
		return
	}

	label := decl.Name.String()
	propType := propertyTypeInstance
	if decl.HasModifier(token.Static) {
		propType = propertyTypeStatic
	}
	if g.complLabelsByPropTypeByClassDecl[g.curClassDecl][propType][label] {
		return
	}
	g.complLabelsByPropTypeByClassDecl[g.curClassDecl][propType][label] = true
	var kind protocol.CompletionItemKind
	var detail string
	var documentation string
	if decl.HasModifier(token.Get, token.Set) {
		kind = protocol.CompletionItemKindProperty
	} else {
		kind = protocol.CompletionItemKindMethod
		var ok bool
		detail, ok = methodDetail(decl, g.curClassDecl)
		if !ok {
			return
		}
		documentation = commentsText(decl.Doc)
	}
	g.complsByPropTypeByClassDecl[g.curClassDecl][propType] = append(g.complsByPropTypeByClassDecl[g.curClassDecl][propType], &completion{
		Label:         label,
		LabelDetails:  &protocol.CompletionItemLabelDetails{Detail: fmt.Sprint(" ", g.curClassDecl.Name)},
		Kind:          kind,
		Detail:        detail,
		Documentation: documentation,
	})
}

func (g *propertyCompletionGenerator) addFieldCompletion(expr *ast.SetExpr) {
	if expr.Object == nil || g.curClassDecl == nil || !g.curClassDecl.Name.IsValid() {
		return
	}
	if _, ok := expr.Object.(*ast.ThisExpr); !ok {
		return
	}
	if !expr.Name.IsValid() {
		return
	}

	label := expr.Name.String()
	propType := propertyTypeInstance
	if g.curMethodDecl.HasModifier(token.Static) {
		propType = propertyTypeStatic
	}
	if g.complLabelsByPropTypeByClassDecl[g.curClassDecl][propType][label] {
		return
	}
	g.complLabelsByPropTypeByClassDecl[g.curClassDecl][propType][label] = true
	g.complsByPropTypeByClassDecl[g.curClassDecl][propType] = append(g.complsByPropTypeByClassDecl[g.curClassDecl][propType], &completion{
		Label:        label,
		LabelDetails: &protocol.CompletionItemLabelDetails{Detail: fmt.Sprint(" ", g.curClassDecl.Name)},
		Kind:         protocol.CompletionItemKindField,
	})
}

func sortPropertyCompletions(compls []*completion) {
	orders := []func(c *completion) string{
		func(c *completion) string {
			if strings.HasPrefix(c.Label, "_") {
				return "1"
			}
			return "0"
		},
		func(c *completion) string {
			if c.Kind == protocol.CompletionItemKindMethod {
				return "0"
			}
			return "1"
		},
		func(c *completion) string {
			return c.Label
		},
		func(c *completion) string {
			if c.LabelDetails != nil {
				return c.LabelDetails.Detail
			}
			return ""
		},
	}
	slices.SortFunc(compls, func(x, y *completion) int {
		for _, order := range orders {
			if c := cmp.Compare(order(x), order(y)); c != 0 {
				return c
			}
		}
		return 0
	})
}

func varCompletion(name *ast.Ident) (*completion, bool) {
	if !name.IsValid() {
		return nil, false
	}
	detail, ok := varDetail(name)
	if !ok {
		return nil, false
	}
	return &completion{
		Label:  name.String(),
		Detail: detail,
		Kind:   protocol.CompletionItemKindVariable,
	}, true
}

func funCompletion(decl *ast.FunDecl) (*completion, bool) {
	if !decl.Name.IsValid() {
		return nil, false
	}
	detail, ok := funDetail(decl)
	if !ok {
		return nil, false
	}
	return &completion{
		Label:         decl.Name.String(),
		Kind:          protocol.CompletionItemKindFunction,
		Detail:        detail,
		Documentation: commentsText(decl.Doc),
	}, true
}

func classCompletion(decl *ast.ClassDecl) (*completion, bool) {
	if !decl.Name.IsValid() {
		return nil, false
	}
	detail, ok := classDetail(decl)
	if !ok {
		return nil, false
	}
	return &completion{
		Label:         decl.Name.String(),
		Kind:          protocol.CompletionItemKindClass,
		Detail:        detail,
		Documentation: commentsText(decl.Doc),
	}, true
}

func declCompletion(decl ast.Decl) (*completion, bool) {
	switch decl := decl.(type) {
	case *ast.VarDecl:
		return varCompletion(decl.Name)
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
