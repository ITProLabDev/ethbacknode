package endpoint

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type RpcMethod string

type RpcProcessor func(ctx RequestContext, request RpcRequest, response RpcResponse)

type RpcAuth struct {
	ClientID string `json:"clientId"`
	TS       int64  `json:"ts,omitempty"`
	Nonce    string `json:"nonce,omitempty"`
	Alg      string `json:"alg,omitempty"`
	Sig      string `json:"sig"`
}

type JsonRpcRequest struct {
	Id        RequestId       `json:"id"`
	JsonRpc   string          `json:"jsonrpc"`
	Method    RpcMethod       `json:"method"`
	Params    json.RawMessage `json:"params,omitempty"`
	paramsMap map[string]interface{}

	Auth *RpcAuth `json:"auth,omitempty"`
}

func (r *JsonRpcRequest) GetMethod() (method RpcMethod) {
	return r.Method
}

func (r *JsonRpcRequest) ParseParams(params interface{}) (err error) {
	return json.Unmarshal(r.Params, params)
}

func (r *JsonRpcRequest) GetParamString(key string) (param string, err error) {
	if r.paramsMap == nil {
		err = r.parseParamsMap()
		if err != nil {
			return "", err
		}
	}
	paramRaw, found := r.GetParamRaw(key)
	if !found {
		return "", ErrParamNotFound
	}
	switch paramRaw.(type) {
	case string:
		param = paramRaw.(string)
		return
	default:
		param = fmt.Sprintf("%v", paramRaw)
	}
	return
}

func (r *JsonRpcRequest) GetParamInt(key string) (param int64, err error) {
	if r.paramsMap == nil {
		err = r.parseParamsMap()
		if err != nil {
			return 0, err
		}
	}
	paramRaw, found := r.GetParamRaw(key)
	if !found {
		return 0, ErrParamNotFound
	}
	if paramRaw == nil {
		param = 0
		return
	}
	switch paramRaw.(type) {
	case int64:
		param = paramRaw.(int64)
		return
	case float64:
		param = int64(paramRaw.(float64))
		return
	case string:
		paramStr := paramRaw.(string)
		return r.parseStringToInt64(paramStr)
	case json.Number:
		paramStr := string(paramRaw.(json.Number))
		return r.parseStringToInt64(paramStr)
	default:
		return 0, ErrInvalidParamType
	}
	return
}

func (r *JsonRpcRequest) GetParamBool(key string) (value bool, err error) {
	if r.paramsMap == nil {
		err = r.parseParamsMap()
		if err != nil {
			return false, err
		}
	}
	paramRaw, found := r.GetParamRaw(key)
	if !found {
		return false, ErrParamNotFound
	}
	switch paramRaw.(type) {
	case bool:
		value = paramRaw.(bool)
		return
	default:
		switch paramRaw.(type) {
		case string:
			paramStr := paramRaw.(string)
			if paramStr == "true" || paramStr == "1" {
				value = true
			}
		case json.Number:
			paramStr := string(paramRaw.(json.Number))
			if paramStr == "true" || paramStr == "1" {
				value = true
			}
		case float64:
			paramNumeric := int(paramRaw.(float64))
			if paramNumeric > 0 {
				value = true
			}
		}
	}
	return
}

func (r *JsonRpcRequest) GetParamRaw(key string) (param interface{}, found bool) {
	if r.paramsMap == nil {
		r.parseParamsMap()
	}
	param, found = r.paramsMap[key]
	return
}

func (r *JsonRpcRequest) GetParamFloat64(key string) (param float64, err error) {
	if r.paramsMap == nil {
		err = r.parseParamsMap()
		if err != nil {
			return 0.0, err
		}
	}
	paramRaw, found := r.GetParamRaw(key)
	if !found {
		return 0, ErrParamNotFound
	}
	if paramRaw == nil {
		param = 0
		return
	}
	switch paramRaw.(type) {
	case float64:
		param = paramRaw.(float64)
		return
	case int64:
		param = float64(paramRaw.(int64))
		return
	case string:
		paramStr := paramRaw.(string)
		return r.parseStringToFloat64(paramStr)
	case json.Number:
		paramStr := string(paramRaw.(json.Number))
		return r.parseStringToFloat64(paramStr)
	default:
		return 0, ErrInvalidParamType
	}
	return
}

func (r *JsonRpcRequest) parseParamsMap() (err error) {
	return json.Unmarshal(r.Params, &r.paramsMap)
}

func (r *JsonRpcRequest) parseStringToInt64(param string) (num int64, err error) {
	return strconv.ParseInt(param, 0, 64)
}

func (r *JsonRpcRequest) parseStringToInt(param string) (num int, err error) {
	return strconv.Atoi(param)
}

func (r *JsonRpcRequest) parseStringToFloat64(param string) (num float64, err error) {
	return strconv.ParseFloat(param, 64)
}

type Number string
