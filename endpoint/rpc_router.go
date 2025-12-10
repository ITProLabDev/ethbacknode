package endpoint

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/valyala/fasthttp"
	"module github.com/ITProLabDev/ethbacknode/tools/log"
	"runtime/debug"
	"strings"
)

func (r *BackRpc) AddRpcProcessor(method RpcMethod, processor RpcProcessor) {
	_, found := r.rpcProcessors[method]
	if found {
		log.Error("Processor for method", method, "already exists")
		panic(errors.New("processor for method " + string(method) + " already exists"))
	}
	r.rpcProcessors[method] = processor
}

func (r *BackRpc) RouteRpcRequest(ctx *fasthttp.RequestCtx) (err error) {
	rpcRequestContext := NewRpcRequestContext()

	rpcRequestContext.stringParams["remoteAddr"] = ctx.RemoteAddr().String()
	rpcRequestContext.stringParams["method"] = string(ctx.Method())
	rpcRequestContext.stringParams["path"] = string(ctx.Path())
	rpcRequestContext.stringParams["uri"] = string(ctx.URI().RequestURI())

	token, err := r.parseApiToken(ctx)
	if err == nil {
		rpcRequestContext.apiToken = token
	}
	if r.debugMode {
		log.Warning("Route rpc request:", string(ctx.Method()), string(ctx.Path()))
	}
	if strings.HasPrefix(string(ctx.Path()), "/api/v1/") {
		log.Warning("Process try RESTful request")
	} else if strings.HasPrefix(string(ctx.Path()), "/rpc") {
		if string(ctx.Method()) != fasthttp.MethodPost {
			return errMethodNotAllowed
		}
		return r.processRpcRequest(ctx, rpcRequestContext)
	} else if strings.HasPrefix(string(ctx.Path()), "/ws") {
		log.Warning("TODO Process try WS connection")
	} else if strings.HasPrefix(string(ctx.Path()), "/docs") {
		//TODO Show API docs
	} else {
		log.Warning("Process try unknown request")
		return errInternalRouteNotFound
	}
	return errInternalRouteNotFound
}

func (r *BackRpc) parseApiToken(ctx *fasthttp.RequestCtx) (token string, err error) {
	token = string(ctx.Request.Header.Peek("X-Api-Token"))
	if token == "" {
		return "", errApiTokenNotProvided
	}
	return token, nil
}

func (r *BackRpc) processRpcRequest(ctx *fasthttp.RequestCtx, rpcRequestContext *RpcRequestContext) (err error) {
	rpcRequest := new(JsonRpcRequest)
	rpcResponse := NewResponse()
	defer func() {
		rc := recover()
		if rc == nil {
			return
		}
		log.Error("Recovered from panic:", rc)
		if r.debugMode {
			debug.PrintStack()
		}
		rpcResponse.SetError(ERROR_CODE_SERVER_ERROR, ERROR_MESSAGE_SERVER_ERROR)
		_ = json.NewEncoder(ctx.Response.BodyWriter()).Encode(rpcResponse)
	}()

	body := ctx.Request.Body()
	if r.debugMode {
		log.Warning(string(body))
	}
	err = json.NewDecoder(bytes.NewBuffer(body)).Decode(&rpcRequest)
	if err != nil {
		return errParseError
	}
	method := rpcRequest.Method
	if method == "" {
		rpcResponse.SetError(ERROR_CODE_INVALID_REQUEST, ERROR_MESSAGE_INVALID_REQUEST)
		return json.NewEncoder(ctx.Response.BodyWriter()).Encode(rpcResponse)
	}
	if r.debugMode {
		log.Warning("Process rpc request:", method)
	}
	rpcResponse.Id = rpcRequest.Id
	processor, ok := r.rpcProcessors[method]
	if !ok {
		rpcResponse.SetError(ERROR_CODE_METHOD_NOT_FOUND, ERROR_MESSAGE_METHOD_NOT_FOUND)
		return json.NewEncoder(ctx.Response.BodyWriter()).Encode(rpcResponse)
	}
	processor(rpcRequestContext, rpcRequest, rpcResponse)
	rb := json.NewEncoder(ctx.Response.BodyWriter()).Encode(rpcResponse)
	return rb
}
