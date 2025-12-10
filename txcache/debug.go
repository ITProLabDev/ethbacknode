package txcache

import (
	"github.com/ITProLabDev/ethbacknode/tools/log"
	"github.com/timshannon/badgerhold"
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
