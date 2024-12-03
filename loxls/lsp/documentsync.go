package lsp

import (
	"errors"
	"fmt"
	"strings"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/lox"
	"github.com/marcuscaisey/lox/golox/parser"
	"github.com/marcuscaisey/lox/loxls/jsonrpc"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

type document struct {
	Text      string
	Program   ast.Program
	HasErrors bool
}

// document returns the document with the given URI, or an error if it doesn't exist.
func (h *Handler) document(uri protocol.DocumentUri) (*document, error) {
	doc, ok := h.docs[uri]
	if !ok {
		return nil, jsonrpc.NewError(jsonrpc.InvalidParams, "Document not found", map[string]any{"uri": uri})
	}
	return doc, nil
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_didOpen
func (h *Handler) textDocumentDidOpen(params *protocol.DidOpenTextDocumentParams) error {
	if err := h.updateDoc(params.TextDocument.Uri, params.TextDocument.Version, string(params.TextDocument.Text)); err != nil {
		return fmt.Errorf("textDocument/didOpen: %s", err)
	}
	return nil
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_didChange
func (h *Handler) textDocumentDidChange(params *protocol.DidChangeTextDocumentParams) error {
	for _, change := range params.ContentChanges {
		switch change := change.Value.(type) {
		case protocol.TextDocumentContentChangeEventOr1:
			return errors.New("textDocument/didChange: incremental updates not supported")
		case protocol.TextDocumentContentChangeEventOr2:
			if err := h.updateDoc(params.TextDocument.Uri, params.TextDocument.Version, string(change.Text)); err != nil {
				return fmt.Errorf("textDocument/didChange: %s", err)
			}
			return nil
		}
	}
	return nil
}

func (h *Handler) updateDoc(uri protocol.DocumentUri, version protocol.Integer, src string) error {
	program, err := parser.Parse(strings.NewReader(string(src)), parser.WithComments())
	diagnostics := protocol.DiagnosticSlice{}
	if err != nil {
		var loxErrs lox.Errors
		if !errors.As(err, &loxErrs) {
			return err
		}
		diagnostics = make(protocol.DiagnosticSlice, len(loxErrs))
		for i, e := range loxErrs {
			diagnostics[i] = protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      protocol.Uinteger(e.Start.Line - 1),
						Character: protocol.Uinteger(e.Start.ColumnUTF16()),
					},
					End: protocol.Position{
						Line:      protocol.Uinteger(e.End.Line - 1),
						Character: protocol.Uinteger(e.End.ColumnUTF16()),
					},
				},
				Severity: ptrTo(protocol.DiagnosticSeverityError),
				Source:   ptrTo(protocol.String("loxls")),
				Message:  protocol.String(e.Msg),
			}
		}
	}
	h.docs[uri] = &document{
		Text:      src,
		Program:   program,
		HasErrors: err != nil,
	}
	return h.client.TextDocumentPublishDiagnostics(&protocol.PublishDiagnosticsParams{
		Uri:         uri,
		Version:     &version,
		Diagnostics: diagnostics,
	})
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_didClose
func (h *Handler) textDocumentDidClose(*protocol.DidCloseTextDocumentParams) error {
	return nil
}
