// Package lsp implements a JSON-RPC 2.0 server handler which implements the Language Server Protocol.
package lsp

import (
	"encoding/json"

	"github.com/marcuscaisey/lox/loxls/jsonrpc"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

// Handler responds to JSON-RPC requests and notifications.
type Handler struct {
	initialized  bool
	shuttingDown bool
}

// NewHandler returns a new Handler.
func NewHandler() *Handler {
	return &Handler{}
}

var (
	notInitializedErr = jsonrpc.NewError(jsonrpc.ErrorCode(protocol.ErrorCodesServerNotInitialized), "Server not initialized", nil)
	shuttingDownErr   = jsonrpc.NewInvalidRequestError("Server shutting down")
)

// HandleRequest responds to a JSON-RPC request.
func (h *Handler) HandleRequest(method string, jsonParams *json.RawMessage) (any, error) {
	if !h.initialized && method != "initialize" {
		return nil, notInitializedErr
	}
	if h.shuttingDown {
		return nil, shuttingDownErr
	}
	switch method {
	case "initialize":
		var params *protocol.InitializeParams
		if err := json.Unmarshal(*jsonParams, &params); err != nil {
			return nil, jsonrpc.NewError(jsonrpc.InvalidParams, "Invalid params", map[string]any{"error": err.Error()})
		}
		return h.initialize(params)
	case "shutdown":
		return h.shutdown()
	default:
		return nil, jsonrpc.NewMethodNotFoundError(method)
	}
}

// HandleNotification responds to a JSON-RPC notification.
func (h *Handler) HandleNotification(method string, jsonParams *json.RawMessage) error {
	if !h.initialized && method != "initialized" && method != "exit" {
		return notInitializedErr
	}
	if h.shuttingDown && method != "exit" {
		return shuttingDownErr
	}
	switch method {
	case "initialized":
		// No further initialisation needed
		return nil
	case "textDocument/didOpen":
		var params *protocol.DidOpenTextDocumentParams
		if err := json.Unmarshal(*jsonParams, &params); err != nil {
			return jsonrpc.NewError(jsonrpc.InvalidParams, "Invalid params", map[string]any{"error": err.Error()})
		}
		return h.textDocumentDidOpen(params)
	case "textDocument/didChange":
		var params *protocol.DidChangeTextDocumentParams
		if err := json.Unmarshal(*jsonParams, &params); err != nil {
			return jsonrpc.NewError(jsonrpc.InvalidParams, "Invalid params", map[string]any{"error": err.Error()})
		}
		return h.textDocumentDidChange(params)
	case "textDocument/didClose":
		var params *protocol.DidCloseTextDocumentParams
		if err := json.Unmarshal(*jsonParams, &params); err != nil {
			return jsonrpc.NewError(jsonrpc.InvalidParams, "Invalid params", map[string]any{"error": err.Error()})
		}
		return h.textDocumentDidClose(params)
	case "exit":
		return h.exit()
	default:
		return jsonrpc.NewMethodNotFoundError(method)
	}
}

func ptrTo[T any](v T) *T {
	return &v
}
