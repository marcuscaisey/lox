package lsp

// This file contains handlers for the methods described under
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#languageFeatures.

import (
	"fmt"
	"slices"
	"strings"

	"github.com/marcuscaisey/lox/golox/analyse"
	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/format"
	"github.com/marcuscaisey/lox/golox/token"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_definition
func (h *Handler) textDocumentDefinition(params *protocol.DefinitionParams) (*protocol.LocationOrLocationSlice, error) {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return nil, err
	}

	defs, ok := definitions(doc, params.Position)
	if !ok {
		return nil, nil
	}

	slices.SortFunc(defs, func(a, b ast.Binding) int { return a.Start().Compare(b.Start()) })
	locs := make(protocol.LocationSlice, len(defs))
	for i, def := range defs {
		locs[i] = &protocol.Location{
			Uri:   filenameToURI(def.Start().File.Name),
			Range: newRange(def.BoundIdent()),
		}
	}

	return &protocol.LocationOrLocationSlice{Value: locs}, nil
}

func definitions(doc *document, pos *protocol.Position) ([]ast.Binding, bool) {
	ident, ok := outermostNodeAt[*ast.Ident](doc.Program, pos)
	if !ok {
		return nil, false
	}
	bindings, ok := doc.IdentBindings[ident]
	return bindings, ok
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_references
func (h *Handler) textDocumentReferences(params *protocol.ReferenceParams) (protocol.LocationSlice, error) {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return nil, err
	}

	refs, ok := references(doc, params.Position, params.Context.IncludeDeclaration)
	if !ok {
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

func references(doc *document, pos *protocol.Position, includeDecl bool) (references []ast.Node, ok bool) {
	if thisRefs, ok := thisReferences(doc, pos); ok {
		return thisRefs, true
	}

	defs, ok := definitions(doc, pos)
	if !ok {
		return nil, false
	}

	var refs []ast.Node
identBindings:
	for ident, bindings := range doc.IdentBindings {
		for _, binding := range bindings {
			for _, def := range defs {
				if def == binding && (includeDecl || ident != def.BoundIdent()) {
					refs = append(refs, ident)
					continue identBindings
				}
			}
		}
	}

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
	ast.Walk(classDecl, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ClassDecl:
			if n != classDecl {
				return false
			}
		case *ast.ThisExpr:
			refs = append(refs, n)
			return false
		default:
		}
		return true
	})
	return refs, true
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_hover
func (h *Handler) textDocumentHover(params *protocol.HoverParams) (*protocol.Hover, error) {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return nil, err
	}

	defs, ok := definitions(doc, params.Position)
	if !ok {
		return nil, nil
	}

	var headers []string
	var body string
	for _, def := range defs {
		decl, ok := def.(ast.Decl)
		if !ok {
			continue
		}
		switch decl := decl.(type) {
		case *ast.VarDecl, *ast.ParamDecl:
			header, ok := varDetail(decl.BoundIdent())
			if !ok {
				continue
			}
			headers = append(headers, header)

		case *ast.FunDecl:
			header, ok := funDetail(decl)
			if !ok {
				continue
			}
			headers = append(headers, header)
			body = commentsText(decl.Doc)

		case *ast.ClassDecl:
			if !decl.Name.IsValid() {
				continue
			}
			b := &strings.Builder{}
			fmt.Fprintf(b, "class %s ", decl.Name)
			if decl.Superclass.IsValid() {
				fmt.Fprintf(b, "< %s ", decl.Superclass)
			}
			fmt.Fprint(b, "{")
			openingNewLineWritten := false
			seenStaticMethods := map[string]bool{}
			seenInstanceMethods := map[string]bool{}
			seenInstanceProps := map[string]bool{}
			seenStaticProps := map[string]bool{}
			for curClassDecl := range analyse.InheritanceChain(decl, doc.IdentBindings) {
				inheritedCommentWritten := false
				seenInstancePropsInClass := map[string]bool{}
				seenStaticPropsInClass := map[string]bool{}
				for _, methodDecl := range curClassDecl.Methods() {
					if !methodDecl.Name.IsValid() {
						continue
					}
					switch name := methodDecl.Name.String(); {
					case methodDecl.HasModifier(token.Static) && methodDecl.HasModifier(token.Get, token.Set):
						if seenStaticProps[name] && !seenStaticPropsInClass[name] {
							continue
						}
						seenStaticProps[name] = true
						seenStaticPropsInClass[name] = true
					case methodDecl.HasModifier(token.Get, token.Set):
						if seenInstanceProps[name] && !seenInstancePropsInClass[name] {
							continue
						}
						seenInstanceProps[name] = true
						seenInstancePropsInClass[name] = true
					case methodDecl.HasModifier(token.Static):
						if seenStaticMethods[name] {
							continue
						}
						seenStaticMethods[name] = true
					default:
						if seenInstanceMethods[name] {
							continue
						}
						seenInstanceMethods[name] = true
					}
					if !openingNewLineWritten {
						fmt.Fprint(b, "\n")
						openingNewLineWritten = true
					}
					if !inheritedCommentWritten && curClassDecl != decl {
						fmt.Fprintf(b, "  // Inherited from %s\n", curClassDecl.Name)
						inheritedCommentWritten = true
					}
					fmt.Fprintf(b, "  %s%s(%s)\n", formatMethodModifiers(methodDecl.Modifiers), methodDecl.Name, formatParams(methodDecl.GetParams()))
				}
			}
			fmt.Fprint(b, "}")
			headers = append(headers, b.String())
			body = commentsText(decl.Doc)

		case *ast.MethodDecl:
			classDecl, ok := innermostNodeAt[*ast.ClassDecl](doc.Program, newPosition(decl.Start()))
			if !ok {
				continue
			}
			header, ok := methodDetail(decl, classDecl)
			if !ok {
				continue
			}
			headers = append(headers, header)
			body = commentsText(decl.Doc)
		}
	}
	if len(headers) == 0 {
		return nil, nil
	}

	contentFormat := protocol.MarkupKindPlainText
	if len(h.capabilities.GetTextDocument().GetHover().GetContentFormat()) > 0 {
		contentFormat = h.capabilities.GetTextDocument().GetHover().GetContentFormat()[0]
	}

	header := strings.Join(headers, "\n")
	if len(headers) > 1 {
		body = fmt.Sprintf("%d implementations", len(headers))
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

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_documentSymbol
func (h *Handler) textDocumentDocumentSymbol(params *protocol.DocumentSymbolParams) (*protocol.SymbolInformationSliceOrDocumentSymbolSlice, error) {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return nil, err
	}

	var docSymbols protocol.DocumentSymbolSlice
	ast.Walk(doc.Program, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.VarDecl:
			if !decl.Name.IsValid() {
				return false
			}
			docSymbols = append(docSymbols, &protocol.DocumentSymbol{
				Name:           decl.Name.String(),
				Kind:           protocol.SymbolKindVariable,
				Range:          newRange(decl),
				SelectionRange: newRange(decl.Name),
			})
			return false
		case *ast.FunDecl:
			if !decl.Name.IsValid() {
				return false
			}
			docSymbols = append(docSymbols, &protocol.DocumentSymbol{
				Name:           decl.Name.String(),
				Detail:         funSignature(decl.GetParams()),
				Kind:           protocol.SymbolKindFunction,
				Range:          newRange(decl),
				SelectionRange: newRange(decl.Name),
			})
			return false
		case *ast.ClassDecl:
			if !decl.Name.IsValid() {
				return false
			}
			class := &protocol.DocumentSymbol{
				Name:           decl.Name.String(),
				Kind:           protocol.SymbolKindClass,
				Range:          newRange(decl),
				SelectionRange: newRange(decl.Name),
			}
			docSymbols = append(docSymbols, class)

			for _, methodDecl := range decl.Methods() {
				if !methodDecl.Name.IsValid() {
					continue
				}
				var kind protocol.SymbolKind
				switch {
				case methodDecl.IsConstructor():
					kind = protocol.SymbolKindConstructor
				default:
					kind = protocol.SymbolKindMethod
				}
				name, ok := formatMethodName(methodDecl, decl)
				if !ok {
					continue
				}
				class.Children = append(class.Children, &protocol.DocumentSymbol{
					Name:           name,
					Detail:         funSignature(methodDecl.GetParams()),
					Kind:           kind,
					Range:          newRange(methodDecl),
					SelectionRange: newRange(methodDecl.Name),
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

	completions, isIncomplete := doc.Completor.Complete(params.Position)

	padding := len(fmt.Sprint(len(completions)))
	items := make([]*protocol.CompletionItem, 0, len(completions))
	for _, completion := range completions {
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
		if completion.Snippet != "" {
			if !h.capabilities.GetTextDocument().GetCompletion().GetCompletionItem().GetSnippetSupport() {
				continue
			}
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

		items = append(items, &protocol.CompletionItem{
			Label:            completion.Label,
			LabelDetails:     completion.LabelDetails,
			Kind:             completion.Kind,
			Detail:           completion.Detail,
			Documentation:    documentation,
			InsertTextFormat: insertTextFormat,
			TextEdit:         textEdit,
			TextEditText:     textEditText,
			SortText:         fmt.Sprintf("%0*d", padding, len(items)),
		})
	}

	return &protocol.CompletionItemSliceOrCompletionList{
		Value: &protocol.CompletionList{
			IsIncomplete: isIncomplete,
			ItemDefaults: itemDefaults,
			Items:        items,
		},
	}, nil
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_signatureHelp
func (h *Handler) textDocumentSignatureHelp(params *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return nil, err
	}

	callExpr, ok := ast.FindLast(doc.Program, func(callExpr *ast.CallExpr) bool {
		if callExpr.LeftParen.IsZero() {
			return false
		}
		start := callExpr.LeftParen.End()
		if callExpr.RightParen.IsZero() {
			return inRangeOrFollowsPositions(params.Position, start, doc.Program.End())
		}
		return inRangePositions(params.Position, start, callExpr.RightParen.End())
	})
	if !ok {
		return nil, nil
	}

	var calleeIdent *ast.Ident
	switch callee := callExpr.Callee.(type) {
	case *ast.IdentExpr:
		calleeIdent = callee.Ident
	case *ast.GetExpr:
		calleeIdent = callee.Name
	default:
		return nil, nil
	}

	var signatures []*protocol.SignatureInformation
	for _, binding := range doc.IdentBindings[calleeIdent] {
		switch decl := binding.(type) {
		case *ast.FunDecl:
			prefix, ok := funDetailPrefix(decl)
			if !ok {
				continue
			}
			signatures = append(signatures, h.signature(prefix, decl.GetParams(), decl.Doc))

		case *ast.ClassDecl:
			if !decl.Name.IsValid() {
				continue
			}
			prefix := decl.Name.String()
			var params []*ast.ParamDecl
			doc := decl.Doc
			for _, methodDecl := range decl.Methods() {
				if methodDecl.IsConstructor() {
					prefixInner, ok := methodDetailPrefix(methodDecl, decl)
					if !ok {
						break
					}
					prefix = prefixInner
					params = methodDecl.GetParams()
					if len(methodDecl.Doc) > 0 {
						doc = methodDecl.Doc
					}
					break
				}
			}
			signatures = append(signatures, h.signature(prefix, params, doc))

		case *ast.MethodDecl:
			if decl.HasModifier(token.Get, token.Set) {
				continue
			}
			classDecl, ok := innermostNodeAt[*ast.ClassDecl](doc.Program, newPosition(decl.Start()))
			if !ok {
				continue
			}
			prefix, ok := methodDetailPrefix(decl, classDecl)
			if !ok {
				continue
			}
			signatures = append(signatures, h.signature(prefix, decl.GetParams(), decl.Doc))

		default:
		}
	}
	if len(signatures) == 0 {
		return nil, nil
	}

	activeSignature := protocol.NewOptional(0)
	contextSupport := h.capabilities.GetTextDocument().GetSignatureHelp().GetContextSupport()
	contextActiveSignature := params.GetContext().GetActiveSignatureHelp().GetActiveSignature()
	if contextSupport && contextActiveSignature != nil {
		activeSignature = contextActiveSignature
	}

	activeParameter := protocol.NewOptional(0)
	for i := range callExpr.Commas {
		start := callExpr.Commas[i].End()
		var end token.Position
		if i+1 < len(callExpr.Commas) {
			end = callExpr.Commas[i+1].End()
		} else if !callExpr.RightParen.IsZero() {
			end = callExpr.RightParen.End()
		} else {
			if inRangeOrFollowsPositions(params.Position, start, doc.Program.End()) {
				activeParameter = protocol.NewOptional(i + 1)
				break
			}
			continue
		}
		if inRangePositions(params.Position, start, end) {
			activeParameter = protocol.NewOptional(i + 1)
			break
		}
	}

	return &protocol.SignatureHelp{
		Signatures:      signatures,
		ActiveSignature: activeSignature,
		ActiveParameter: activeParameter,
	}, nil
}

func (h *Handler) signature(prefix string, params []*ast.ParamDecl, doc []*ast.Comment) *protocol.SignatureInformation {
	parameters := make([]*protocol.ParameterInformation, len(params))
	labelBuilder := new(strings.Builder)
	fmt.Fprint(labelBuilder, prefix, "(")
	labelOffsetSupport := h.capabilities.GetTextDocument().GetSignatureHelp().GetSignatureInformation().GetParameterInformation().GetLabelOffsetSupport()
	for i, paramDecl := range params {
		parameters[i] = &protocol.ParameterInformation{Label: &protocol.StringOrParameterInformationLabelRange{}}
		if labelOffsetSupport {
			parameters[i].Label.Value = &protocol.ParameterInformationLabelRange{
				Start: utf16StringLen(labelBuilder.String()),
				End:   utf16StringLen(labelBuilder.String() + paramDecl.Name.String()),
			}
		} else {
			parameters[i].Label.Value = protocol.String(paramDecl.Name.String())
		}
		fmt.Fprint(labelBuilder, paramDecl.Name)
		if i < len(params)-1 {
			fmt.Fprint(labelBuilder, ", ")
		}
	}
	fmt.Fprint(labelBuilder, ")")

	var documentation *protocol.StringOrMarkupContent
	if text := commentsText(doc); text != "" {
		kind := protocol.MarkupKindPlainText
		if format := h.capabilities.GetTextDocument().GetSignatureHelp().GetSignatureInformation().GetDocumentationFormat(); len(format) > 0 {
			kind = format[0]
		}
		documentation = &protocol.StringOrMarkupContent{
			Value: &protocol.MarkupContent{
				Kind:  kind,
				Value: text,
			},
		}
	}

	return &protocol.SignatureInformation{
		Label:         labelBuilder.String(),
		Documentation: documentation,
		Parameters:    parameters,
	}
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_formatting
func (h *Handler) textDocumentFormatting(params *protocol.DocumentFormattingParams) ([]*protocol.TextEdit, error) {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return nil, err
	}

	if doc.HasParseErrors {
		return nil, nil
	}

	formatted := format.Node(doc.Program)
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

	refs, ok := references(doc, params.Position, true)
	if !ok {
		return nil, nil
	}

	edits := make([]*protocol.TextEditOrAnnotatedTextEdit, len(refs))
	for i, ref := range refs {
		edits[i] = &protocol.TextEditOrAnnotatedTextEdit{
			Value: &protocol.TextEdit{
				Range:   newRange(ref),
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

func filenameToURI(filename string) string {
	return fmt.Sprintf("file://%s", filename)
}
