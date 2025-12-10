package txcache

import "module github.com/ITProLabDev/ethbacknode/storage"

func WithConfigStorage(storage storage.BinStorage) ManagerOption {
	return func(s *Manager) error {
		s.config.storage = storage
		return nil
	}
}
func WithTxStorage(storage *storage.BadgerHoldStorage) ManagerOption {
	return func(s *Manager) error {
		s.txCache = storage
		return nil
	}
}
