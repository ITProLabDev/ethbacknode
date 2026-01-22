package subscriptions

import (
	"errors"
	"github.com/ITProLabDev/ethbacknode/types"
	"github.com/timshannon/badgerhold"
	"math/big"
)

// getTransactionById retrieves a transaction record by its ID.
// Returns ErrUnknownTransaction if not found.
func (s *Manager) getTransactionById(txId string) (tx *TransferInfoRecord, err error) {
	tx = new(TransferInfoRecord)
	s.transactionPool.Do(func(db *badgerhold.Store) {
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
// saveTransaction persists a transaction record to storage.
func (s *Manager) saveTransaction(tx *TransferInfoRecord) (err error) {
	s.transactionPool.Do(func(db *badgerhold.Store) {
		err = db.Upsert(tx.TxID, tx)
	})
	return err
}
// SearchTransactionsBeforeBlock finds unconfirmed transactions in blocks before the given number.
// Used to detect transactions that have reached confirmation threshold.
func (s *Manager) SearchTransactionsBeforeBlock(blockNum int) (txList []*TransferInfoRecord, err error) {
	s.transactionPool.Do(func(db *badgerhold.Store) {
		err = db.Find(&txList, badgerhold.Where(
			"BlockNum").
			Lt(blockNum).
			And("Confirmed").Eq(false).
			And("InPool").Eq(false).
			SortBy("BlockNum"))
	})
	if err != nil {
		return nil, err
	}
	return txList, nil
}
// SearchTransactionsAfterBlock finds unconfirmed transactions in blocks after the given number.
// Used to track pending confirmation updates.
func (s *Manager) SearchTransactionsAfterBlock(blockNum int) (txList []*TransferInfoRecord, err error) {
	s.transactionPool.Do(func(db *badgerhold.Store) {
		err = db.Find(&txList, badgerhold.Where(
			"BlockNum").
			Gt(blockNum).
			And("Confirmed").Eq(false).
			And("InPool").Eq(false).
			SortBy("BlockNum"))
	})
	if err != nil {
		return nil, err
	}
	return txList, nil
}

// TransferInfoRecord is the persistent storage format for transaction data.
// Stored in BadgerHold with indexed fields for efficient queries.
type TransferInfoRecord struct {
	TxID              string   `json:"tx_id" badgerhold:"key"`
	Timestamp         int64    `json:"timestamp"`
	BlockNum          int      `json:"blockNum" badgerhold:"index"`
	Ignore            bool     `json:"ignore"`
	Success           bool     `json:"success"`
	Transfer          bool     `json:"transfer"`
	NativeCoin        bool     `json:"nativeCoin,omitempty"`
	Symbol            string   `json:"symbol,omitempty"`
	SmartContract     bool     `json:"smartContract,omitempty"`
	From              string   `json:"from" badgerhold:"index"`
	To                string   `json:"to" badgerhold:"index"`
	Amount            *big.Int `json:"amount"`
	Token             string   `json:"token,omitempty"`
	TokenSymbol       string   `json:"tokenSymbol,omitempty"`
	Fee               *big.Int `json:"fee"`
	InPool            bool     `json:"inPool"`
	Confirmed         bool     `json:"confirmed" badgerhold:"index"`
	ChainSpecificData []byte   `json:"chainSpecificData,omitempty"`
}

// fillFromTransferInfo populates the record from a TransferInfo struct.
func (t *TransferInfoRecord) fillFromTransferInfo(info *types.TransferInfo) *TransferInfoRecord {
	t.TxID = info.TxID
	t.Timestamp = info.Timestamp
	t.BlockNum = info.BlockNum
	t.Success = info.Success
	t.Transfer = info.Transfer
	t.NativeCoin = info.NativeCoin
	t.Symbol = info.Symbol
	t.SmartContract = info.SmartContract
	t.From = info.From
	t.To = info.To
	t.Amount = info.Amount
	t.Token = info.Token
	t.TokenSymbol = info.TokenSymbol
	t.Fee = info.Fee
	t.InPool = info.InPool
	t.Confirmed = info.Confirmed
	t.ChainSpecificData = info.ChainSpecificData
	return t
}

// toTransferInfo copies the record data to a TransferInfo struct.
func (t *TransferInfoRecord) toTransferInfo(info *types.TransferInfo) {
	info.TxID = t.TxID
	info.Timestamp = t.Timestamp
	info.BlockNum = t.BlockNum
	info.Success = t.Success
	info.Transfer = t.Transfer
	info.NativeCoin = t.NativeCoin
	info.Symbol = t.Symbol
	info.SmartContract = t.SmartContract
	info.From = t.From
	info.To = t.To
	info.Amount = t.Amount
	info.Token = t.Token
	info.TokenSymbol = t.TokenSymbol
	info.Fee = t.Fee
	info.InPool = t.InPool
	info.Confirmed = t.Confirmed
	info.ChainSpecificData = t.ChainSpecificData
}

// isEqual compares the record with a TransferInfo for equality.
func (t *TransferInfoRecord) isEqual(tx *types.TransferInfo) bool {
	if t.TxID != tx.TxID {
		return false
	}
	// skip Timestamp check, because mempool transactions have no correct timestamp
	if t.BlockNum != tx.BlockNum {
		return false
	}
	if t.Success != tx.Success {
		return false
	}
	if t.Transfer != tx.Transfer {
		return false
	}
	if t.NativeCoin != tx.NativeCoin {
		return false
	}
	if t.Symbol != tx.Symbol {
		return false
	}
	if t.SmartContract != tx.SmartContract {
		return false
	}
	if t.From != tx.From {
		return false
	}
	if t.To != tx.To {
		return false
	}
	if t.Amount.Cmp(tx.Amount) != 0 {
		return false
	}
	if t.Token != tx.Token {
		return false
	}
	if t.TokenSymbol != tx.TokenSymbol {
		return false
	}
	if t.Fee.Cmp(tx.Fee) != 0 {
		return false
	}
	if t.InPool != tx.InPool {
		return false
	}
	return true
}
