// Package watchdog provides blockchain monitoring services for detecting
// transactions involving managed addresses. It polls the blockchain for new blocks
// and mempool transactions, firing events when relevant addresses are involved.
package watchdog

import (
	"github.com/ITProLabDev/ethbacknode/address"
	"github.com/ITProLabDev/ethbacknode/storage"
	"github.com/ITProLabDev/ethbacknode/types"
	"sync"
)

// ServiceOption is a function that configures a Service.
type ServiceOption func(s *Service)

// WithAddressManager sets the address manager for checking known addresses.
func WithAddressManager(pool *address.Manager) ServiceOption {
	return func(s *Service) {
		s.addressPool = pool
	}
}
// WithConfigStorage sets the storage backend for watchdog configuration.
func WithConfigStorage(storage storage.BinStorage) ServiceOption {
	return func(s *Service) {
		s.config.storage = storage
	}
}
// WithStateStorage sets the storage backend for watchdog state persistence.
func WithStateStorage(storage storage.BinStorage) ServiceOption {
	return func(s *Service) {
		s.state.storage = storage
	}
}
// WithClient sets the blockchain client for querying blocks and transactions.
func WithClient(client types.ChainClient) ServiceOption {
	return func(s *Service) {
		s.client = client
	}
}

// SetLastStateTo overrides the last processed block number on startup.
// Useful for reprocessing historical blocks.
func SetLastStateTo(newState int64) ServiceOption {
	return func(s *Service) {
		s.state.setToBlock = true
		s.state.setToBlockNum = newState
	}
}

// NewService creates a new watchdog service with the specified options.
func NewService(options ...ServiceOption) *Service {
	service := &Service{
		config: &Config{
			storage: _configDefaultStorage(),
		},
		state:         new(lastState),
		events:        make(chan *event, 250),
		maxRetryCount: 3,
	}
	for _, option := range options {
		option(service)
	}
	return service
}

// Service is the main watchdog service that monitors the blockchain.
// It polls for new blocks and mempool content, detecting transactions
// involving managed addresses and firing events for subscribers.
type Service struct {
	run                     bool
	checkInterval           int
	externalEvent           bool
	externalPullEventChanel chan *PullEvent
	state                   *lastState
	mux                     sync.RWMutex
	config                  *Config
	addressPool             *address.Manager
	client                  types.ChainClient
	blockEventHandlers      []BlockEvent
	transactionHandlers     []TransactionEvent
	//jsVm                    *goja.Runtime
	events           chan *event
	pullEventChannel chan *PullEvent
	maxRetryCount    int
	quit             chan struct{}
}

// Run starts the watchdog service.
// Validates configuration, loads state, and starts monitoring loops.
// Returns an error if required dependencies are not configured.
func (w *Service) Run() (err error) {
	if w.client == nil {
		return ErrChainClientNotSet
	}
	if w.addressPool == nil {
		return ErrAddressPoolNotSet
	}
	if w.config.storage == nil {
		return ErrConfigStorageEmpty
	}
	err = w.config.Load()
	if err != nil {
		return err
	}
	w.run = w.config.Run
	w.checkInterval = w.config.PullInterval
	w.externalEvent = w.config.PullByExternalEvent
	if !w.state.setToBlock {
		err = w.state.Load()
		if err != nil {
			return err
		}
	} else {
		err = w.state.UpdateState(w.state.setToBlockNum)
		if err != nil {
			return err
		}
	}
	go w.runLoop()
	go w.eventLoop()
	return nil
}
