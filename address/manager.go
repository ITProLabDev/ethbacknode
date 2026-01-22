package address

import (
	"github.com/ITProLabDev/ethbacknode/storage"
	"github.com/ITProLabDev/ethbacknode/tools/log"
	"sync"
)

// MemPoolOption is a function that configures a Manager.
type MemPoolOption func(pool *Manager) error

// NewManager creates a new address manager with the specified options.
// Loads existing addresses from storage and initializes the free address pool.
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

// WithAddressStorage sets the storage backend for addresses.
func WithAddressStorage(store storage.SimpleStorage) MemPoolOption {
	return func(pool *Manager) error {
		pool.db = store
		return nil
	}
}

// WithConfigStorage sets the storage backend for configuration.
func WithConfigStorage(store storage.BinStorage) MemPoolOption {
	return func(pool *Manager) error {
		pool.config.storage = store
		return nil
	}
}

// WithAddressCodec sets the address encoder/decoder.
func WithAddressCodec(codec AddressCodec) MemPoolOption {
	return func(pool *Manager) error {
		pool.addressCodec = codec
		return nil
	}
}

// rawPool is a map type for address storage.
type rawPool map[string]*Address

// Manager manages a pool of blockchain addresses.
// It maintains separate pools for all addresses and free (unsubscribed) addresses.
// Thread-safe for concurrent access.
type Manager struct {
	db            storage.SimpleStorage  // Persistent storage for addresses
	config        *Config                // Manager configuration
	mux           sync.RWMutex           // Mutex for thread-safe access
	fastPool      fastStore              // Fast in-memory address lookup
	allAddresses  map[string]*Address    // All managed addresses
	freeAddresses map[string]*Address    // Unsubscribed addresses available for use
	addressCodec  AddressCodec           // Address encoder/decoder
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
