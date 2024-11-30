package lsp

import (
	"os"

	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

func (h *Handler) initialize(*protocol.InitializeParams) (*protocol.InitializeResult, error) {
	h.initialized = true
	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			PositionEncoding: ptrTo(protocol.PositionEncodingKindUTF16),
			TextDocumentSync: &protocol.TextDocumentSyncKindOrTextDocumentSyncOptions{
				Value: protocol.TextDocumentSyncOptions{
					OpenClose: ptrTo(protocol.Boolean(true)),
					Change:    ptrTo(protocol.TextDocumentSyncKindFull),
				},
			},
		},
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    "loxls",
			Version: ptrTo(protocol.String(version)),
		},
	}, nil
}

func (h *Handler) shutdown() (any, error) {
	h.shuttingDown = true
	return nil, nil
}

func (h *Handler) exit() error {
	code := 0
	if !h.shuttingDown {
		code = 1
	}
	os.Exit(code)
	return nil
}
