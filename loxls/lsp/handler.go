// Package lsp implements a JSON-RPC 2.0 server handler which implements the Language Server Protocol.
package lsp

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/marcuscaisey/lox/loxls/jsonrpc"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

// Handler handles JSON-RPC requests and notifications.
type Handler struct {
	initialized  bool
	shuttingDown bool
	client       *client
	log          *logger
}

// NewHandler returns a new Handler.
func NewHandler() *Handler {
	return &Handler{}
}

// HandleRequest responds to a JSON-RPC request.
func (h *Handler) HandleRequest(method string, jsonParams *json.RawMessage) (any, error) {
	if !h.initialized && method != "initialize" {
		return nil, jsonrpc.NewError(jsonrpc.ErrorCode(protocol.ErrorCodesServerNotInitialized), "Server not initialized", nil)
	}
	if h.shuttingDown {
		return nil, jsonrpc.NewInvalidRequestError("Server shutting down")
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
func (h *Handler) HandleNotification(method string, jsonParams *json.RawMessage) {
	if err := h.handleNotification(method, jsonParams); err != nil {
		h.log.Error(err.Error())
	}
}

func (h *Handler) handleNotification(method string, jsonParams *json.RawMessage) error {
	if !h.initialized && method != "initialized" && method != "exit" {
		return fmt.Errorf("%s notification received before server initialized", method)
	}
	if h.shuttingDown && method != "exit" {
		return fmt.Errorf("%s notification received whilst server shutting down", method)
	}
	switch method {
	case "initialized":
		// No further initialisation needed
	case "textDocument/didOpen":
		var params *protocol.DidOpenTextDocumentParams
		if err := json.Unmarshal(*jsonParams, &params); err != nil {
			return fmt.Errorf("%s: %s", method, err)
		}
		return h.textDocumentDidOpen(params)
	case "textDocument/didChange":
		var params *protocol.DidChangeTextDocumentParams
		if err := json.Unmarshal(*jsonParams, &params); err != nil {
			return fmt.Errorf("%s: %s", method, err)
		}
		return h.textDocumentDidChange(params)
	case "textDocument/didClose":
		var params *protocol.DidCloseTextDocumentParams
		if err := json.Unmarshal(*jsonParams, &params); err != nil {
			return fmt.Errorf("%s: %s", method, err)
		}
		return h.textDocumentDidClose(params)
	case "exit":
		return h.exit()
	default:
		return fmt.Errorf("%s method not found", method)
	}
	return nil
}

// SetClient sets the client that the handler can use to send requests and notifications to the server's client.
func (h *Handler) SetClient(client *jsonrpc.Client) {
	h.client = newClient(client)
	h.log = newLogger(h.client)
}

type logger struct {
	client *client
}

func newLogger(client *client) *logger {
	return &logger{
		client: client,
	}
}

func (l *logger) Error(a ...any) {
	l.log(protocol.MessageTypeError, fmt.Sprint(a...))
}

func (l *logger) Errorf(format string, a ...any) {
	l.log(protocol.MessageTypeError, fmt.Sprintf(format, a...))
}

func (l *logger) log(typ protocol.MessageType, msg string) {
	err := l.client.WindowLogMessage(&protocol.LogMessageParams{
		Type:    typ,
		Message: protocol.String(msg),
	})
	if err != nil {
		slog.Warn("Failed to log", "error", err)
	}
}

func ptrTo[T any](v T) *T {
	return &v
}
