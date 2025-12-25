package security

import "github.com/ITProLabDev/ethbacknode/storage"

func WithStorageManager(storageManager *storage.ModuleManager) Option {
	return func(manager *Manager) {
		manager.storageManager = storageManager
	}
}
