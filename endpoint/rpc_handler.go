package endpoint

import (
	"encoding/json"
	"errors"

	"github.com/ITProLabDev/ethbacknode/address"
	"github.com/ITProLabDev/ethbacknode/security"
	"github.com/ITProLabDev/ethbacknode/subscriptions"
	"github.com/ITProLabDev/ethbacknode/tools/log"
	"github.com/ITProLabDev/ethbacknode/types"
	"github.com/ITProLabDev/ethbacknode/watchdog"

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
	security         *security.Manager
	watchdog         *watchdog.Service
	txCache          types.TxCache
	fallbackResponse HttpResponse
	addressCodec     address.AddressCodec
	rpcProcessors    map[RpcMethod]RpcProcessor
	securityManager  *security.Manager
	burnAddress      string
}

type BackRpcOption func(r *BackRpc)

func NewBackRpc(addressPool *address.Manager, chainClient types.ChainClient, subscriptions *subscriptions.Manager, watchdog *watchdog.Service, txCache types.TxCache, options ...BackRpcOption) *BackRpc {
	r := &BackRpc{
		addressPool:     addressPool,
		chainClient:     chainClient,
		knownTokens:     make(map[string]*types.TokenInfo),
		subscriptions:   subscriptions,
		watchdog:        watchdog,
		txCache:         txCache,
		addressCodec:    chainClient.GetAddressCodec(),
		rpcProcessors:   make(map[RpcMethod]RpcProcessor),
		securityManager: security.NewManager(),
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
		log.Debug("Handle rpc request:", string(ctx.Method()), string(ctx.Path()))
	}
	//TODO add rate limit
	//TODO limit request size
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

func (r *BackRpc) RegisterProcessor(method RpcMethod, processor RpcProcessor) {
	r.rpcProcessors[method] = processor
}

func (r *BackRpc) RegisterSecuredProcessor(method RpcMethod, processor RpcProcessor) {
	r.rpcProcessors[method] = func(ctx RequestContext, request RpcRequest, response RpcResponse) {
		serviceId, err := request.GetParamInt("serviceId")
		if err != nil {
			response.SetError(ERROR_CODE_INVALID_REQUEST, ERROR_MESSAGE_INVALID_REQUEST)
			return
		}
		if r.debugMode {
			log.Debug("rpc processor auth for serviceId:", serviceId)
		}
		subscriber, err := r.subscriptions.SubscriptionGet(subscriptions.ServiceId(serviceId))
		if err != nil {
			log.Error("Invalid serviceId:", serviceId, ", err:", err)
			response.SetErrorWithData(ERROR_CODE_UNAUTHORIZED, ERROR_MESSAGE_UNAUTHORIZED, "serviceId required")
			return
		}
		if subscriber.ApiToken != "" || subscriber.ApiKey != "" {
			switch {
			case subscriber.ApiToken != "":
				requestApiToken, err := ctx.GetApiToken()
				if err != nil {
					response.SetErrorWithData(ERROR_CODE_UNAUTHORIZED, ERROR_MESSAGE_UNAUTHORIZED, "api token required")
					return
				}
				if subscriber.ApiToken != requestApiToken {
					response.SetErrorWithData(ERROR_CODE_UNAUTHORIZED, ERROR_MESSAGE_UNAUTHORIZED, "api token required")
					return
				}
				ctx.Authorized(true)
			case subscriber.ApiKey != "":
				if r.securityManager == nil {
					log.Critical("SecurityManager not initialized")
					response.SetError(ERROR_CODE_SERVER_ERROR, ERROR_MESSAGE_SERVER_ERROR)
					return
				}
				paramsPaw := make(map[string]json.RawMessage)
				if sign, found := paramsPaw["signature"]; !found || sign == nil {
					response.SetErrorWithData(ERROR_CODE_UNAUTHORIZED, ERROR_MESSAGE_UNAUTHORIZED, "request signature required")
					return
				}
				//todo validate sign
			}
		}
		processor(ctx, request, response)
	}
}
