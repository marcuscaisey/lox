package jsonrpc

// ErrorCode is a number indicating the error type that occurred.
type ErrorCode int

// JSON-RPC error codes
//
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#errorCodes
const (
	ParseError     ErrorCode = -32700
	InvalidRequest ErrorCode = -32600
	MethodNotFound ErrorCode = -32601
	InvalidParams  ErrorCode = -32602
	InternalError  ErrorCode = -32603
)

// NewError returns an error which can be encoded as a JSON-RPC error response.
func NewError(code ErrorCode, message string, data any) error {
	return &responseError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// NewMethodNotFoundError returns an error indicating that the requested method was not found.
func NewMethodNotFoundError(method string) error {
	return NewError(MethodNotFound, "Method not found", map[string]string{"method": method})
}

func newParseError(error string) error {
	return newErrorWithErrorField(ParseError, "Parse error", error)
}

func newInvalidRequestError(error string) error {
	return newErrorWithErrorField(InvalidRequest, "Invalid Request", error)
}

func newInternalError(error string) *responseError {
	return newErrorWithErrorField(InternalError, "Internal error", error).(*responseError)
}

func newErrorWithErrorField(code ErrorCode, message, error string) error {
	return NewError(code, message, map[string]string{"error": error})
}
