package lsp

import (
	"errors"
	"fmt"
	"strings"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/interpreter"
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
func (h *Handler) document(uri string) (*document, error) {
	doc, ok := h.docsByURI[uri]
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
		case *protocol.IncrementalTextDocumentContentChangeEvent:
			return errors.New("textDocument/didChange: incremental updates not supported")
		case *protocol.FullTextDocumentContentChangeEvent:
			if err := h.updateDoc(params.TextDocument.Uri, params.TextDocument.Version, string(change.Text)); err != nil {
				return fmt.Errorf("textDocument/didChange: %s", err)
			}
			return nil
		}
	}
	return nil
}

func (h *Handler) updateDoc(uri string, version int, src string) error {
	program, err := parser.Parse(strings.NewReader(string(src)), parser.WithComments())

	var loxErrs lox.Errors
	if err != nil {
		if !errors.As(err, &loxErrs) {
			return err
		}
	} else {
		_, loxErrs = interpreter.ResolveIdents(program)
		loxErrs = append(loxErrs, interpreter.CheckSemantics(program)...)
	}

	diagnostics := make([]*protocol.Diagnostic, len(loxErrs))
	for i, e := range loxErrs {
		diagnostics[i] = &protocol.Diagnostic{
			Range: &protocol.Range{
				Start: &protocol.Position{
					Line:      e.Start.Line - 1,
					Character: e.Start.ColumnUTF16(),
				},
				End: &protocol.Position{
					Line:      e.End.Line - 1,
					Character: e.End.ColumnUTF16(),
				},
			},
			Severity: protocol.DiagnosticSeverityError,
			Source:   "loxls",
			Message:  e.Msg,
		}
	}

	h.docsByURI[uri] = &document{
		Text:      src,
		Program:   program,
		HasErrors: err != nil,
	}

	return h.client.TextDocumentPublishDiagnostics(&protocol.PublishDiagnosticsParams{
		Uri:         uri,
		Version:     version,
		Diagnostics: diagnostics,
	})
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_didClose
func (h *Handler) textDocumentDidClose(params *protocol.DidCloseTextDocumentParams) error {
	delete(h.docsByURI, params.TextDocument.Uri)
	return nil
}
