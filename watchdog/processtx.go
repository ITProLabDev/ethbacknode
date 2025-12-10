package watchdog

import (
	"module github.com/ITProLabDev/ethbacknode/tools/log"
	"module github.com/ITProLabDev/ethbacknode/types"
)

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
