package jsonrpc

import (
	"encoding/json"
	"fmt"
	"io"
)

// Client is a JSON-RPC client.
type Client struct {
	in     io.Reader
	out    io.Writer
	server *server
}

func newClient(in io.Reader, out io.Writer, server *server) *Client {
	return &Client{
		in:     in,
		out:    out,
		server: server,
	}
}

// Notify sends a notification to the server.
func (c *Client) Notify(method string, params any) error {
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("sending %q notification: marshalling parameters to JSON: %s", method, err)
	}
	notif := &notification{
		JSONRPC: validJSONRPC,
		Method:  method,
		Params:  ptrTo(json.RawMessage(data)),
	}
	if err := c.server.write(notif); err != nil {
		return fmt.Errorf("sending %q notification: %s", method, err)
	}
	return nil
}

func ptrTo[T any](v T) *T {
	return &v
}
