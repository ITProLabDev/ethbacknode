package security

import "github.com/ITProLabDev/ethbacknode/storage"

// WithStorageManager sets the storage manager for configuration persistence.
func WithStorageManager(storageManager *storage.ModuleManager) Option {
	return func(manager *Manager) {
		manager.storageManager = storageManager
	}
}
