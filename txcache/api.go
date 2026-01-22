package txcache

import (
	"github.com/ITProLabDev/ethbacknode/tools/log"
	"github.com/ITProLabDev/ethbacknode/types"
	"github.com/timshannon/badgerhold"
	"sort"
)

// GetTransferInfo retrieves a cached transaction by its hash.
// Returns ErrUnknownTransaction if not found.
func (m *Manager) GetTransferInfo(txHash string) (tx *types.TransferInfo, err error) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	txRecord, err := m.getTransactionById(txHash)
	if err != nil {
		return nil, err
	}
	return txRecord.getTransferInfo(), nil
}

// GetTransfersByAddress retrieves all cached transactions for an address.
// Returns transactions where the address is sender or recipient, sorted by timestamp.
func (m *Manager) GetTransfersByAddress(address string) (txs []*types.TransferInfo, err error) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	var txRecords []*TransferInfoCachedRecord
	m.txCache.Do(func(db *badgerhold.Store) {
		query := badgerhold.Where("From").Eq(address).Or(badgerhold.Where("To").Eq(address))
		err = db.Find(&txRecords, query)
	})
	if err != nil {
		return nil, err
	}
	for _, txRecord := range txRecords {
		txs = append(txs, txRecord.getTransferInfo())
	}
	sort.Sort(sortTransferInfo(txs))
	return txs, nil
}

// TransactionEvent handles a new transaction event from the watchdog.
// Caches the transaction with initial confirmation count.
func (m *Manager) TransactionEvent(transactionInfo *types.TransferInfo) {
	if m.config.Debug {
		log.Debug("TxCache: transaction event", transactionInfo.TxID)
		log.Dump(transactionInfo)
	}
	m.eventPipe <- func() {
		m.mux.Lock()
		defer m.mux.Unlock()
		transactionInfoStatic := new(TransferInfoCachedRecord)
		transactionInfoStatic.loadFromTransferInfo(transactionInfo)
		if !transactionInfoStatic.InPool {
			transactionInfoStatic.Confirmations = 1
		}
		err := m.saveTransaction(transactionInfoStatic)
		if err != nil {
			log.Error("TxCache: can not save transaction info:", err)
		}
	}
}

// BlockEvent handles a new block event to update confirmation counts.
func (m *Manager) BlockEvent(blockNum int64, blockId string) {
	if m.config.Debug {
		log.Debug("TxCache: block event", blockNum)
	}
	m.eventPipe <- func() {
		m.blockUpdateEvent(blockNum)
	}
}

// sortTransferInfo implements sort.Interface for TransferInfo slices by timestamp.
type sortTransferInfo []*types.TransferInfo

func (s sortTransferInfo) Len() int {
	return len(s)
}

func (s sortTransferInfo) Less(i, j int) bool {
	return s[i].Timestamp < s[j].Timestamp
}

func (s sortTransferInfo) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
