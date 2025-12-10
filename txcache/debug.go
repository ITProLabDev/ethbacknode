package txcache

import (
	"github.com/timshannon/badgerhold"
	"module github.com/ITProLabDev/ethbacknode/tools/log"
)

func (m *Manager) DumpDb() {
	var txList []*TransferInfoCachedRecord
	var err error
	m.txCache.Do(func(db *badgerhold.Store) {
		err = db.Find(&txList, badgerhold.Where("Confirmed").Eq(false))
	})
	if err != nil {
		log.Error("dumpDb error", "err", err)
		return
	}
	log.Dump(txList)
}
