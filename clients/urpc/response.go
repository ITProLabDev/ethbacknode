package urpc

import "encoding/json"

func NewResponse() *Response {
	response := new(Response)
	response.JsonRpc = JSON_RPC_VERSION_2_0
	return response
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
type Response struct {
	Id      RequestId       `json:"id"`
	JsonRpc string          `json:"jsonrpc"`
	Error   *Error          `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

func (r *Response) String() string {
	b, _ := json.MarshalIndent(r, "", " ")
	return string(b)
}

func (r *Response) ParseResult(target interface{}) (err error) {
	err = json.Unmarshal(r.Result, target)
	if err != nil {
		target = nil
		return err
	}
	return nil
}

func (r *Response) IsSuccess() bool {
	return r.Error == nil
}

func (r *Response) ParseError() error {
	if r.Error == nil {
		return nil
	}
	return &WarpedError{Code: r.Error.Code, Message: r.Error.Message}
}

func (r *Response) unjson(data []byte) (err error) {
	return json.Unmarshal(data, r)
}
