package endpoint

// Router defines the interface for registering RPC method handlers.
type Router interface {
	Handle(method string, processor Processor)
}

// Processor defines the interface for RPC request processing.
type Processor interface {
	Process(ctx RequestContext, request RpcRequest, result RpcResponse) (err error)
}

// RequestContext provides access to request-scoped data and authentication.
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

// RpcRequest defines the interface for accessing JSON-RPC request data.
type RpcRequest interface {
	GetMethod() (method RpcMethod)
	ParseParams(params interface{}) (err error)
	GetParamString(key string) (value string, err error)
	GetParamInt(key string) (value int64, err error)
	GetParamBool(key string) (value bool, err error)
}

// RpcResponse defines the interface for building JSON-RPC responses.
type RpcResponse interface {
	SetResult(result interface{})
	SetError(code int, message string)
	SetErrorWithData(code int, message, data string)
}
