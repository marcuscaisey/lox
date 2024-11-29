// Package lsp implements a JSON-RPC 2.0 server handler which implements the Language Server Protocol.
package lsp

import (
	"encoding/json"

	"github.com/marcuscaisey/lox/loxls/jsonrpc"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

const version = "0.0.1"

// Handler responds to JSON-RPC requests and notifications.
type Handler struct {
	initialized bool
}

// NewHandler returns a new Handler.
func NewHandler() *Handler {
	return &Handler{}
}

var serverNotInitializedErr = jsonrpc.NewError(jsonrpc.ErrorCode(protocol.ErrorCodesServerNotInitialized), "Server not initialized", nil)

// HandleRequest responds to a JSON-RPC request.
func (h *Handler) HandleRequest(method string, params *json.RawMessage) (any, error) {
	if !h.initialized && method != "initialize" {
		return nil, serverNotInitializedErr
	}
	switch method {
	case "initialize":
		var initializeParams *protocol.InitializeParams
		if err := json.Unmarshal(*params, &initializeParams); err != nil {
			return nil, jsonrpc.NewError(jsonrpc.InvalidParams, "Invalid params", map[string]any{"error": err.Error()})
		}
		return h.initialize(initializeParams)
	default:
		return nil, jsonrpc.NewMethodNotFoundError(method)
	}
}

// HandleNotification responds to a JSON-RPC notification.
func (h *Handler) HandleNotification(method string, _ *json.RawMessage) error {
	if !h.initialized && method != "initialized" && method != "exit" {
		return serverNotInitializedErr
	}
	switch method {
	case "initialized":
		// No further initialisation needed
		return nil
	default:
		return jsonrpc.NewMethodNotFoundError(method)
	}
}

func (h *Handler) initialize(*protocol.InitializeParams) (*protocol.InitializeResult, error) {
	h.initialized = true
	// TODO: populate this result properly
	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{},
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    "loxls",
			Version: pointerTo(protocol.String(version)),
		},
	}, nil
}

func pointerTo[T any](v T) *T {
	return &v
}
