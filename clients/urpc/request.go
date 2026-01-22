package urpc

import (
	"encoding/json"
	"fmt"
)

// Parameter type constants for Request serialization.
const (
	paramsTypeNamed = iota // Named parameters (object)
	paramsTypeArray        // Positional parameters (array)
	paramsTypeRaw          // Raw JSON parameters
)

// NewRequest creates a new JSON-RPC 2.0 request with positional parameters.
func NewRequest(method string, params ...interface{}) (req *Request) {
	req = &Request{
		Id:         "1",
		JsonRpc:    JSON_RPC_VERSION_2_0,
		Method:     method,
		paramsType: paramsTypeArray,
	}
	for _, p := range params {
		req.ParamsArray = append(req.ParamsArray, p)
	}
	return
}
// NewRequestWithNamedParams creates a new JSON-RPC 2.0 request with named parameters.
func NewRequestWithNamedParams(method string, params map[string]interface{}) (req *Request) {
	req = &Request{
		Id:          "1",
		JsonRpc:     JSON_RPC_VERSION_2_0,
		Method:      method,
		ParamsNamed: make(map[string]interface{}),
		paramsType:  paramsTypeNamed,
	}
	if params != nil {
		for k, v := range params {
			req.ParamsNamed[k] = v
		}
	}
	return
}

// NewRequestWithRawParams creates a new JSON-RPC 2.0 request with raw JSON parameters.
func NewRequestWithRawParams(method string, params json.RawMessage) (req *Request) {
	req = &Request{
		Id:         "1",
		JsonRpc:    JSON_RPC_VERSION_2_0,
		Method:     method,
		Params:     params,
		paramsType: paramsTypeRaw,
	}
	return
}

// NewRequestWithObject creates a new JSON-RPC 2.0 request with parameters from a struct.
// The params object is marshaled to JSON.
func NewRequestWithObject(method string, params interface{}) (req *Request) {
	paramsBytes, _ := json.Marshal(params)
	req = &Request{
		Id:         "1",
		JsonRpc:    JSON_RPC_VERSION_2_0,
		Method:     method,
		Params:     paramsBytes,
		paramsType: paramsTypeRaw,
	}
	return
}

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	Id           RequestId              `json:"id"`
	JsonRpc      string                 `json:"jsonrpc"`
	Method       string                 `json:"method"`
	ParamsNamed  map[string]interface{} `json:"paramsN,omitempty"`
	ParamsArray  []interface{}          `json:"paramsA,omitempty"`
	Params       json.RawMessage        `json:"params,omitempty"`
	ParamsObject interface{}            `json:"paramsO,omitempty"`
	paramsType   int
}

// SetId sets the request ID.
func (r *Request) SetId(id RequestId) {
	r.Id = id
}

// SetNamedParam sets a named parameter.
func (r *Request) SetNamedParam(key string, value interface{}) {
	r.ParamsNamed[key] = value
}

// AddParams appends positional parameters.
func (r *Request) AddParams(values ...interface{}) {
	r.ParamsArray = append(r.ParamsArray, values...)
}

// SetParams replaces all positional parameters.
func (r *Request) SetParams(values ...interface{}) {
	r.ParamsArray = values
}

// String returns a pretty-printed JSON representation of the request.
func (r *Request) String() string {
	b, _ := json.MarshalIndent(r, "", " ")
	return string(b)
}

// MarshalJSON implements custom JSON marshaling for JSON-RPC 2.0 format.
func (r *Request) MarshalJSON() ([]byte, error) {
	var params string
	switch r.paramsType {
	case paramsTypeArray:
		if len(r.ParamsArray) > 0 {
			paramsEncoded, err := json.Marshal(r.ParamsArray)
			if err != nil {
				return nil, err
			}
			params = string(paramsEncoded)
		} else {
			params = "[]"
		}
	case paramsTypeNamed:
		if len(r.ParamsNamed) > 0 {
			paramsEncoded, err := json.Marshal(r.ParamsNamed)
			if err != nil {
				return nil, err
			}
			params = string(paramsEncoded)
		} else {
			params = "null"
		}
	case paramsTypeRaw:
		params = string(r.Params)
	}
	return []byte(fmt.Sprintf(`{"jsonrpc":"%s","method":"%s","params":%s,"id":%s}`, r.JsonRpc, r.Method, params, r.Id)), nil
}

// RequestId is a JSON-RPC request identifier.
// Stored as string but marshaled as numeric for JSON-RPC compatibility.
type RequestId string

// String returns the string representation of the request ID.
func (id RequestId) String() string {
	return string(id)
}

// MarshalJSON serializes the ID as a number (JSON-RPC standard).
func (id RequestId) MarshalJSON() ([]byte, error) {
	if id == "" {
		return []byte("0"), nil
	}
	out := fmt.Sprintf("%s", id)
	return []byte(out), nil
}

// UnmarshalJSON deserializes a numeric ID, stripping quotes and leading zeros.
func (id *RequestId) UnmarshalJSON(data []byte) error {
	var dc []byte
	fc := true
	for _, b := range data {
		if b < 58 && b > 47 {
			if fc && b != 48 {
				fc = false
			}
			if !fc {
				dc = append(dc, b)
			}
		}
	}
	if len(dc) == 0 {
		*id = "0"
		return nil
	}
	*id = RequestId(dc)
	return nil
}
