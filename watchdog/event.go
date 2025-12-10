package watchdog

import "module github.com/ITProLabDev/ethbacknode/types"

type BlockEvent func(blockNum int64, blockId string)
type TransactionEvent func(transactionInfo *types.TransferInfo)

type event struct {
	blockEvent       bool
	transactionEvent bool
	blockNum         int64
	blockId          string
	blockTime        int64
	transaction      *types.TransferInfo
}

func (w *Service) RegisterBlockEventListen(handler BlockEvent) {
	w.blockEventHandlers = append(w.blockEventHandlers, handler)
}
func (w *Service) RegisterTransactionEventListen(handler TransactionEvent) {
	w.transactionHandlers = append(w.transactionHandlers, handler)
}

func (w *Service) eventLoop() {
	for {
		event := <-w.events
		if event.blockEvent {
			for _, h := range w.blockEventHandlers {
				go h(event.blockNum, event.blockId)
			}
		} else if event.transactionEvent {
			for _, h := range w.transactionHandlers {
				go h(event.transaction)
			}
		}
	}
}
