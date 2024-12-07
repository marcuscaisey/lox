package lsp

import (
	"fmt"
	"strings"

	"github.com/marcuscaisey/lox/lox/ast"
	"github.com/marcuscaisey/lox/lox/format"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

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
				Name:           n.Name.Lexeme,
				Kind:           protocol.SymbolKindVariable,
				Range:          protocol.NewRange(n.Start(), n.End()),
				SelectionRange: protocol.NewRange(n.Name.Start(), n.Name.End()),
			})
			return false
		case ast.FunDecl:
			docSymbols = append(docSymbols, &protocol.DocumentSymbol{
				Name:           n.Name.Lexeme,
				Detail:         format.Signature(n.Function),
				Kind:           protocol.SymbolKindFunction,
				Range:          protocol.NewRange(n.Start(), n.End()),
				SelectionRange: protocol.NewRange(n.Name.Start(), n.Name.End()),
			})
			return false
		case ast.ClassDecl:
			class := &protocol.DocumentSymbol{
				Name:           n.Name.Lexeme,
				Kind:           protocol.SymbolKindClass,
				Range:          protocol.NewRange(n.Start(), n.End()),
				SelectionRange: protocol.NewRange(n.Name.Start(), n.Name.End()),
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
					Name:           fmt.Sprintf("%s.%s%s", class.Name, decl.Name.Lexeme, modifiers),
					Detail:         format.Signature(decl.Function),
					Kind:           kind,
					Range:          protocol.NewRange(decl.Start(), decl.End()),
					SelectionRange: protocol.NewRange(decl.Name.Start(), decl.Name.End()),
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
