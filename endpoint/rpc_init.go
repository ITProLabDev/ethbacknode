package endpoint

func (r *BackRpc) InitProcessors() {
	r.rpcProcessors["ping"] = r.rpcProcessPing
	r.rpcProcessors["info"] = r.rpcProcessNodeInfo
	r.rpcProcessors["getNodeInfo"] = r.rpcProcessNodeInfo

	r.rpcProcessors["infoGetTokenList"] = r.rpcProcessInfoGetTokenList
	r.rpcProcessors["info.get.token.list"] = r.rpcProcessInfoGetTokenList

	r.rpcProcessors["address.balance"] = r.rpcProcessGetBalance
	r.rpcProcessors["addressGetBalance"] = r.rpcProcessGetBalance

	r.rpcProcessors["address.subscribe"] = r.rpcProcessAddressSubscribe
	r.rpcProcessors["addressSubscribe"] = r.rpcProcessAddressSubscribe

	r.rpcProcessors["address.get.new"] = r.rpcProcessAddressGetNew
	r.rpcProcessors["addressGetNew"] = r.rpcProcessAddressGetNew

	r.rpcProcessors["address.recover"] = r.rpcProcessAddressRecover
	r.rpcProcessors["addressRecover"] = r.rpcProcessAddressRecover

	r.rpcProcessors["address.generate"] = r.rpcProcessAddressGenerate
	r.rpcProcessors["addressGenerate"] = r.rpcProcessAddressGenerate

	r.rpcProcessors["service.register"] = r.rpcProcessServiceRegister
	r.rpcProcessors["serviceRegister"] = r.rpcProcessServiceRegister
	r.rpcProcessors["service.config"] = r.rpcProcessServiceConfig
	r.rpcProcessors["serviceConfig"] = r.rpcProcessServiceConfig

	r.rpcProcessors["transfer.info"] = r.rpcProcessGetTransferInfo
	r.rpcProcessors["transferInfo"] = r.rpcProcessGetTransferInfo
	r.rpcProcessors["transfer.info.for.address"] = r.rpcProcessGetTransfersForAddress
	r.rpcProcessors["transferInfoForAddress"] = r.rpcProcessGetTransfersForAddress

	r.rpcProcessors["transfer.assets"] = r.rpcProcessTransferAssets
	r.rpcProcessors["transferAssets"] = r.rpcProcessTransferAssets

	r.rpcProcessors["transfer.get.estimated.fee"] = r.rpcProcessTransferGetEstimatedFee
	r.rpcProcessors["transferGetEstimatedFee"] = r.rpcProcessTransferGetEstimatedFee

}
