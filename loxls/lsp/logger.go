package lsp

import (
	"fmt"
	"log/slog"

	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

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

func (l *logger) Warning(a ...any) {
	l.log(protocol.MessageTypeWarning, fmt.Sprint(a...))
}

func (l *logger) Warningf(format string, a ...any) {
	l.log(protocol.MessageTypeWarning, fmt.Sprintf(format, a...))
}

func (l *logger) Info(a ...any) {
	l.log(protocol.MessageTypeInfo, fmt.Sprint(a...))
}

func (l *logger) Infof(format string, a ...any) {
	l.log(protocol.MessageTypeInfo, fmt.Sprintf(format, a...))
}

func (l *logger) Log(a ...any) {
	l.log(protocol.MessageTypeLog, fmt.Sprint(a...))
}

func (l *logger) Logf(format string, a ...any) {
	l.log(protocol.MessageTypeLog, fmt.Sprintf(format, a...))
}

func (l *logger) log(typ protocol.MessageType, msg string) {
	err := l.client.WindowLogMessage(&protocol.LogMessageParams{
		Type:    typ,
		Message: msg,
	})
	if err != nil {
		slog.Warn("Failed to log", "error", err)
	}
}
