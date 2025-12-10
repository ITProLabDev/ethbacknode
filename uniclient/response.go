package uniclient

import (
	"encoding/json"
)

func NewResponse() *Response {
	response := &Response{
		JsonRpc: JSON_RPC_VERSION_2_0,
	}
	return response
}

type Response struct {
	Id      RequestId       `json:"id"`
	JsonRpc string          `json:"jsonrpc"`
	Error   *RpcError       `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

func (r *Response) HasError() bool {
	return r.Error != nil
}

type RpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RpcError) Error() string {
	return e.Message
}

func (r *Response) ParseResult(params interface{}) (err error) {
	return json.Unmarshal(r.Result, params)
}
