package endpoint

// rpcProcessContractCall invokes a contract view method and returns its decoded
// outputs. params: {address, method, args (optional, positional)}.
//
// NOTE: args support in this first version is limited to no-arg view methods
// (e.g. totalSupply, decimals). Typed-argument encoding from JSON is a
// follow-up; passing args returns an error to avoid silently mis-encoding.
func (r *BackRpc) rpcProcessContractCall(ctx RequestContext, request RpcRequest, response RpcResponse) {
	type params struct {
		Address string `json:"address"`
		Method  string `json:"method"`
		Args    []any  `json:"args"`
	}
	p := &params{}
	if err := request.ParseParams(p); err != nil {
		response.SetError(ERROR_CODE_PARSE_ERROR, ERROR_MESSAGE_PARSE_ERROR)
		return
	}
	if p.Address == "" || p.Method == "" {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "address and method are required")
		return
	}
	if len(p.Args) > 0 {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "contractCall with arguments is not yet supported")
		return
	}
	if r.contractCaller == nil {
		response.SetError(ERROR_CODE_SERVER_ERROR, "contract caller not configured")
		return
	}
	out, err := r.contractCaller.CallMethod(p.Address, p.Method)
	if err != nil {
		response.SetError(ERROR_CODE_INVALID_REQUEST, err.Error())
		return
	}
	response.SetResult(out)
}
