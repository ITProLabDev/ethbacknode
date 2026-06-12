package endpoint

import (
	"errors"
	"strconv"
)

// errBadScope is returned when a subscription scope is invalid. (The eventlog
// layer validates the scope string; this sentinel is for test clarity.)
var errBadScope = errors.New("invalid scope")

// rpcProcessContractRegister registers a contract + ABI. params:
// {name, symbol, address, abi (canonical JSON ABI string)}.
func (r *BackRpc) rpcProcessContractRegister(ctx RequestContext, request RpcRequest, response RpcResponse) {
	type params struct {
		Name    string `json:"name"`
		Symbol  string `json:"symbol"`
		Address string `json:"address"`
		ABI     string `json:"abi"`
	}
	p := &params{}
	if err := request.ParseParams(p); err != nil {
		response.SetError(ERROR_CODE_PARSE_ERROR, ERROR_MESSAGE_PARSE_ERROR)
		return
	}
	if p.Address == "" || p.ABI == "" {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "address and abi are required")
		return
	}
	if r.contractAdder == nil {
		response.SetError(ERROR_CODE_SERVER_ERROR, "contract registry not configured")
		return
	}
	if err := r.contractAdder.AddContractFromABI(p.Name, p.Symbol, p.Address, []byte(p.ABI)); err != nil {
		response.SetError(ERROR_CODE_INVALID_REQUEST, err.Error())
		return
	}
	response.SetResult(map[string]string{"name": p.Name, "address": p.Address, "status": "registered"})
}

// rpcProcessContractList returns the registered contracts as name→address.
func (r *BackRpc) rpcProcessContractList(ctx RequestContext, request RpcRequest, response RpcResponse) {
	if r.abiManager == nil {
		response.SetError(ERROR_CODE_SERVER_ERROR, "contract registry not configured")
		return
	}
	response.SetResult(r.abiManager.GetSmartContractList())
}

// rpcProcessContractSubscribe subscribes a service to a contract's events.
// params: {serviceId (number), address, scope (whole_contract|managed_only)}.
//
// serviceId is a JSON number (the secured wrapper authenticates it via
// GetParamInt and requires the subscriber to already exist). The eventlog layer
// stores it as the decimal string form so the delivery sink can route to the
// same subscriptions.Manager serviceId.
func (r *BackRpc) rpcProcessContractSubscribe(ctx RequestContext, request RpcRequest, response RpcResponse) {
	type params struct {
		ServiceID int64  `json:"serviceId"`
		Address   string `json:"address"`
		Scope     string `json:"scope"`
	}
	p := &params{}
	if err := request.ParseParams(p); err != nil {
		response.SetError(ERROR_CODE_PARSE_ERROR, ERROR_MESSAGE_PARSE_ERROR)
		return
	}
	if p.ServiceID == 0 || p.Address == "" {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "serviceId and address are required")
		return
	}
	if r.eventLog == nil {
		response.SetError(ERROR_CODE_SERVER_ERROR, "event subscriber not configured")
		return
	}
	serviceID := strconv.FormatInt(p.ServiceID, 10)
	if err := r.eventLog.SubscribeAndSaveStrings(serviceID, p.Address, p.Scope); err != nil {
		response.SetError(ERROR_CODE_INVALID_REQUEST, err.Error())
		return
	}
	response.SetResult(map[string]interface{}{"serviceId": p.ServiceID, "address": p.Address, "scope": p.Scope, "status": "subscribed"})
}

// rpcProcessContractUnsubscribe removes a service's subscription to a contract.
// params: {serviceId (number), address}.
func (r *BackRpc) rpcProcessContractUnsubscribe(ctx RequestContext, request RpcRequest, response RpcResponse) {
	type params struct {
		ServiceID int64  `json:"serviceId"`
		Address   string `json:"address"`
	}
	p := &params{}
	if err := request.ParseParams(p); err != nil {
		response.SetError(ERROR_CODE_PARSE_ERROR, ERROR_MESSAGE_PARSE_ERROR)
		return
	}
	if p.ServiceID == 0 || p.Address == "" {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "serviceId and address are required")
		return
	}
	if r.eventLog == nil {
		response.SetError(ERROR_CODE_SERVER_ERROR, "event subscriber not configured")
		return
	}
	serviceID := strconv.FormatInt(p.ServiceID, 10)
	if err := r.eventLog.UnsubscribeStrings(serviceID, p.Address); err != nil {
		response.SetError(ERROR_CODE_INVALID_REQUEST, err.Error())
		return
	}
	response.SetResult(map[string]interface{}{"serviceId": p.ServiceID, "address": p.Address, "status": "unsubscribed"})
}

// rpcProcessContractListSubscriptions returns the active event subscriptions.
func (r *BackRpc) rpcProcessContractListSubscriptions(ctx RequestContext, request RpcRequest, response RpcResponse) {
	if r.eventLog == nil {
		response.SetError(ERROR_CODE_SERVER_ERROR, "event subscriber not configured")
		return
	}
	response.SetResult(r.eventLog.ListSubscriptions())
}
