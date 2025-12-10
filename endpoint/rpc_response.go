package endpoint

import "encoding/json"

func NewResponse() *JsonRpcResponse {
	response := &JsonRpcResponse{
		JsonRpc: JSON_RPC_VERSION,
	}
	return response
}

type JsonRpcResponse struct {
	Id      RequestId       `json:"id"`
	JsonRpc string          `json:"jsonrpc"`
	Error   *JsonRpcError   `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

type JsonRpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (r *JsonRpcResponse) SetResultBytes(result []byte) {
	r.Result = result
	return
}

func (r *JsonRpcResponse) SetResult(result interface{}) {
	rawMessage, _ := json.Marshal(result)
	r.Result = rawMessage
	return
}

func (r *JsonRpcResponse) SetError(code int, message string) {
	if r.Error == nil {
		r.Error = new(JsonRpcError)
	}
	r.Error.Code, r.Error.Message = code, message
}
