package lsp

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf16"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/format"
	"github.com/marcuscaisey/lox/golox/loxerr"
	"github.com/marcuscaisey/lox/golox/parser"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_definition
func (h *Handler) textDocumentDefinition(params *protocol.DefinitionParams) (*protocol.LocationOrLocationSlice, error) {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return nil, err
	}

	var defs []ast.Node
	if propertyDefs, ok := propertyDefinitions(doc, params.Position); ok {
		defs = propertyDefs
	} else if decl, ok := declaration(doc, params.Position); ok {
		defs = append(defs, decl.Ident())
	} else {
		return nil, nil
	}

	slices.SortFunc(defs, func(a, b ast.Node) int { return a.Start().Compare(b.Start()) })
	locs := make(protocol.LocationSlice, len(defs))
	for i, def := range defs {
		locs[i] = &protocol.Location{
			Uri:   filenameToURI(def.Start().File.Name),
			Range: newRange(def),
		}
	}

	return &protocol.LocationOrLocationSlice{Value: locs}, nil
}

func propertyDefinitions(doc *document, pos *protocol.Position) ([]ast.Node, bool) {
	var name string
	ast.Walk(doc.Program, func(n ast.Node) bool {
		var ident *ast.Ident
		switch n := n.(type) {
		case *ast.MethodDecl:
			ident = n.Name
		case *ast.GetExpr:
			ident = n.Name
		case *ast.SetExpr:
			ident = n.Name
		default:
			return true
		}
		if ident.IsValid() && inRange(pos, ident) {
			name = ident.Token.Lexeme
			return false
		} else {
			return true
		}
	})
	if name == "" {
		return nil, false
	}

	var defs []ast.Node
	ast.Walk(doc.Program, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.MethodDecl:
			if n.Name.IsValid() && n.Name.Token.Lexeme == name {
				defs = append(defs, n.Name)
			}
		case *ast.SetExpr:
			if n.Name.IsValid() && n.Name.Token.Lexeme == name {
				defs = append(defs, n.Name)
			}
		default:
		}
		return true
	})

	return defs, true
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_references
func (h *Handler) textDocumentReferences(params *protocol.ReferenceParams) (protocol.LocationSlice, error) {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return nil, err
	}

	var refs []ast.Node
	if propertyRefs, ok := propertyReferences(doc, params.Position, params.Context.IncludeDeclaration); ok {
		refs = propertyRefs
	} else if thisRefs, ok := thisReferences(doc, params.Position); ok {
		refs = thisRefs
	} else if declRefs, ok := declarationReferences(doc, params.Position, params.Context.IncludeDeclaration); ok {
		refs = declRefs
	} else {
		return nil, nil
	}

	slices.SortFunc(refs, func(a, b ast.Node) int { return a.Start().Compare(b.Start()) })
	locs := make(protocol.LocationSlice, len(refs))
	for i, ref := range refs {
		locs[i] = &protocol.Location{
			Uri:   filenameToURI(ref.Start().File.Name),
			Range: newRange(ref),
		}
	}

	return locs, nil
}

func propertyReferences(doc *document, pos *protocol.Position, includeDecl bool) ([]ast.Node, bool) {
	var name string
	ast.Walk(doc.Program, func(n ast.Node) bool {
		var ident *ast.Ident
		switch n := n.(type) {
		case *ast.MethodDecl:
			ident = n.Name
		case *ast.GetExpr:
			ident = n.Name
		case *ast.SetExpr:
			ident = n.Name
		default:
			return true
		}
		if ident.IsValid() && inRange(pos, ident) {
			name = ident.Token.Lexeme
			return false
		} else {
			return true
		}
	})
	if name == "" {
		return nil, false
	}

	var refs []ast.Node
	ast.Walk(doc.Program, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.MethodDecl:
			if n.Name.IsValid() && n.Name.Token.Lexeme == name && includeDecl {
				refs = append(refs, n.Name)
			}
		case *ast.GetExpr:
			if n.Name.IsValid() && n.Name.Token.Lexeme == name {
				refs = append(refs, n.Name)
			}
		case *ast.SetExpr:
			if n.Name.IsValid() && n.Name.Token.Lexeme == name {
				refs = append(refs, n.Name)
			}
		default:
		}
		return true
	})

	return refs, true
}

func thisReferences(doc *document, pos *protocol.Position) ([]ast.Node, bool) {
	if _, ok := outermostNodeAt[*ast.ThisExpr](doc.Program, pos); !ok {
		return nil, false
	}
	classDecl, ok := innermostNodeAt[*ast.ClassDecl](doc.Program, pos)
	if !ok {
		return nil, false
	}
	var refs []ast.Node
	ast.Walk(classDecl, func(thisExpr *ast.ThisExpr) bool {
		refs = append(refs, thisExpr)
		return false
	})
	return refs, true
}

func declarationReferences(doc *document, pos *protocol.Position, includeDecl bool) ([]ast.Node, bool) {
	decl, ok := declaration(doc, pos)
	if !ok {
		return nil, false
	}

	var refs []ast.Node
	for ident, identDecl := range doc.IdentDecls {
		if identDecl == decl && (includeDecl || ident != decl.Ident()) {
			refs = append(refs, ident)
		}
	}

	return refs, true
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_hover
func (h *Handler) textDocumentHover(params *protocol.HoverParams) (*protocol.Hover, error) {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return nil, err
	}

	decl, ok := declaration(doc, params.Position)
	if !ok {
		return nil, nil
	}

	var header string
	var body string
	switch decl := decl.(type) {
	case *ast.VarDecl, *ast.ParamDecl:
		header = fmt.Sprintf("var %s", decl.Ident().Token.Lexeme)
	case *ast.FunDecl:
		header = fmt.Sprintf("fun %s(%s)", decl.Name.Token.Lexeme, formatParams(decl.Function.Params))
		body = commentsText(decl.Doc)
	case *ast.ClassDecl:
		var b strings.Builder
		fmt.Fprintf(&b, "class %s", decl.Name.Token.Lexeme)
		if methods := decl.Methods(); len(methods) > 0 {
			fmt.Fprint(&b, " {\n")
			for _, methodDecl := range methods {
				fmt.Fprintf(&b, "    %s\n", hoverMethodDeclHeader(methodDecl))
			}
			fmt.Fprint(&b, "}")
		}
		header = b.String()
		body = commentsText(decl.Doc)
	case *ast.MethodDecl:
		header = hoverMethodDeclHeader(decl)
		body = commentsText(decl.Doc)
	}

	contentFormat := protocol.MarkupKindPlainText
	if len(h.capabilities.GetTextDocument().GetHover().GetContentFormat()) > 0 {
		contentFormat = h.capabilities.GetTextDocument().GetHover().GetContentFormat()[0]
	}

	var contents string
	if contentFormat == protocol.MarkupKindMarkdown {
		contents = fmt.Sprintf("```lox\n%s\n```", header)
		if body != "" {
			contents = fmt.Sprintf("%s\n---\n%s", contents, body)
		}
	} else {
		contents = header
		if body != "" {
			contents = fmt.Sprintf("%s\n%s", header, body)
		}
	}

	return &protocol.Hover{
		Contents: &protocol.MarkupContentOrMarkedStringOrMarkedStringSlice{
			Value: &protocol.MarkupContent{
				Kind:  contentFormat,
				Value: contents,
			},
		},
	}, nil
}

func hoverMethodDeclHeader(decl *ast.MethodDecl) string {
	var b strings.Builder
	for _, modifier := range decl.Modifiers {
		fmt.Fprintf(&b, "%s ", modifier.Lexeme)
	}
	fmt.Fprintf(&b, "%s(%s)", decl.Name.Token.Lexeme, formatParams(decl.Function.Params))
	return b.String()
}

func formatParams(params []*ast.ParamDecl) string {
	if len(params) == 0 {
		return ""
	}
	var b strings.Builder
	for i, param := range params {
		fmt.Fprint(&b, param.Name.Token.Lexeme)
		if i < len(params)-1 {
			fmt.Fprint(&b, ", ")
		}
	}
	return b.String()
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_documentSymbol
func (h *Handler) textDocumentDocumentSymbol(params *protocol.DocumentSymbolParams) (*protocol.SymbolInformationSliceOrDocumentSymbolSlice, error) {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return nil, err
	}

	var docSymbols protocol.DocumentSymbolSlice
	ast.Walk(doc.Program, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.VarDecl:
			if !n.Name.IsValid() {
				return false
			}
			docSymbols = append(docSymbols, &protocol.DocumentSymbol{
				Name:           n.Name.Token.Lexeme,
				Kind:           protocol.SymbolKindVariable,
				Range:          newRange(n),
				SelectionRange: newRange(n.Name),
			})
			return false
		case *ast.FunDecl:
			if !n.Name.IsValid() {
				return false
			}
			docSymbols = append(docSymbols, &protocol.DocumentSymbol{
				Name:           n.Name.Token.Lexeme,
				Detail:         funDetail(n.Function),
				Kind:           protocol.SymbolKindFunction,
				Range:          newRange(n),
				SelectionRange: newRange(n.Name),
			})
			return false
		case *ast.ClassDecl:
			if !n.Name.IsValid() {
				return false
			}
			class := &protocol.DocumentSymbol{
				Name:           n.Name.Token.Lexeme,
				Detail:         classDetail(n),
				Kind:           protocol.SymbolKindClass,
				Range:          newRange(n),
				SelectionRange: newRange(n.Name),
			}
			docSymbols = append(docSymbols, class)

			for _, method := range n.Methods() {
				if !method.Name.IsValid() {
					continue
				}
				modifiers := ""
				if len(method.Modifiers) > 0 {
					lexemes := make([]string, len(method.Modifiers))
					for i, modifier := range method.Modifiers {
						lexemes[i] = modifier.Lexeme
					}
					modifiers = fmt.Sprintf(" [%s]", strings.Join(lexemes, " "))
				}
				var kind protocol.SymbolKind
				switch {
				case method.IsConstructor():
					kind = protocol.SymbolKindConstructor
				default:
					kind = protocol.SymbolKindMethod
				}

				class.Children = append(class.Children, &protocol.DocumentSymbol{
					Name:           fmt.Sprintf("%s.%s%s", class.Name, method.Name.Token.Lexeme, modifiers),
					Detail:         funDetail(method.Function),
					Kind:           kind,
					Range:          newRange(method),
					SelectionRange: newRange(method.Name),
				})
			}
			return false
		default:
			return true
		}
	})

	var symbols protocol.SymbolInformationSliceOrDocumentSymbolSliceValue = docSymbols
	if !h.capabilities.GetTextDocument().GetDocumentSymbol().GetHierarchicalDocumentSymbolSupport() {
		symbols = toSymbolInformations(docSymbols, doc.URI)
	}
	return &protocol.SymbolInformationSliceOrDocumentSymbolSlice{Value: symbols}, nil
}

func toSymbolInformations(docSymbols protocol.DocumentSymbolSlice, uri string) protocol.SymbolInformationSlice {
	symbolInfos := make(protocol.SymbolInformationSlice, 0, len(docSymbols))
	for _, docSymbol := range docSymbols {
		symbolInfos = append(symbolInfos, &protocol.SymbolInformation{
			BaseSymbolInformation: &protocol.BaseSymbolInformation{
				Name: docSymbol.Name,
				Kind: docSymbol.Kind,
				Tags: docSymbol.Tags,
			},
			Location: &protocol.Location{
				Uri:   uri,
				Range: docSymbol.Range,
			},
		})
		for _, symbolInfo := range toSymbolInformations(docSymbol.Children, uri) {
			symbolInfo.ContainerName = docSymbol.Name
			symbolInfos = append(symbolInfos, symbolInfo)
		}
	}
	return symbolInfos
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_completion
func (h *Handler) textDocumentCompletion(params *protocol.CompletionParams) (*protocol.CompletionItemSliceOrCompletionList, error) {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return nil, err
	}

	if _, ok := outermostNodeAtOrBefore[*ast.Comment](doc.Program, params.Position); ok {
		return nil, nil
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

	completions := doc.Completor.Complete(params.Position)

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

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_formatting
func (h *Handler) textDocumentFormatting(params *protocol.DocumentFormattingParams) ([]*protocol.TextEdit, error) {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return nil, err
	}

	if doc.HasParseErrors {
		// TODO: return error here instead?
		h.log.Infof("textDocument/formatting: %s has errors. Skipping formatting.", params.TextDocument.Uri)
		return nil, nil
	}

	program, err := parser.Parse(strings.NewReader(doc.Text), doc.Filename, parser.WithComments(true))
	if err != nil {
		var loxErrs loxerr.Errors
		if errors.As(err, &loxErrs) {
			diagnostics := loxErrsToDiagnostics(loxErrs)
			err := h.client.TextDocumentPublishDiagnostics(&protocol.PublishDiagnosticsParams{
				Uri:         doc.URI,
				Version:     doc.Version,
				Diagnostics: diagnostics,
			})
			if err != nil {
				return nil, fmt.Errorf("textDocument/formatting: %w", err)
			}
			return nil, nil
		} else {
			return nil, fmt.Errorf("textDocument/formatting: %w", err)
		}
	}

	formatted := format.Node(program)
	if formatted == doc.Text {
		return nil, nil
	}

	textLines := strings.Split(strings.TrimSuffix(doc.Text, "\n"), "\n")
	return []*protocol.TextEdit{
		{
			Range: &protocol.Range{
				Start: &protocol.Position{Line: 0},
				End:   &protocol.Position{Line: len(textLines)},
			},
			NewText: formatted,
		},
	}, nil
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_rename
func (h *Handler) textDocumentRename(params *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return nil, err
	}

	references, ok := declarationReferences(doc, params.Position, true)
	if !ok {
		return nil, nil
	}

	edits := make([]*protocol.TextEditOrAnnotatedTextEdit, len(references))
	for i, reference := range references {
		edits[i] = &protocol.TextEditOrAnnotatedTextEdit{
			Value: &protocol.TextEdit{
				Range:   newRange(reference),
				NewText: params.NewName,
			},
		}
	}

	return &protocol.WorkspaceEdit{
		DocumentChanges: []*protocol.TextDocumentEditOrCreateFileOrRenameFileOrDeleteFile{
			{
				Value: &protocol.TextDocumentEdit{
					TextDocument: &protocol.OptionalVersionedTextDocumentIdentifier{
						TextDocumentIdentifier: &protocol.TextDocumentIdentifier{Uri: doc.URI},
						Version:                doc.Version,
					},
					Edits: edits,
				},
			},
		},
	}, nil
}

func declaration(doc *document, pos *protocol.Position) (ast.Decl, bool) {
	ident, ok := outermostNodeAt[*ast.Ident](doc.Program, pos)
	if !ok {
		return nil, false
	}
	decl, ok := doc.IdentDecls[ident]
	return decl, ok
}

func commentsText(doc []*ast.Comment) string {
	lines := make([]string, len(doc))
	for i, comment := range doc {
		lines[i] = strings.TrimSpace(strings.TrimPrefix(comment.Comment.Lexeme, "//"))
	}
	return strings.Join(lines, "\n")
}

func funDetail(fun *ast.Function) string {
	if fun == nil {
		return "fun()"
	}
	params := make([]string, 0, len(fun.Params))
	for _, paramDecl := range fun.Params {
		if paramDecl.Name.IsValid() {
			params = append(params, paramDecl.Name.Token.Lexeme)
		}
	}
	return fmt.Sprintf("fun(%s)", strings.Join(params, ", "))
}

func classDetail(decl *ast.ClassDecl) string {
	if !decl.Name.IsValid() {
		return ""
	}
	return fmt.Sprintf("class %s", decl.Name.Token.Lexeme)
}

func filenameToURI(filename string) string {
	return fmt.Sprintf("file://%s", filename)
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
