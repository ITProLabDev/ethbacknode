package uniclient

import (
	"encoding/json"
)

// NewResponse creates a new JSON-RPC 2.0 response.
func NewResponse() *Response {
	response := &Response{
		JsonRpc: JSON_RPC_VERSION_2_0,
	}
	return response
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	Id      RequestId       `json:"id"`
	JsonRpc string          `json:"jsonrpc"`
	Error   *RpcError       `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

// HasError returns true if the response contains an error.
func (r *Response) HasError() bool {
	return r.Error != nil
}

// RpcError represents a JSON-RPC 2.0 error object.
type RpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface for RpcError.
func (e *RpcError) Error() string {
	return e.Message
}

// ParseResult unmarshals the result field into the given struct.
func (r *Response) ParseResult(params interface{}) (err error) {
	return json.Unmarshal(r.Result, params)
}
