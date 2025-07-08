// Package lsp implements a JSON-RPC 2.0 server handler which implements the Language Server Protocol.
package lsp

import (
	"encoding/json"
	"fmt"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/loxls/jsonrpc"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

// Handler handles JSON-RPC requests and notifications.
type Handler struct {
	// Dependencies
	client *client
	log    *logger

	// Internal state
	initialized          bool
	shuttingDown         bool
	stubBuiltinsFilename string
	stubBuiltins         []ast.Decl
	docs                 map[string]*document
	capabilities         *protocol.ClientCapabilities
}

// NewHandler returns a new Handler.
func NewHandler() *Handler {
	return &Handler{
		docs: map[string]*document{},
	}
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
		return handleRequest(h.initialize, jsonParams)
	case "shutdown":
		return h.shutdown()
	case "textDocument/definition":
		return handleRequest(h.textDocumentDefinition, jsonParams)
	case "textDocument/references":
		return handleRequest(h.textDocumentReferences, jsonParams)
	case "textDocument/hover":
		return handleRequest(h.textDocumentHover, jsonParams)
	case "textDocument/documentSymbol":
		return handleRequest(h.textDocumentDocumentSymbol, jsonParams)
	case "textDocument/completion":
		return handleRequest(h.textDocumentCompletion, jsonParams)
	case "textDocument/formatting":
		return handleRequest(h.textDocumentFormatting, jsonParams)
	case "textDocument/rename":
		return handleRequest(h.textDocumentRename, jsonParams)
	default:
		return nil, jsonrpc.NewMethodNotFoundError(method)
	}
}

type requestHandler[T any, R any] func(T) (R, error)

func handleRequest[T any, R any](handler requestHandler[T, R], jsonParams *json.RawMessage) (any, error) {
	var params T
	if err := json.Unmarshal(*jsonParams, &params); err != nil {
		return nil, jsonrpc.NewError(jsonrpc.InvalidParams, "Invalid params", map[string]any{"error": err.Error()})
	}
	return handler(params)
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
	case "exit":
		return h.exit()
	case "textDocument/didOpen":
		return handleNotification(method, h.textDocumentDidOpen, jsonParams)
	case "textDocument/didChange":
		return handleNotification(method, h.textDocumentDidChange, jsonParams)
	case "textDocument/didClose":
		return handleNotification(method, h.textDocumentDidClose, jsonParams)
	default:
		return fmt.Errorf("%s method not found", method)
	}
	return nil
}

type notificationHandler[T any] func(T) error

func handleNotification[T any](method string, handler notificationHandler[T], jsonParams *json.RawMessage) error {
	var params T
	if err := json.Unmarshal(*jsonParams, &params); err != nil {
		return fmt.Errorf("%s: %s", method, err)
	}
	return handler(params)
}

// SetClient sets the client that the handler can use to send requests and notifications to the server's client.
func (h *Handler) SetClient(client *jsonrpc.Client) {
	h.client = newClient(client)
	h.log = newLogger(h.client)
}
