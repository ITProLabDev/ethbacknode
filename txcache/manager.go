package txcache

import (
	"backnode/storage"
	"sync"
)

type ManagerOption func(*Manager) error

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

type Manager struct {
	config    *Config
	txCache   *storage.BadgerHoldStorage
	eventPipe chan func()
	mux       sync.RWMutex
}

func (m *Manager) eventLoop() {
	for {
		select {
		case event := <-m.eventPipe:
			event()
		}
	}
}
