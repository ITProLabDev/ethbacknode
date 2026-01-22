package watchdog

import "github.com/ITProLabDev/ethbacknode/types"

// BlockEvent is a callback function invoked when a new block is detected.
type BlockEvent func(blockNum int64, blockId string)

// TransactionEvent is a callback function invoked when a transaction
// involving a managed address is detected.
type TransactionEvent func(transactionInfo *types.TransferInfo)

// event is an internal event structure for the event queue.
type event struct {
	blockEvent       bool
	transactionEvent bool
	blockNum         int64
	blockId          string
	blockTime        int64
	transaction      *types.TransferInfo
}

// RegisterBlockEventListen adds a handler to be called when new blocks are detected.
func (w *Service) RegisterBlockEventListen(handler BlockEvent) {
	w.blockEventHandlers = append(w.blockEventHandlers, handler)
}
// RegisterTransactionEventListen adds a handler for transaction events.
func (w *Service) RegisterTransactionEventListen(handler TransactionEvent) {
	w.transactionHandlers = append(w.transactionHandlers, handler)
}

// eventLoop processes events from the internal queue and dispatches to handlers.
// Runs as a goroutine, invoking handlers asynchronously.
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
