package txmanager

import (
	"fmt"
	"strings"

	"github.com/timshannon/badgerhold"

	"github.com/ITProLabDev/ethbacknode/storage"
)

// nonceRecord is the persisted trace of one nonce this node has issued for an
// address. It exists only so a restart can reconstruct highestNonceFor: the
// allocator's in-memory cursor does not survive a process restart, but a
// transaction actually broadcast under a nonce cannot be forgotten without
// risking that nonce being issued again.
type nonceRecord struct {
	Hash  string `badgerhold:"key"`
	From  string `badgerhold:"index"`
	Nonce int64
}

// Store is the narrow slice of a badgerhold-backed store this package needs.
// Declared here, rather than requiring *badgerhold.Store directly, so a test
// can supply a real temp-directory-backed store without going through
// storage.BadgerHoldStorage's file-path plumbing. *badgerhold.Store satisfies
// this already.
type Store interface {
	Upsert(key, data interface{}) error
	Find(result interface{}, query *badgerhold.Query) error
}

// records is the store with the queries this package reads through.
type records struct {
	store Store
}

func newRecords(store Store) *records { return &records{store: store} }

// save persists that nonce was issued for from, under txHash. Address
// matching elsewhere in this package is case-insensitive, so the address is
// normalized before it becomes an index key.
func (r *records) save(from, txHash string, nonce int64) error {
	rec := &nonceRecord{Hash: txHash, From: normalizeAddress(from), Nonce: nonce}
	if err := r.store.Upsert(rec.Hash, rec); err != nil {
		return fmt.Errorf("txmanager: save nonce record %s: %w", txHash, err)
	}
	return nil
}

// highestNonceFor returns the highest nonce this node has issued for an
// address, and whether there was one at all.
func (r *records) highestNonceFor(from string) (nonce int64, found bool, err error) {
	var out []*nonceRecord
	query := badgerhold.Where("From").Eq(normalizeAddress(from)).SortBy("Nonce").Reverse().Limit(1)
	if err := r.store.Find(&out, query); err != nil {
		return 0, false, fmt.Errorf("txmanager: highest nonce for %s: %w", from, err)
	}
	if len(out) == 0 {
		return 0, false, nil
	}
	return out[0].Nonce, true, nil
}

// normalizeAddress makes address matching checksum-tolerant: an EIP-55
// checksummed address and its lowercase form must name the same nonce
// cursor, matching the abi package's registry convention (own-abi-no-eth-libs).
func normalizeAddress(address string) string { return strings.ToLower(address) }

// badgerHoldStore adapts *storage.BadgerHoldStorage to Store, going through
// its Do method -- the same idiom txcache already uses for badgerhold access
// in this codebase.
type badgerHoldStore struct {
	storage *storage.BadgerHoldStorage
}

// NewBadgerHoldStore wraps a BadgerHold-backed storage (as returned by
// storage.ModuleManager.GetNewBadgerHoldStorage) as a txmanager Store.
func NewBadgerHoldStore(s *storage.BadgerHoldStorage) Store {
	return &badgerHoldStore{storage: s}
}

func (b *badgerHoldStore) Upsert(key, data interface{}) (err error) {
	b.storage.Do(func(db *badgerhold.Store) {
		err = db.Upsert(key, data)
	})
	return err
}

func (b *badgerHoldStore) Find(result interface{}, query *badgerhold.Query) (err error) {
	b.storage.Do(func(db *badgerhold.Store) {
		err = db.Find(result, query)
	})
	return err
}
