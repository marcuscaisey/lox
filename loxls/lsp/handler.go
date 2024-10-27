// Package lsp implements a JSON-RPC 2.0 server handler which implements the Language Server Protocol.
package lsp

import (
	"encoding/json"

	"github.com/marcuscaisey/lox/loxls/jsonrpc"
)

// Handler responds to JSON-RPC requests and notifications.
type Handler struct{}

// NewHandler returns a new Handler.
func NewHandler() *Handler {
	return &Handler{}
}

// HandleRequest responds to a JSON-RPC request.
func (h *Handler) HandleRequest(method string, _ *json.RawMessage) (any, error) {
	return nil, jsonrpc.NewMethodNotFoundError(method)
}

// HandleNotification responds to a JSON-RPC notification.
func (h *Handler) HandleNotification(method string, _ *json.RawMessage) error {
	return jsonrpc.NewMethodNotFoundError(method)
}
