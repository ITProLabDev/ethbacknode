package endpoint

type Router interface {
	Handle(method string, processor Processor)
}

type Processor interface {
	Process(ctx RequestContext, request RpcRequest, result RpcResponse) (err error)
}

type RequestContext interface {
	GetString(key string) (value string, err error)
	GetInt(key string) (value int64, err error)
	GetBool(key string) (value bool, err error)
	GetApiToken() (token string, err error)
	SetBool(key string, value bool)
	SetString(key string, value string)
	SetInt(key string, value int64)
	Authorized(bool)
}

type RpcRequest interface {
	GetMethod() (method RpcMethod)
	ParseParams(params interface{}) (err error)
	GetParamString(key string) (value string, err error)
	GetParamInt(key string) (value int64, err error)
	GetParamBool(key string) (value bool, err error)
}

type RpcResponse interface {
	SetResult(result interface{})
	SetError(code int, message string)
	SetErrorWithData(code int, message, data string)
}
