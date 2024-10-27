// Entry point for the Lox language server.
package main

import (
	"log/slog"
	"os"

	"github.com/marcuscaisey/lox/loxls/jsonrpc"
	"github.com/marcuscaisey/lox/loxls/lsp"
)

func main() {
	handler := slog.NewTextHandler(os.Stderr, nil)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("Starting lox language server. Reading from stdin, writing to stdout.")

	if err := jsonrpc.Serve(os.Stdin, os.Stdout, lsp.NewHandler()); err != nil {
		slog.Error("Something went wrong", "error", err.Error())
		os.Exit(1)
	}
}
