package txcache

import (
	"errors"
	"github.com/timshannon/badgerhold"
)

func (m *Manager) getTransactionById(txId string) (tx *TransferInfoCachedRecord, err error) {
	tx = new(TransferInfoCachedRecord)
	m.txCache.Do(func(db *badgerhold.Store) {
		err = db.Get(txId, tx)
		if errors.Is(err, badgerhold.ErrNotFound) {
			err = ErrUnknownTransaction
		}
	})
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (m *Manager) getTransactionsByAddress(address string) (txs []*TransferInfoCachedRecord, err error) {
	m.txCache.Do(func(db *badgerhold.Store) {
		q := badgerhold.Where("From").Eq(address).Or(badgerhold.Where("To").Eq(address))
		err = db.Find(&txs, q)
	})
	if err != nil {
		return nil, err
	}
	return txs, nil
}

func (m *Manager) saveTransaction(tx *TransferInfoCachedRecord) (err error) {
	m.txCache.Do(func(db *badgerhold.Store) {
		err = db.Upsert(tx.TxID, tx)
	})
	return err
}
