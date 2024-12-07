// Package protocol contains the types required to implement handlers for the LSP methods that loxls supports.
package protocol

//go:generate go run ./typegen
//typegen:method initialize
//typegen:method initialized
//typegen:method shutdown
//typegen:method exit
//typegen:method textDocument/didOpen
//typegen:method textDocument/didChange
//typegen:method textDocument/didClose
//typegen:method textDocument/documentSymbol
//typegen:method textDocument/publishDiagnostics
//typegen:method textDocument/formatting
//typegen:method window/logMessage
