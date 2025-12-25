package security

import "github.com/ITProLabDev/ethbacknode/storage"

type Option func(*Manager)

type Manager struct {
	config *Config

	storageManager *storage.ModuleManager
}

func NewManager(opts ...Option) *Manager {
	manager := &Manager{
		config: &Config{},
	}
	for _, opt := range opts {
		opt(manager)
	}
	return manager
}

func (m *Manager) Set(opts ...Option) {
	for _, opt := range opts {
		opt(m)
	}
}

func (m *Manager) Init() (err error) {
	m.config.storage = m.storageManager.GetBinFileStorage("config.json")
	err = m.config.Load()
	if err != nil {
		return err
	}
	return nil
}
