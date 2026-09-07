package uniclient

// Smart Contract Layer client methods: register/subscribe/unsubscribe/list a
// contract, and call its read-only view methods. See API.md's "Smart
// Contract Layer" section for the full wire contract these mirror.
//
// DecodedValue deliberately holds Value as a raw interface{} rather than a
// typed Go value: the server encodes it JSON-safely per ABI type (byte types
// as 0x-hex strings, big integers as decimal strings -- see API.md's "Decoded
// value format"), and this package does not import abi/ to stay a
// self-contained client library any Go service can embed.

// DecodedValue is one decoded ABI value, as returned by ContractCall and
// carried on a contractEvent/contractTransaction notification's inputs.
type DecodedValue struct {
	Name  string      `json:"name,omitempty"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// ContractRegisterResult is the result of ContractRegister.
type ContractRegisterResult struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Status  string `json:"status"`
}

// ContractRegister registers a contract + its ABI so the service can decode
// its events/transactions and call its methods. Secured.
func (c *Client) ContractRegister(name, symbol, address, rawABI string) (result *ContractRegisterResult, err error) {
	type contractRegisterRequest struct {
		Name    string `json:"name"`
		Symbol  string `json:"symbol"`
		Address string `json:"address"`
		ABI     string `json:"abi"`
	}
	request := NewRequest("contractRegister", &contractRegisterRequest{
		Name: name, Symbol: symbol, Address: address, ABI: rawABI,
	})
	result = new(ContractRegisterResult)
	if err = c.rpcCall(request, result); err != nil {
		return nil, err
	}
	return result, nil
}

// ContractSubscribeResult is the result of ContractSubscribe.
type ContractSubscribeResult struct {
	ServiceID int    `json:"serviceId"`
	Address   string `json:"address"`
	Scope     string `json:"scope"`
	Status    string `json:"status"`
}

// ContractSubscribe subscribes the client's configured service (WithServiceId)
// to a registered contract's events (contractEvent) and, for whole_contract
// scope, its transactions (contractTransaction). Secured; the contract must
// already be registered via ContractRegister.
//
// selectors and methods both narrow contractTransaction delivery to specific
// operations and may be combined -- selectors are raw 4-byte hex (e.g.
// "0xa9059cbb"), methods are canonical signatures (e.g.
// "transfer(address,uint256)") the server resolves to their selector. Either
// may be nil; neither given means no filter (every transaction delivered).
// See API.md's "Filtering contractTransaction by operation".
func (c *Client) ContractSubscribe(address, scope string, selectors, methods []string) (result *ContractSubscribeResult, err error) {
	type contractSubscribeRequest struct {
		ServiceID int      `json:"serviceId"`
		Address   string   `json:"address"`
		Scope     string   `json:"scope"`
		Selectors []string `json:"selectors,omitempty"`
		Methods   []string `json:"methods,omitempty"`
	}
	request := NewRequest("contractSubscribe", &contractSubscribeRequest{
		ServiceID: c.serviceId, Address: address, Scope: scope, Selectors: selectors, Methods: methods,
	})
	result = new(ContractSubscribeResult)
	if err = c.rpcCall(request, result); err != nil {
		return nil, err
	}
	return result, nil
}

// ContractUnsubscribeResult is the result of ContractUnsubscribe.
type ContractUnsubscribeResult struct {
	ServiceID int    `json:"serviceId"`
	Address   string `json:"address"`
	Status    string `json:"status"`
}

// ContractUnsubscribe removes the client's configured service's subscription
// to a contract (both contractEvent and contractTransaction). Secured;
// idempotent -- removing a non-existent subscription is a no-op success.
func (c *Client) ContractUnsubscribe(address string) (result *ContractUnsubscribeResult, err error) {
	type contractUnsubscribeRequest struct {
		ServiceID int    `json:"serviceId"`
		Address   string `json:"address"`
	}
	request := NewRequest("contractUnsubscribe", &contractUnsubscribeRequest{ServiceID: c.serviceId, Address: address})
	result = new(ContractUnsubscribeResult)
	if err = c.rpcCall(request, result); err != nil {
		return nil, err
	}
	return result, nil
}

// ContractList returns the registered contracts as name -> address. Open
// (no auth required).
func (c *Client) ContractList() (contracts map[string]string, err error) {
	request := NewRequest("contractList", nil)
	contracts = make(map[string]string)
	if err = c.rpcCall(request, &contracts); err != nil {
		return nil, err
	}
	return contracts, nil
}

// ContractSubscriptionInfo is one entry returned by ContractSubscriptions.
type ContractSubscriptionInfo struct {
	ServiceID string   `json:"serviceId"`
	Address   string   `json:"address"`
	Scope     string   `json:"scope"`
	Selectors []string `json:"selectors"`
}

// ContractSubscriptions lists every active contract subscription across all
// services. Open (no auth required).
func (c *Client) ContractSubscriptions() (subs []*ContractSubscriptionInfo, err error) {
	request := NewRequest("contractSubscriptions", nil)
	if err = c.rpcCall(request, &subs); err != nil {
		return nil, err
	}
	return subs, nil
}

// ContractCall calls a read-only (view/pure) method of a registered contract
// and returns its decoded outputs. Open (no auth required).
//
// args is forwarded as-is; as of API.md's current server behavior, a
// non-empty args slice is rejected ("contractCall with arguments is not yet
// supported") -- this method still accepts it so callers do not need an API
// change here once the server adds typed-argument support.
func (c *Client) ContractCall(address, method string, args ...interface{}) (result []DecodedValue, err error) {
	type contractCallRequest struct {
		Address string        `json:"address"`
		Method  string        `json:"method"`
		Args    []interface{} `json:"args,omitempty"`
	}
	request := NewRequest("contractCall", &contractCallRequest{Address: address, Method: method, Args: args})
	if err = c.rpcCall(request, &result); err != nil {
		return nil, err
	}
	return result, nil
}
