package lsp

import (
	"errors"
	"fmt"
	"strings"

	"github.com/marcuscaisey/lox/golox/lox"
	"github.com/marcuscaisey/lox/golox/parser"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_didOpen
func (h *Handler) textDocumentDidOpen(params *protocol.DidOpenTextDocumentParams) error {
	if err := h.publishDiagnostics(params.TextDocument.Uri, params.TextDocument.Version, string(params.TextDocument.Text)); err != nil {
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
			if err := h.publishDiagnostics(params.TextDocument.Uri, params.TextDocument.Version, string(change.Text)); err != nil {
				return fmt.Errorf("textDocument/didChange: %s", err)
			}
			return nil
		}
	}
	return nil
}

func (h *Handler) publishDiagnostics(uri protocol.DocumentUri, version protocol.Integer, text string) error {
	r := strings.NewReader(string(text))
	_, err := parser.Parse(r)
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
	if err := h.client.TextDocumentPublishDiagnostics(&protocol.PublishDiagnosticsParams{
		Uri:         uri,
		Version:     &version,
		Diagnostics: diagnostics,
	}); err != nil {
		return err
	}
	return nil
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_didClose
func (h *Handler) textDocumentDidClose(*protocol.DidCloseTextDocumentParams) error {
	return nil
}
