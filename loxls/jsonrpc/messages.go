package jsonrpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
)

const validJSONRPC = "2.0"

type message interface {
	isMessage()
}

// request is a request message to describe a request between the client and the server. Every processed request must
// send a response back to the sender of the request.
//
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#requestMessage
type request struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      intOrStr         `json:"id"`               // The request id.
	Method  string           `json:"method"`           // The method to be invoked.
	Params  *json.RawMessage `json:"params,omitempty"` // The method's params.
}

func (r *request) isMessage() {}

// notification is a notification message. A processed notification message must not send a response back. They
// work like events.
//
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#notificationMessage
type notification struct {
	JSONRPC string           `json:"jsonrpc"`
	Method  string           `json:"method"`           // The method to be invoked.
	Params  *json.RawMessage `json:"params,omitempty"` // The notification's params.
}

func (n *notification) isMessage() {}

// response is a response Message sent as a result of a request. If a request doesn’t provide a result value the
// receiver of a request still needs to return a response message to conform to the JSON-RPC specification. The result
// property of the response should be set to null in this case to signal a successful request.
//
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#responseMessage
type response struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      *intOrStr `json:"id"` // The request id.
	// The result of a request. This member is REQUIRED on success.
	// This member MUST NOT exist if there was an error invoking the method.
	Result *json.RawMessage `json:"result,omitempty"`
	Error  *responseError   `json:"error,omitempty"` // The error object in case a request fails.
}

func (r *response) isMessage() {}

// responseError is an error object in case a request fails.
//
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#responseError
type responseError struct {
	Code    ErrorCode `json:"code"`    // A number indicating the error type that occurred.
	Message string    `json:"message"` // A string providing a short description of the error.
	// A primitive or structured value that contains additional information about the error. Can be omitted.
	Data any `json:"data,omitempty"`
}

func (e *responseError) Error() string {
	return fmt.Sprintf("jsonrpc error: code = %d message = %q data = %v", e.Code, e.Message, e.Data)
}

type combinedMessage struct {
	JSONRPC optional[string]           `json:"jsonrpc"`
	ID      nullOptional[*intOrStr]    `json:"id"`
	Method  optional[string]           `json:"method"`
	Params  optional[*json.RawMessage] `json:"params"`
	Result  optional[*json.RawMessage] `json:"result"`
	Error   optional[*responseError]   `json:"error"`
}

func unmarshalMessage(content []byte) (message, error) {
	var combinedMsg combinedMessage
	if err := json.Unmarshal(content, &combinedMsg); err != nil {
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			return nil, newParseError(err.Error())
		}
		return nil, newInvalidRequestError(err.Error())
	}

	if !combinedMsg.JSONRPC.IsPresent() {
		return nil, newInvalidRequestError("jsonrpc is required")
	}
	jsonrpc := combinedMsg.JSONRPC.Get()
	if jsonrpc != validJSONRPC {
		return nil, newInvalidRequestError(fmt.Sprintf("invalid jsonrpc value %q, must be %q", jsonrpc, validJSONRPC))
	}

	// We read requests (including notifications) and responses from the same stream. If we can't determine which one it
	// is, we assume it's a request as this is more likely.
	if combinedMsg.ID.IsPresent() && (combinedMsg.Result.IsPresent() || combinedMsg.Error.IsPresent()) &&
		!combinedMsg.Method.IsPresent() && !combinedMsg.Params.IsPresent() {

		resp := &response{JSONRPC: jsonrpc, ID: combinedMsg.ID.Get()}
		if combinedMsg.Result.IsPresent() && combinedMsg.Error.IsPresent() {
			return nil, errors.New("unmarshalling response: result and error are mutually exclusive")
		}
		if combinedMsg.Result.IsPresent() {
			resp.Result = combinedMsg.Result.Get()
		} else {
			resp.Error = combinedMsg.Error.Get()
		}
		return resp, nil
	}

	// Message is a request or notification
	if !combinedMsg.Method.IsPresent() {
		return nil, newInvalidRequestError("method is required")
	}
	method := combinedMsg.Method.Get()
	var params *json.RawMessage
	if combinedMsg.Params.IsPresent() {
		params = combinedMsg.Params.Get()
	}
	var msg message
	if combinedMsg.ID.IsPresent() {
		if combinedMsg.ID.IsNull() {
			return nil, newInvalidRequestError("id cannot be null")
		}
		msg = &request{JSONRPC: jsonrpc, ID: *combinedMsg.ID.Get(), Method: method, Params: params}
	} else {
		msg = &notification{JSONRPC: jsonrpc, Method: method, Params: params}
	}
	if combinedMsg.Result.IsPresent() {
		return nil, newInvalidRequestError("result is not a valid request field")
	}
	if combinedMsg.Error.IsPresent() {
		return nil, newInvalidRequestError("error is not a valid request field")
	}
	return msg, nil
}

// intOrStr is either an integer or a string
type intOrStr struct {
	int   int
	str   string
	isInt bool
}

func (is *intOrStr) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		return nil
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch v := v.(type) {
	case string:
		is.str = v
	case float64:
		if math.Trunc(v) != v {
			return &json.UnmarshalTypeError{
				Value: fmt.Sprint(v),
				Type:  reflect.TypeOf(intOrStr{}),
			}
		}
		is.isInt = true
		is.int = int(v)
	default:
		return &json.UnmarshalTypeError{
			Value: fmt.Sprint(v),
			Type:  reflect.TypeOf(intOrStr{}),
		}
	}
	return nil
}

func (is intOrStr) MarshalJSON() ([]byte, error) {
	var v any = is.int
	if !is.isInt {
		v = is.str
	}
	return json.Marshal(v)
}

func (is intOrStr) String() string {
	if is.isInt {
		return fmt.Sprint(is.int)
	} else {
		return is.str
	}
}

// optional is a JSON value which can be present and non-null, or not present.
type optional[T any] []T

func (o optional[T]) IsPresent() bool {
	return o != nil
}

func (o optional[T]) Get() T {
	if !o.IsPresent() {
		panic("get of an absent value")
	}
	return o[0]
}

func (o *optional[T]) UnmarshalJSON(data []byte) error {
	*o = optional[T]{}
	if bytes.Equal(data, []byte("null")) {
		return &json.UnmarshalTypeError{
			Value: "null",
			Type:  reflect.TypeOf(new(T)).Elem(),
		}
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = append(*o, v)
	return nil
}

// nullOptional is a JSON value that can be present and non-null, present and null, or not present.
type nullOptional[T any] []*T

func (o nullOptional[T]) IsPresent() bool {
	return o != nil
}

func (o nullOptional[T]) IsNull() bool {
	return o.IsPresent() && o[0] == nil
}

func (o nullOptional[T]) Get() T {
	if !o.IsPresent() {
		panic("get of an absent value")
	}
	if o.IsNull() {
		panic("get of a null value")
	}
	return *o[0]
}

func (o *nullOptional[T]) UnmarshalJSON(data []byte) error {
	*o = nullOptional[T]{}
	if bytes.Equal(data, []byte("null")) {
		*o = append(*o, nil)
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = append(*o, &v)
	return nil
}
