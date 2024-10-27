// Package jsonrpc provides a JSON-RPC 2.0 server implementation for the version of the protocol defined at
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#baseProtocol.
package jsonrpc

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"strconv"
	"strings"
)

// Handler responds to JSON-RPC requests and notifications.
type Handler interface {
	HandleRequest(method string, params *json.RawMessage) (any, error)
	HandleNotification(method string, params *json.RawMessage) error
}

// Serve reads JSON-RPC messages from in, passes them to handler, and writes the responses to out.
func Serve(in io.Reader, out io.Writer, handler Handler) error {
	server := newServer(in, out, handler)
	return server.Serve()
}

type server struct {
	in      *bufio.Reader
	out     io.Writer
	handler Handler
}

func newServer(in io.Reader, out io.Writer, handler Handler) *server {
	return &server{
		in:      bufio.NewReader(in),
		out:     out,
		handler: handler,
	}
}

func (s *server) Serve() error {
	for {
		msg, err := s.read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				slog.Info("EOF reached, stopping server")
				return nil
			}
			var respErr *responseError
			if errors.As(err, &respErr) {
				resp := &response{JSONRPC: validJSONRPC, ID: nil, Error: respErr}
				if writeErr := s.write(resp); writeErr != nil {
					return fmt.Errorf("serving jsonrpc requests: %v", writeErr)
				}
				continue
			}
			return fmt.Errorf("serving jsonrpc requests: %v", err)
		}

		if err := s.handle(msg); err != nil {
			return fmt.Errorf("serving jsonrpc requests: %v", err)
		}
	}
}

type headers struct {
	ContentLength int64
	ContentType   string
}

// reads a message according to
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#baseProtocol
func (s *server) read() (message, error) {
	headers, err := s.readHeaders()
	if err != nil {
		return nil, fmt.Errorf("reading message: %w", err)
	}

	content, err := io.ReadAll(io.LimitReader(s.in, headers.ContentLength))
	if err != nil {
		return nil, fmt.Errorf("reading message: reading content: %w", err)
	}

	msg, err := unmarshalMessage(content)
	if err != nil {
		return nil, fmt.Errorf("reading message: %w", err)
	}

	return msg, nil
}

const (
	contentLengthHeader = "Content-Length"
	contentTypeHeader   = "Content-Type"
	validMediaType      = "application/vscode-jsonrpc"
)

func (s *server) readHeaders() (*headers, error) {
	headers := &headers{}
	contentLengthPresent := false
	for {
		line, err := s.readHeaderLine()
		if err != nil {
			return nil, err
		}
		if line == "" {
			break
		}

		field, value, found := strings.Cut(line, ":")
		if !found {
			return nil, fmt.Errorf("header line does not contain colon: %q", line)
		}
		value = strings.TrimSpace(value)

		switch strings.ToLower(field) {
		case strings.ToLower(contentLengthHeader):
			contentLengthPresent = true
			n, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid %s header %q: %s", contentLengthHeader, value, err)
			}
			headers.ContentLength = n

		case strings.ToLower(contentTypeHeader):
			mediaType, params, err := mime.ParseMediaType(value)
			if err != nil {
				return nil, fmt.Errorf("invalid %s header %q: %s", contentTypeHeader, value, err)
			}
			if mediaType != validMediaType {
				return nil, fmt.Errorf("invalid %s header %q: only %s MIME type is supported", contentTypeHeader, value, validMediaType)
			}
			if charset, ok := params["charset"]; ok && charset != "utf-8" && charset != "utf8" {
				return nil, fmt.Errorf("invalid %s header %q: charset must be utf-8", contentTypeHeader, value)
			}
			headers.ContentType = value

		default:
			return nil, fmt.Errorf("unknown header: %q", line)
		}
	}

	if !contentLengthPresent {
		return nil, fmt.Errorf("missing %s header", contentLengthHeader)
	}

	return headers, nil
}

func (s *server) readHeaderLine() (string, error) {
	var b strings.Builder
	for {
		s, err := s.in.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("reading header line: %w", err)
		}
		b.WriteString(s)
		if len(s) >= 2 && s[len(s)-2] == '\r' {
			break
		}
	}
	return strings.TrimSuffix(b.String(), "\r\n"), nil
}

func (s *server) write(msg message) error {
	content, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("writing message: %w", err)
	}
	if _, err := fmt.Fprintf(s.out, "%s: %d\r\n\r\n%s", contentLengthHeader, len(content), content); err != nil {
		return fmt.Errorf("writing message: %w", err)
	}
	return nil
}

func (s *server) handle(msg message) error {
	switch msg := msg.(type) {
	case *request:
		result, err := s.handler.HandleRequest(msg.Method, msg.Params)
		resp := &response{JSONRPC: validJSONRPC, ID: &msg.ID}
		if err != nil {
			var respErr *responseError
			if errors.As(err, &respErr) {
				resp.Error = respErr
			} else {
				resp.Error = newInternalError(err.Error())
			}
		} else {
			resultBytes, err := json.Marshal(result)
			if err != nil {
				resp.Error = newInternalError(fmt.Sprintf("unable to marshal result: %v", err))
			} else {
				rawMsg := json.RawMessage(resultBytes)
				resp.Result = &rawMsg
			}
		}
		if writeErr := s.write(resp); writeErr != nil {
			return fmt.Errorf("handling message: %w", writeErr)
		}

	case *notification:
		if err := s.handler.HandleNotification(msg.Method, msg.Params); err != nil {
			slog.Error("Error handling notification", "error", err.Error())
		}

	case *response:
		var msgJSON string
		bytes, err := json.Marshal(msg)
		if err != nil {
			msgJSON = "unable to marshal message"
		} else {
			msgJSON = string(bytes)
		}
		slog.Info("Ignoring response message", "message", msgJSON)
	}

	return nil
}
