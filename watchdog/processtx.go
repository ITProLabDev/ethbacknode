package watchdog

import (
	"github.com/ITProLabDev/ethbacknode/tools/log"
	"github.com/ITProLabDev/ethbacknode/types"
)

// processTx checks if a transaction involves any managed addresses.
// Fires a transaction event if the sender or recipient is managed.
// Avoids duplicate events when both parties are managed.
func (w *Service) processTx(tx *types.TransferInfo) error {
	eventSent := false
	if w.addressPool.IsAddressKnown(tx.From) {
		if w.config.Debug {
			log.Debug("FROM Address known, fire event:", tx.From, "to:", tx.To)
		}
		w.events <- &event{
			transactionEvent: true,
			transaction:      tx,
		}
		eventSent = true
	}
	if w.addressPool.IsAddressKnown(tx.To) {
		if w.config.Debug {
			log.Debug("TO Address known, fire event:", tx.From, "to:", tx.To)
		}
		if !eventSent {
			w.events <- &event{
				transactionEvent: true,
				transaction:      tx,
			}
		}
	}
	return nil
}
