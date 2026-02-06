package txcache

import "github.com/ITProLabDev/ethbacknode/storage"

// WithConfigStorage sets the storage backend for configuration.
func WithConfigStorage(storage storage.BinStorage) ManagerOption {
	return func(s *Manager) error {
		s.config.storage = storage
		return nil
	}
}
// WithTxStorage sets the BadgerHold storage for transaction records.
func WithTxStorage(storage *storage.BadgerHoldStorage) ManagerOption {
	return func(s *Manager) error {
		s.txCache = storage
		return nil
	}
}
