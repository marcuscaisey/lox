package lsp

import (
	"fmt"
	"strings"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/format"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_definition
func (h *Handler) textDocumentDefinition(params *protocol.DefinitionParams) (*protocol.LocationOrLocationSlice, error) {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return nil, err
	}

	decl, ok := declaration(doc, params.Position)
	if !ok {
		return nil, nil
	}

	return &protocol.LocationOrLocationSlice{
		Value: &protocol.Location{
			Uri:   filenameToURI(decl.Start().File.Name()),
			Range: newRange(decl.Ident()),
		},
	}, nil
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_references
func (h *Handler) textDocumentReferences(params *protocol.ReferenceParams) (protocol.LocationSlice, error) {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return nil, err
	}

	decl, ok := declaration(doc, params.Position)
	if !ok {
		return nil, nil
	}

	references := references(doc, decl)
	var locations protocol.LocationSlice
	for _, reference := range references {
		if reference == decl.Ident() && !params.Context.IncludeDeclaration {
			continue
		}
		locations = append(locations, &protocol.Location{
			Uri:   filenameToURI(reference.Start().File.Name()),
			Range: newRange(reference),
		})
	}

	return locations, nil
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

	decl, ok := declaration(doc, params.Position)
	if !ok {
		return nil, nil
	}

	references := references(doc, decl)
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
	ident, ok := ast.Find(doc.Program, inRangePredicate[*ast.Ident](pos))
	if !ok {
		return nil, false
	}
	decl, ok := doc.IdentDecls[ident]
	return decl, ok
}

func references(doc *document, decl ast.Decl) []*ast.Ident {
	var references []*ast.Ident
	for ident, identDecl := range doc.IdentDecls {
		if identDecl.Start() == decl.Start() {
			references = append(references, ident)
		}
	}
	return references
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
