package lsp

import (
	"errors"
	"fmt"
	"strings"

	"github.com/marcuscaisey/lox/lox"
	"github.com/marcuscaisey/lox/lox/analysis"
	"github.com/marcuscaisey/lox/lox/ast"
	"github.com/marcuscaisey/lox/lox/parser"
	"github.com/marcuscaisey/lox/loxls/jsonrpc"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

type document struct {
	URI        string
	Version    int
	Text       string
	Program    ast.Program
	IdentDecls map[ast.Ident]ast.Decl
	HasErrors  bool
}

// document returns the document with the given URI, or an error if it doesn't exist.
func (h *Handler) document(uri string) (*document, error) {
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
	var identDecls map[ast.Ident]ast.Decl
	if err != nil {
		if !errors.As(err, &loxErrs) {
			return err
		}
	} else {
		identDecls, loxErrs = analysis.ResolveIdents(program)
		loxErrs = append(loxErrs, analysis.CheckSemantics(program)...)
		loxErrs.Sort()
	}

	diagnostics := make([]*protocol.Diagnostic, len(loxErrs))
	for i, e := range loxErrs {
		diagnostics[i] = &protocol.Diagnostic{
			Range:    newRange(e.Start, e.End),
			Severity: protocol.DiagnosticSeverityError,
			Source:   "loxls",
			Message:  e.Msg,
		}
	}

	h.docs[uri] = &document{
		URI:        uri,
		Version:    version,
		Text:       src,
		Program:    program,
		IdentDecls: identDecls,
		HasErrors:  err != nil,
	}

	return h.client.TextDocumentPublishDiagnostics(&protocol.PublishDiagnosticsParams{
		Uri:         uri,
		Version:     version,
		Diagnostics: diagnostics,
	})
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_didClose
func (h *Handler) textDocumentDidClose(params *protocol.DidCloseTextDocumentParams) error {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return err
	}
	delete(h.docs, doc.URI)
	return nil
}
