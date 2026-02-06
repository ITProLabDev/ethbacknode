package subscriptions

import "github.com/ITProLabDev/ethbacknode/tools/log"

// BlockEvent handles a new block event from the watchdog.
// Queues the event for processing in the event loop.
func (s *Manager) BlockEvent(blockNum int64, blockId string) {
	s.eventPipe <- func() {
		blockNumStatic := blockNum
		blockIdStatic := blockId
		s.blockEvent(blockNumStatic, blockIdStatic)
	}
}
// blockEvent processes a block event.
// Notifies services, checks transaction confirmations, and updates statuses.
func (s *Manager) blockEvent(blockNum int64, blockId string) {
	go s.blockNotifyServices(blockNum, blockId)
	minConfirmations := s.blockchainClient.MinConfirmations() - 1
	confirmedBlock := int(blockNum) - minConfirmations
	if confirmedBlock < 1 {
		confirmedBlock = 1
	}
	txList, err := s.SearchTransactionsAfterBlock(confirmedBlock)
	if err != nil {
		log.Error("Can not load transactions:", err)
		return
	}
	if len(txList) != 0 {
		log.Info("Found", len(txList), "unconfirmed transactions")
		log.Dump(txList)
	}
	// Send confirmations count event
	for _, tx := range txList {
		// skip duplicated notification
		if tx.BlockNum < int(blockNum) {
			txNotification := new(TransferNotification).fill(tx)
			txNotification.Confirmations = int(blockNum) - tx.BlockNum + 1
			if !tx.Ignore {
				go s.transactionEventPostProcess(txNotification)
			}
		}

	}
	txList, err = s.SearchTransactionsBeforeBlock(confirmedBlock)
	if err != nil {
		log.Error("Can not load transactions:", err)
		return
	}
	if len(txList) != 0 {
		log.Info("Found", len(txList), "confirmed transactions")
	}
	// Send confirmations event
	for _, tx := range txList {
		tx.Confirmed = true
		err = s.saveTransaction(tx)
		if err != nil {
			log.Error("Can not save transaction info:", err)
			continue
		}
		txNotification := new(TransferNotification).fill(tx)
		txNotification.Confirmations = int(blockNum) - tx.BlockNum
		if !tx.Ignore {
			go s.transactionEventPostProcess(txNotification)
		}
	}
}

// blockNotifyServices notifies all subscribers that have ReportNewBlock enabled.
func (s *Manager) blockNotifyServices(blockNum int64, blockId string) {
	blockNotification := &BlockNotification{
		ChainId:  s.blockchainClient.GetChainId(),
		BlockNum: blockNum,
		BlockId:  blockId,
	}
	s.subscriptionViewAll(func(service *Subscription) {
		if service.ReportNewBlock {
			go service.sendNotification("blockEvent", blockNotification, s.config.Debug)
		}
	})
}

// BlockNotification is the payload sent to subscribers for block events.
type BlockNotification struct {
	ChainId  string
	BlockNum int64
	BlockId  string
}
