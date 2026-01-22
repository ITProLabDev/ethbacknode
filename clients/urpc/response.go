package urpc

import "encoding/json"

// NewResponse creates a new JSON-RPC 2.0 response.
func NewResponse() *Response {
	response := new(Response)
	response.JsonRpc = JSON_RPC_VERSION_2_0
	return response
}

// Error represents a JSON-RPC 2.0 error object.
type Error struct {
	Code    int    `json:"code"`    // Error code
	Message string `json:"message"` // Error message
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	Id      RequestId       `json:"id"`               // Request ID (echoed back)
	JsonRpc string          `json:"jsonrpc"`          // JSON-RPC version
	Error   *Error          `json:"error,omitempty"`  // Error object (nil on success)
	Result  json.RawMessage `json:"result,omitempty"` // Result data (raw JSON)
}

// String returns a pretty-printed JSON representation of the response.
func (r *Response) String() string {
	b, _ := json.MarshalIndent(r, "", " ")
	return string(b)
}

// ParseResult unmarshals the result into the target object.
func (r *Response) ParseResult(target interface{}) (err error) {
	err = json.Unmarshal(r.Result, target)
	if err != nil {
		target = nil
		return err
	}
	return nil
}

// IsSuccess returns true if the response contains no error.
func (r *Response) IsSuccess() bool {
	return r.Error == nil
}

// ParseError converts the response error to a Go error.
// Returns nil if there is no error.
func (r *Response) ParseError() error {
	if r.Error == nil {
		return nil
	}
	return &WarpedError{Code: r.Error.Code, Message: r.Error.Message}
}

// unjson deserializes JSON data into the response.
func (r *Response) unjson(data []byte) (err error) {
	return json.Unmarshal(data, r)
}
