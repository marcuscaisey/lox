// Package protocol contains the types required to implement handlers for the LSP methods that loxls supports.
package protocol

//go:generate go run ./typegen initialize initialized shutdown exit textDocument/didOpen textDocument/didChange textDocument/didClose window/logMessage
