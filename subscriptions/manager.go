// Package subscriptions manages service subscriptions for transaction and block events.
// It handles subscriber registration, event notification, and transaction confirmation tracking.
package subscriptions

import (
	"github.com/ITProLabDev/ethbacknode/address"
	"github.com/ITProLabDev/ethbacknode/storage"
	"github.com/ITProLabDev/ethbacknode/types"
	"sync"
)

// WithAddressManager sets the address manager for looking up subscribed addresses.
func WithAddressManager(pool *address.Manager) Option {
	return func(w *Manager) error {
		w.addressPool = pool
		return nil
	}
}
// WithTransactionStorage sets the storage backend for transaction records.
func WithTransactionStorage(storage *storage.BadgerHoldStorage) Option {
	return func(w *Manager) error {
		w.transactionPool = storage
		return nil
	}
}

// WithBlockchainClient sets the blockchain client for chain queries and transfers.
func WithBlockchainClient(client types.ChainClient) Option {
	return func(w *Manager) error {
		w.blockchainClient = client
		return nil
	}
}

// WithSubscribersStorage sets the storage backend for subscriber data.
func WithSubscribersStorage(storage storage.BinStorage) Option {
	return func(w *Manager) error {
		w.subscribersStorage = storage
		return nil
	}
}

// WithGlobalConfig sets the global application configuration.
func WithGlobalConfig(config types.Config) Option {
	return func(w *Manager) error {
		w.globalConfig = config
		return nil
	}
}

// WithConfigStorage sets the storage backend for subscription configuration.
func WithConfigStorage(storage storage.BinStorage) Option {
	return func(s *Manager) error {
		s.config.storage = storage
		return nil
	}
}

// NewManager creates a new subscription manager with the specified options.
// Loads existing subscriptions and starts the event processing loop.
func NewManager(options ...Option) (*Manager, error) {
	s := &Manager{
		config: &Config{
			storage: _configDefaultStorage(),
		},
		eventPipe: make(chan func()),
	}
	for _, opt := range options {
		err := opt(s)
		if err != nil {
			return nil, err
		}
	}
	err := s.config.Load()
	if err != nil {
		return nil, err
	}
	err = s.subscriptionsLoad()
	if err != nil {
		return nil, err
	}
	go s.eventLoop()
	return s, nil
}

// Option is a function that configures a Manager.
type Option func(w *Manager) error

// Manager handles service subscriptions and event notifications.
// Tracks transactions, sends notifications to subscribers, and manages
// confirmation-based event delivery.
type Manager struct {
	config          *Config
	globalConfig    types.Config
	lastSeenBlock   int
	addressPool     *address.Manager
	transactionPool *storage.BadgerHoldStorage

	blockchainClient types.ChainClient

	subscribersMux     sync.RWMutex
	subscribersStorage storage.BinStorage
	subscribers        map[ServiceId]*Subscription

	eventPipe chan func()
	notifyMux sync.RWMutex
}
