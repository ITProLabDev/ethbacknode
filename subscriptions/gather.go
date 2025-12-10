package subscriptions

import (
	"github.com/ITProLabDev/ethbacknode/address"
	"github.com/ITProLabDev/ethbacknode/tools/log"
)

func (s *Manager) gatherNativeCoinToMaster(address *address.Address, txInfo *TransferNotification) {
	serviceInfo, err := s.SubscriptionGet(ServiceId(address.ServiceId))
	if err != nil {
		return
	}
	if !serviceInfo.GatherToMaster {
		//service don't need to gather
		return
	}
	log.Warning("Service ", address.ServiceId, " need to gather from", address.Address, " to master")
	if len(serviceInfo.MasterList) == 0 {
		log.Warning("Service ", address.ServiceId, " don't have master list")
		return
	}
	masterAddress := serviceInfo.MasterList[0]
	var txId string
	if txInfo.NativeCoin {
		txId, err = s.blockchainClient.TransferAllByPrivateKey(address.PrivateKey, address.Address, masterAddress)
		if err != nil {
			if s.globalConfig.Flag("debug") {
				log.Warning("Service ", address.ServiceId, "Can not transfer all to master:", err, ", skip")
			}
			return
		}
	}
	if txId != "" {
		sendTxInfo, err := s.blockchainClient.TransferInfoByHash(txId)
		if err != nil {
			log.Error("Service ", address.ServiceId, "Can not get transfer info:", err)
			return
		}
		txRecord := new(TransferInfoRecord).fillFromTransferInfo(sendTxInfo)
		txRecord.Ignore = true
		err = s.saveTransaction(txRecord)
		if err != nil {
			log.Error("Service ", address.ServiceId, "Can not save transfer info:", err)
		}
	}
}
