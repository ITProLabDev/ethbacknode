package subscriptions

import (
	"errors"
	"module github.com/ITProLabDev/ethbacknode/tools/log"
	"module github.com/ITProLabDev/ethbacknode/types"
)

func (s *Manager) TransactionEvent(transactionInfo *types.TransferInfo) {
	s.eventPipe <- func() {
		transactionInfoStatic := transactionInfo
		s.transactionEventProcess(transactionInfoStatic)
	}
}

func (s *Manager) transactionEventProcess(transactionInfo *types.TransferInfo) {
	var txInfo *TransferInfoRecord
	var err error
	if s.config.Debug {
		if transactionInfo.InPool {
			log.Warning("Process Mempool transaction")
			log.Dump(transactionInfo)
		} else {
			log.Warning("Process In Block transaction")
			log.Dump(transactionInfo)
		}
	}
	txInfo, err = s.getTransactionById(transactionInfo.TxID)
	if errors.Is(err, ErrUnknownTransaction) {
		//seems like new transaction, save it and send event
		txInfo = new(TransferInfoRecord).fillFromTransferInfo(transactionInfo)
		err = s.saveTransaction(txInfo)
		if err != nil {
			log.Error("Can not save transaction info:", err)
			return
		}
	} else if err != nil {
		log.Error("error getting transaction by id", err)
		return
	} else if txInfo.isEqual(transactionInfo) {
		if s.config.Debug {
			log.Debug("Transaction already known, skip")
		}
		return
	} else if txInfo.InPool && !transactionInfo.InPool && !txInfo.Ignore {
		txInfo.BlockNum = transactionInfo.BlockNum
		txInfo.Timestamp = transactionInfo.Timestamp
		txInfo.InPool = false
		err = s.saveTransaction(txInfo)
		if err != nil {
			log.Error("Can not save transaction info:", err)
		}
	} else {
		if s.config.Debug {
			log.Debug("Transaction ignored, skip")
		}
		return
	}
	txNotification := new(TransferNotification).fill(txInfo)
	if !txNotification.InPool {
		txNotification.Confirmations = 1
	}
	if txInfo.Ignore {
		return
	}
	go s.transactionEventPostProcess(txNotification)
}

func (s *Manager) transactionEventPostProcess(transactionInfo *TransferNotification) {
	s.notifyMux.Lock()
	defer s.notifyMux.Unlock()
	from := transactionInfo.From
	to := transactionInfo.To
	if s.addressPool.IsAddressKnown(from) {
		addressInfo, _ := s.addressPool.GetAddress(from)
		serviceInfo, err := s.SubscriptionGet(ServiceId(addressInfo.ServiceId))
		if err != nil {
			return
		}
		//TODO move to channels
		if serviceInfo.ReportOutgoingTx {
			transactionInfo.ChainId = s.blockchainClient.GetChainId()
			go s.NotifySubscriber(ServiceId(addressInfo.ServiceId), "transactionEvent", transactionInfo)
		}
	}
	if s.addressPool.IsAddressKnown(to) {
		addressInfo, _ := s.addressPool.GetAddress(to)
		serviceInfo, err := s.SubscriptionGet(ServiceId(addressInfo.ServiceId))
		if err != nil {
			return
		}
		//TODO move to channels
		if serviceInfo.ReportIncomingTx {
			transactionInfo.ChainId = s.blockchainClient.GetChainId()
			transactionInfo.UserId = addressInfo.UserId
			transactionInfo.InvoiceId = addressInfo.InvoiceId
			go s.NotifySubscriber(ServiceId(addressInfo.ServiceId), "transactionEvent", transactionInfo)
		}
		if transactionInfo.Confirmed && transactionInfo.Success {
			s.gatherNativeCoinToMaster(addressInfo, transactionInfo)
		}
	}
}
