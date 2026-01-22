package txcache

import (
	"github.com/ITProLabDev/ethbacknode/tools/log"
	"github.com/timshannon/badgerhold"
)

// blockUpdateEvent updates confirmation counts for recent transactions.
// Marks transactions as confirmed when they exceed the confirmation threshold.
func (m *Manager) blockUpdateEvent(blockNum int64) {
	m.mux.Lock()
	defer m.mux.Unlock()
	var err error
	var txToUpdate []*TransferInfoCachedRecord
	m.txCache.Do(func(db *badgerhold.Store) {
		query := badgerhold.Where("BlockNum").Ge(int(blockNum) - m.config.RegisterConfirmations)
		err = db.Find(&txToUpdate, query)
	})
	if err != nil {
		log.Error("TxCache: can not find transactions to update", "err", err)
	}
	if m.config.Debug {
		log.Debug("TxCache: block update event", blockNum)
		log.Dump(txToUpdate)
	}
	for _, tx := range txToUpdate {
		if int(blockNum)-tx.BlockNum > 1 && !tx.InPool {
			tx.Confirmed = tx.BlockNum <= int(blockNum)-m.config.Confirmations
			tx.Confirmations = int(blockNum) - tx.BlockNum
			err = m.saveTransaction(tx)
			if err != nil {
				log.Error("TxCache: can not save transaction info:", err)
			}
		}
	}
}
