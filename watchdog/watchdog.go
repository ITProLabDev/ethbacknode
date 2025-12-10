package watchdog

import (
	"module github.com/ITProLabDev/ethbacknode/address"
	"module github.com/ITProLabDev/ethbacknode/storage"
	"module github.com/ITProLabDev/ethbacknode/types"
	"sync"
)

type ServiceOption func(s *Service)

func WithAddressManager(pool *address.Manager) ServiceOption {
	return func(s *Service) {
		s.addressPool = pool
	}
}
func WithConfigStorage(storage storage.BinStorage) ServiceOption {
	return func(s *Service) {
		s.config.storage = storage
	}
}
func WithStateStorage(storage storage.BinStorage) ServiceOption {
	return func(s *Service) {
		s.state.storage = storage
	}
}
func WithClient(client types.ChainClient) ServiceOption {
	return func(s *Service) {
		s.client = client
	}
}

func SetLastStateTo(newState int64) ServiceOption {
	return func(s *Service) {
		s.state.setToBlock = true
		s.state.setToBlockNum = newState
	}
}

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
