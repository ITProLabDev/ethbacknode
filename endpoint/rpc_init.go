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

	// Write methods are SECURED (require serviceId + API token, like the other
	// subscriber-mutating methods). Read-only list methods are open.
	r.RegisterSecuredProcessor("contract.register", r.rpcProcessContractRegister)
	r.RegisterSecuredProcessor("contractRegister", r.rpcProcessContractRegister)
	r.RegisterSecuredProcessor("contract.subscribe", r.rpcProcessContractSubscribe)
	r.RegisterSecuredProcessor("contractSubscribe", r.rpcProcessContractSubscribe)
	r.RegisterSecuredProcessor("contract.unsubscribe", r.rpcProcessContractUnsubscribe)
	r.RegisterSecuredProcessor("contractUnsubscribe", r.rpcProcessContractUnsubscribe)
	r.RegisterProcessor("contract.list", r.rpcProcessContractList)
	r.RegisterProcessor("contractList", r.rpcProcessContractList)
	r.RegisterProcessor("contract.subscriptions", r.rpcProcessContractListSubscriptions)
	r.RegisterProcessor("contractSubscriptions", r.rpcProcessContractListSubscriptions)
	r.RegisterProcessor("contract.call", r.rpcProcessContractCall)
	r.RegisterProcessor("contractCall", r.rpcProcessContractCall)
}
