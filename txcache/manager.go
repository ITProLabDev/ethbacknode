// Package txcache provides transaction caching and confirmation tracking.
// It stores transaction records with confirmation counts and provides
// query APIs for retrieving transactions by hash or address.
package txcache

import (
	"github.com/ITProLabDev/ethbacknode/storage"
	"sync"
)

// ManagerOption is a function that configures a Manager.
type ManagerOption func(*Manager) error

// NewManager creates a new transaction cache manager with the specified options.
// Loads configuration and starts the event processing loop.
func NewManager(options ...ManagerOption) (*Manager, error) {
	manager := &Manager{
		config:    NewConfig(),
		eventPipe: make(chan func(), 16),
	}
	for _, opt := range options {
		err := opt(manager)
		if err != nil {
			return nil, err
		}
	}
	err := manager.config.Load()
	if err != nil {
		return nil, err
	}
	go manager.eventLoop()
	return manager, nil
}

// Manager manages the transaction cache.
// Stores transactions with confirmation tracking and provides query APIs.
type Manager struct {
	config    *Config
	txCache   *storage.BadgerHoldStorage
	eventPipe chan func()
	mux       sync.RWMutex
}

// eventLoop processes events from the event pipe sequentially.
func (m *Manager) eventLoop() {
	for {
		select {
		case event := <-m.eventPipe:
			event()
		}
	}
}
