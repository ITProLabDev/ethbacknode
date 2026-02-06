package endpoint

// InitProcessors registers all built-in RPC method processors.
// Sets up processors for ping, info, address, balance, transfer, and service methods.
func (r *BackRpc) InitProcessors() {
	r.RegisterProcessor("ping", r.rpcProcessPing)
	r.RegisterProcessor("info", r.rpcProcessNodeInfo)
	r.RegisterProcessor("getNodeInfo", r.rpcProcessNodeInfo)

	r.RegisterProcessor("infoGetTokenList", r.rpcProcessInfoGetTokenList)
	r.RegisterProcessor("info.get.token.list", r.rpcProcessInfoGetTokenList)

	r.RegisterProcessor("address.balance", r.rpcProcessGetBalance)
	r.RegisterProcessor("addressGetBalance", r.rpcProcessGetBalance)

	r.RegisterProcessor("address.subscribe", r.rpcProcessAddressSubscribe)
	r.RegisterProcessor("addressSubscribe", r.rpcProcessAddressSubscribe)

	r.RegisterSecuredProcessor("address.get.new", r.rpcProcessAddressGetNew)
	r.RegisterSecuredProcessor("addressGetNew", r.rpcProcessAddressGetNew)

	r.RegisterProcessor("address.recover", r.rpcProcessAddressRecover)
	r.RegisterProcessor("addressRecover", r.rpcProcessAddressRecover)

	r.RegisterSecuredProcessor("address.generate", r.rpcProcessAddressGenerate)
	r.RegisterSecuredProcessor("addressGenerate", r.rpcProcessAddressGenerate)

	r.RegisterProcessor("service.register", r.rpcProcessServiceRegister)
	r.RegisterProcessor("serviceRegister", r.rpcProcessServiceRegister)

	r.RegisterSecuredProcessor("service.config", r.rpcProcessServiceConfig)
	r.RegisterSecuredProcessor("serviceConfig", r.rpcProcessServiceConfig)

	r.RegisterProcessor("transfer.info", r.rpcProcessGetTransferInfo)
	r.RegisterProcessor("transferInfo", r.rpcProcessGetTransferInfo)

	r.RegisterSecuredProcessor("transfer.info.for.address", r.rpcProcessGetTransfersForAddress)
	r.RegisterSecuredProcessor("transferInfoForAddress", r.rpcProcessGetTransfersForAddress)

	r.RegisterSecuredProcessor("transfer.assets", r.rpcProcessTransferAssets)
	r.RegisterSecuredProcessor("transferAssets", r.rpcProcessTransferAssets)

	r.RegisterSecuredProcessor("transfer.get.estimated.fee", r.rpcProcessTransferGetEstimatedFee)
	r.RegisterSecuredProcessor("transferGetEstimatedFee", r.rpcProcessTransferGetEstimatedFee)
}
