package address

import (
	"module github.com/ITProLabDev/ethbacknode/storage"
	"module github.com/ITProLabDev/ethbacknode/tools/log"
	"sync"
)

type MemPoolOption func(pool *Manager) error

func NewManager(options ...MemPoolOption) (pool *Manager, err error) {
	pool = &Manager{
		allAddresses:  make(map[string]*Address),
		freeAddresses: make(map[string]*Address),
		fastPool:      newAddressMemStore(nullStore{}),
		config: &Config{
			storage: _configDefaultStorage(),
		},
	}
	for _, opt := range options {
		err = opt(pool)
		if err != nil {
			return nil, err
		}
	}
	err = pool.config.Load()
	if err != nil {
		return nil, err
	}
	err = pool.preLoadAddresses()
	if err != nil {
		return nil, err
	}
	pool.checkFreeAddressPool()
	return pool, nil
}

func WithAddressStorage(store storage.SimpleStorage) MemPoolOption {
	return func(pool *Manager) error {
		pool.db = store
		return nil
	}
}

func WithConfigStorage(store storage.BinStorage) MemPoolOption {
	return func(pool *Manager) error {
		pool.config.storage = store
		return nil
	}
}

func WithAddressCodec(codec AddressCodec) MemPoolOption {
	return func(pool *Manager) error {
		pool.addressCodec = codec
		return nil
	}
}

type rawPool map[string]*Address

type Manager struct {
	db            storage.SimpleStorage
	config        *Config
	mux           sync.RWMutex
	fastPool      fastStore
	allAddresses  map[string]*Address
	freeAddresses map[string]*Address
	addressCodec  AddressCodec
}

func (s rawPool) AppendKeys(store []string) []string {
	store = make([]string, len(s))
	i := 0
	var free int
	for _, a := range s {
		store[i] = a.Address
		if !a.Subscribed {
			free++
		}
		i++
	}
	log.Info("Current pool stat:", free, "free addresses from", len(s))
	return store
}

func (s rawPool) Get(key string) *Address {
	return s[key]
}

type nullStore struct {
}

func (n nullStore) AppendKeys([]string) []string {
	return nil
}

func (n nullStore) Get(string) *Address {
	return nil
}
