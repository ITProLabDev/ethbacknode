package endpoint

import (
	"errors"
	"module github.com/ITProLabDev/ethbacknode/address"
	"module github.com/ITProLabDev/ethbacknode/subscriptions"
	"module github.com/ITProLabDev/ethbacknode/tools/log"
	"module github.com/ITProLabDev/ethbacknode/types"
	"module github.com/ITProLabDev/ethbacknode/watchdog"

	"github.com/valyala/fasthttp"
)

const (
	MIME_TYPE_JSON           = "application/json"
	ERROR_METHOD_NOT_ALLOWED = `{"error": {"code": -32601, "message": "method not found"}}`
)

type BackRpc struct {
	debugMode        bool
	addressPool      *address.Manager
	chainClient      types.ChainClient
	knownTokens      map[string]*types.TokenInfo
	subscriptions    *subscriptions.Manager
	watchdog         *watchdog.Service
	txCache          types.TxCache
	fallbackResponse HttpResponse
	addressCodec     address.AddressCodec
	rpcProcessors    map[RpcMethod]RpcProcessor
	burnAddress      string
}

type BackRpcOption func(r *BackRpc)

func WithDebugMode(debugMode bool) BackRpcOption {
	return func(r *BackRpc) {
		r.debugMode = debugMode
	}
}

func WithFallbackResponse(response HttpResponse) BackRpcOption {
	return func(r *BackRpc) {
		r.fallbackResponse = response
	}
}

func WithRpcProcessor(method RpcMethod, processor RpcProcessor) BackRpcOption {
	return func(r *BackRpc) {
		r.AddRpcProcessor(method, processor)
	}
}

func NewBackRpc(addressPool *address.Manager, chainClient types.ChainClient, subscriptions *subscriptions.Manager, watchdog *watchdog.Service, txcache types.TxCache, options ...BackRpcOption) *BackRpc {
	r := &BackRpc{
		addressPool:   addressPool,
		chainClient:   chainClient,
		knownTokens:   make(map[string]*types.TokenInfo),
		subscriptions: subscriptions,
		watchdog:      watchdog,
		txCache:       txcache,
		addressCodec:  chainClient.GetAddressCodec(),
		rpcProcessors: make(map[RpcMethod]RpcProcessor),
	}
	r.InitProcessors()
	for _, option := range options {
		option(r)
	}
	knownTokens := chainClient.TokensList()
	for _, token := range knownTokens {
		r.knownTokens[token.Symbol] = token
	}
	return r
}

func (r *BackRpc) Handle(ctx *fasthttp.RequestCtx) {
	if r.debugMode {
		log.Warning("Handle rpc request:", string(ctx.Method()), string(ctx.Path()))
	}
	//TODO check credentials
	//TODO check rate limit
	//TODO check request size
	err := r.RouteRpcRequest(ctx)
	if errors.Is(err, errInternalRouteNotFound) {
		if string(ctx.Method()) == fasthttp.MethodGet {
			if r.fallbackResponse == nil {
				ctx.SetContentType(MIME_TYPE_JSON)
				ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
				ctx.SetBodyString(ERROR_METHOD_NOT_ALLOWED)
			} else {
				ctx.SetContentType(r.fallbackResponse.ContentType())
				ctx.SetStatusCode(r.fallbackResponse.StatusCode())
				ctx.SetBodyString(r.fallbackResponse.Body())
			}
			return
		}
		ctx.SetContentType(MIME_TYPE_JSON)
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		ctx.SetBodyString(ERROR_METHOD_NOT_ALLOWED)
	}
}

type HttpResponse interface {
	ContentType() string
	StatusCode() int
	Body() string
}
