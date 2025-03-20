package lsp

import (
	"os"

	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#initialize
func (h *Handler) initialize(params *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	h.initialized = true

	if textDocument := params.Capabilities.TextDocument; textDocument != nil {
		if documentSymbol := textDocument.DocumentSymbol; documentSymbol != nil {
			h.clientSupportsHierarchicalDocumentSymbols = documentSymbol.HierarchicalDocumentSymbolSupport
		}
	}

	return &protocol.InitializeResult{
		Capabilities: &protocol.ServerCapabilities{
			PositionEncoding: protocol.PositionEncodingKindUTF16,
			TextDocumentSync: &protocol.TextDocumentSyncOptionsOrTextDocumentSyncKind{
				Value: &protocol.TextDocumentSyncOptions{
					OpenClose: true,
					Change:    protocol.TextDocumentSyncKindFull,
				},
			},
			DefinitionProvider: &protocol.BooleanOrDefinitionOptions{
				Value: protocol.Boolean(true),
			},
			ReferencesProvider: &protocol.BooleanOrReferenceOptions{
				Value: protocol.Boolean(true),
			},
			DocumentSymbolProvider: &protocol.BooleanOrDocumentSymbolOptions{
				Value: protocol.Boolean(true),
			},
			DocumentFormattingProvider: &protocol.BooleanOrDocumentFormattingOptions{
				Value: protocol.Boolean(true),
			},
			RenameProvider: &protocol.BooleanOrRenameOptions{
				Value: protocol.Boolean(true),
			},
		},
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    "loxls",
			Version: version,
		},
	}, nil
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#shutdown
func (h *Handler) shutdown() (any, error) {
	h.shuttingDown = true
	return nil, nil
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#exit
func (h *Handler) exit() error {
	code := 0
	if !h.shuttingDown {
		code = 1
	}
	os.Exit(code)
	return nil
}
