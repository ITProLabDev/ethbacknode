// Package security provides request signing and verification for API authentication.
// It supports multiple key formats and signature algorithms.
package security

import "github.com/ITProLabDev/ethbacknode/storage"

// Option is a function that configures a Manager.
type Option func(*Manager)

// Manager handles request signing and security configuration.
type Manager struct {
	config *Config

	storageManager *storage.ModuleManager
}

// NewManager creates a new security manager with the specified options.
func NewManager(opts ...Option) *Manager {
	manager := &Manager{
		config: &Config{},
	}
	for _, opt := range opts {
		opt(manager)
	}
	return manager
}

// Set applies additional options to the manager.
func (m *Manager) Set(opts ...Option) {
	for _, opt := range opts {
		opt(m)
	}
}

// Init initializes the security manager by loading configuration.
func (m *Manager) Init() (err error) {
	m.config.storage = m.storageManager.GetBinFileStorage("config.json")
	err = m.config.Load()
	if err != nil {
		return err
	}
	return nil
}
