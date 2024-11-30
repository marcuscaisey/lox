package lsp

import (
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

func (h *Handler) textDocumentDidOpen(*protocol.DidOpenTextDocumentParams) error {
	return nil
}

func (h *Handler) textDocumentDidChange(*protocol.DidChangeTextDocumentParams) error {
	return nil
}

func (h *Handler) textDocumentDidClose(*protocol.DidCloseTextDocumentParams) error {
	return nil
}
