package endpoint

import (
	"fmt"
	"strconv"

	"github.com/ITProLabDev/ethbacknode/abi"
	"github.com/ITProLabDev/ethbacknode/common/hexnum"
)

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

// rpcProcessContractSubscribe subscribes a service to a contract's events and
// transactions. params: {serviceId (number), address,
// scope (whole_contract|managed_only), selectors ([]string, optional),
// methods ([]string, optional)}.
//
// selectors and methods both narrow contractTransaction delivery to specific
// operations (contractEvent is unaffected) and may be combined -- a hybrid,
// since they serve different callers: selectors are raw 4-byte hex (e.g.
// "0xa9059cbb") and work even for a method not in this node's registered ABI,
// since matching needs only the 4 bytes observed on-chain, no lookup; methods
// are canonical signatures (e.g. "transfer(address,uint256)"), resolved to
// their selector the same way abi's own signature hashing works -- also no
// ABI lookup needed, just more ergonomic when the caller knows the signature
// by name. Neither given means no filter: every transaction to the contract
// matches, unchanged from before this existed.
//
// serviceId is a JSON number. The secured wrapper authenticates it via
// GetParamInt and requires the subscriber to already exist (and rejects the
// internal service 0). The processor re-reads it from params to convert to the
// decimal-string form the eventlog layer stores, so the delivery sink can route
// the event back to the same subscriptions.Manager serviceId.
func (r *BackRpc) rpcProcessContractSubscribe(ctx RequestContext, request RpcRequest, response RpcResponse) {
	type params struct {
		ServiceID int64    `json:"serviceId"`
		Address   string   `json:"address"`
		Scope     string   `json:"scope"`
		Selectors []string `json:"selectors"`
		Methods   []string `json:"methods"`
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
	// The contract must have a registered ABI: events of an unknown contract can
	// never be decoded, so the subscription would silently deliver nothing.
	// Reject up front (mirrors the legacy ERC-20 _isTokenKnown guard, but keyed
	// on contract address) so callers register the ABI before subscribing.
	if r.abiManager == nil {
		response.SetError(ERROR_CODE_SERVER_ERROR, "contract registry not configured")
		return
	}
	if !r.abiManager.IsContractKnown(p.Address) {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "unknown contract: register its ABI before subscribing")
		return
	}
	selectors, err := resolveSelectors(p.Selectors, p.Methods)
	if err != nil {
		response.SetError(ERROR_CODE_INVALID_REQUEST, err.Error())
		return
	}
	serviceID := strconv.FormatInt(p.ServiceID, 10)
	if err := r.eventLog.SubscribeAndSaveStrings(serviceID, p.Address, p.Scope, selectors); err != nil {
		response.SetError(ERROR_CODE_INVALID_REQUEST, err.Error())
		return
	}
	response.SetResult(map[string]interface{}{"serviceId": p.ServiceID, "address": p.Address, "scope": p.Scope, "status": "subscribed"})
}

// resolveSelectors merges raw hex selectors and canonical method signatures
// into one selector set for a contractTransaction filter. Returns nil (no
// filter) when both are empty, never an empty non-nil slice, so a caller
// that names nothing keeps today's unfiltered behavior.
func resolveSelectors(rawHex, signatures []string) ([][4]byte, error) {
	if len(rawHex) == 0 && len(signatures) == 0 {
		return nil, nil
	}
	out := make([][4]byte, 0, len(rawHex)+len(signatures))
	for _, h := range rawHex {
		b, err := hexnum.ParseHexBytes(h)
		if err != nil {
			return nil, fmt.Errorf("selectors: %q: %w", h, err)
		}
		if len(b) != 4 {
			return nil, fmt.Errorf("selectors: %q is %d bytes, want 4", h, len(b))
		}
		var sel [4]byte
		copy(sel[:], b)
		out = append(out, sel)
	}
	for _, sig := range signatures {
		out = append(out, abi.Selector(sig))
	}
	return out, nil
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
