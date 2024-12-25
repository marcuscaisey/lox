package lsp

import (
	"fmt"
	"strings"

	"github.com/marcuscaisey/lox/lox/ast"
	"github.com/marcuscaisey/lox/lox/format"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_definition
func (h *Handler) textDocumentDefinition(params *protocol.DefinitionParams) (*protocol.LocationOrLocationSlice, error) {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return nil, err
	}

	var ident ast.Ident
	ast.Walk(doc.Program, func(n ast.Node) bool {
		switch n := n.(type) {
		case ast.Ident:
			if posInRange(params.Position, n) {
				ident = n
			}
			return false
		default:
			return true
		}
	})
	if ident == (ast.Ident{}) {
		return nil, nil
	}

	decl, ok := doc.IdentDecls[ident]
	if !ok {
		return nil, nil
	}

	return &protocol.LocationOrLocationSlice{
		Value: &protocol.Location{
			Uri:   doc.URI,
			Range: newRange(decl.Start(), decl.End()),
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
		switch n := n.(type) {
		case ast.VarDecl:
			docSymbols = append(docSymbols, &protocol.DocumentSymbol{
				Name:           n.Name.Token.Lexeme,
				Kind:           protocol.SymbolKindVariable,
				Range:          newRange(n.Start(), n.End()),
				SelectionRange: newRange(n.Name.Start(), n.Name.End()),
			})
			return false
		case ast.FunDecl:
			docSymbols = append(docSymbols, &protocol.DocumentSymbol{
				Name:           n.Name.Token.Lexeme,
				Detail:         format.Signature(n.Function),
				Kind:           protocol.SymbolKindFunction,
				Range:          newRange(n.Start(), n.End()),
				SelectionRange: newRange(n.Name.Start(), n.Name.End()),
			})
			return false
		case ast.ClassDecl:
			class := &protocol.DocumentSymbol{
				Name:           n.Name.Token.Lexeme,
				Kind:           protocol.SymbolKindClass,
				Range:          newRange(n.Start(), n.End()),
				SelectionRange: newRange(n.Name.Start(), n.Name.End()),
			}
			docSymbols = append(docSymbols, class)
			for _, decl := range n.Methods() {
				modifiers := ""
				if len(decl.Modifiers) > 0 {
					lexemes := make([]string, len(decl.Modifiers))
					for i, modifier := range decl.Modifiers {
						lexemes[i] = modifier.Lexeme
					}
					modifiers = fmt.Sprintf(" [%s]", strings.Join(lexemes, " "))
				}
				var kind protocol.SymbolKind
				switch {
				case decl.IsConstructor():
					kind = protocol.SymbolKindConstructor
				default:
					kind = protocol.SymbolKindMethod
				}
				class.Children = append(class.Children, &protocol.DocumentSymbol{
					Name:           fmt.Sprintf("%s.%s%s", class.Name, decl.Name.Token.Lexeme, modifiers),
					Detail:         format.Signature(decl.Function),
					Kind:           kind,
					Range:          newRange(decl.Start(), decl.End()),
					SelectionRange: newRange(decl.Name.Start(), decl.Name.End()),
				})
			}
			return false
		default:
			return true
		}
	})

	var symbols protocol.SymbolInformationSliceOrDocumentSymbolSliceValue = docSymbols
	if !h.clientSupportsHierarchicalDocumentSymbols {
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

	if doc.HasErrors {
		// TODO: return error here instead?
		h.log.Infof("textDocument/formatting: %s has errors. Skipping formatting.", params.TextDocument.Uri)
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
