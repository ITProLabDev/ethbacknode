package subscriptions

import (
	"backnode/address"
	"backnode/storage"
	"backnode/types"
	"sync"
)

func WithAddressManager(pool *address.Manager) Option {
	return func(w *Manager) error {
		w.addressPool = pool
		return nil
	}
}
func WithTransactionStorage(storage *storage.BadgerHoldStorage) Option {
	return func(w *Manager) error {
		w.transactionPool = storage
		return nil
	}
}

func WithBlockchainClient(client types.ChainClient) Option {
	return func(w *Manager) error {
		w.blockchainClient = client
		return nil
	}
}

func WithSubscribersStorage(storage storage.BinStorage) Option {
	return func(w *Manager) error {
		w.subscribersStorage = storage
		return nil
	}
}

func WithGlobalConfig(config types.Config) Option {
	return func(w *Manager) error {
		w.globalConfig = config
		return nil
	}
}

func WithConfigStorage(storage storage.BinStorage) Option {
	return func(s *Manager) error {
		s.config.storage = storage
		return nil
	}
}

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

type Option func(w *Manager) error

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
