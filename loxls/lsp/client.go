package lsp

import (
	"github.com/marcuscaisey/lox/loxls/jsonrpc"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

type client struct {
	jsonrpcClient *jsonrpc.Client
}

func newClient(jsonrpcClient *jsonrpc.Client) *client {
	return &client{
		jsonrpcClient: jsonrpcClient,
	}
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_publishDiagnostics
func (c *client) TextDocumentPublishDiagnostics(params *protocol.PublishDiagnosticsParams) error {
	return c.jsonrpcClient.Notify("textDocument/publishDiagnostics", params)
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#window_logMessage
func (c *client) WindowLogMessage(params *protocol.LogMessageParams) error {
	return c.jsonrpcClient.Notify("window/logMessage", params)
}
