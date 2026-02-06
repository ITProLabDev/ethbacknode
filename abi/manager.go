// Package abi provides smart contract ABI encoding/decoding for ERC-20 tokens.
// It manages known smart contracts and supports ERC-20 call data encoding.
package abi

import (
	"encoding/json"
	"github.com/ITProLabDev/ethbacknode/address"
	"github.com/ITProLabDev/ethbacknode/storage"
	"github.com/ITProLabDev/ethbacknode/tools/log"
	"strings"
	"sync"
)

// Option is a function that configures a SmartContractsManager.
type Option func(*SmartContractsManager)

// WithStorage sets the storage backend for known contracts.
func WithStorage(storage storage.BinStorage) Option {
	return func(m *SmartContractsManager) {
		m.storage = storage
	}
}
// WithAddressCodec sets the address encoder/decoder for ABI encoding.
func WithAddressCodec(codec address.AddressCodec) Option {
	return func(m *SmartContractsManager) {
		m.addressCodec = codec
	}
}

// NewManager creates a new smart contracts manager with the specified options.
func NewManager(options ...Option) *SmartContractsManager {
	manager := &SmartContractsManager{
		storage:   _smartContractsDefaultStorage(),
		bySymbol:  make(map[string]*SmartContractInfo),
		byName:    make(map[string]*SmartContractInfo),
		byAddress: make(map[string]*SmartContractInfo),
	}
	for _, option := range options {
		option(manager)
	}
	return manager
}

// SmartContractsManager manages known smart contracts and provides ABI encoding.
// Maintains lookup maps by symbol, name, and address for efficient queries.
type SmartContractsManager struct {
	mux          sync.RWMutex
	storage      storage.BinStorage
	contracts    []*SmartContractInfo
	bySymbol     map[string]*SmartContractInfo
	byName       map[string]*SmartContractInfo
	byAddress    map[string]*SmartContractInfo
	erc20abi     *SmartContractAbi
	addressCodec address.AddressCodec
}

func _smartContractsDefaultStorage() storage.BinStorage {
	configStore, err := storage.NewBinFileStorage("Config", "Data", "abi", "known_contracts.json")
	if err != nil {
		log.Error("Can not get default config storage:", err)
	}
	return configStore
}

func (m *SmartContractsManager) afterLoad() {
	for _, c := range m.contracts {
		m.byName[c.Name] = c
		m.byAddress[strings.ToLower(c.ContractAddress)] = c
	}
}

func (m *SmartContractsManager) addUnsafe(c *SmartContractInfo) {
	for _, ec := range m.contracts {
		if ec.Name == c.Name && ec.ContractAddress == c.ContractAddress {
			return
		} else if ec.Name == c.Name {
			log.Critical("Duplicated Contract Name:", c.Name)
			return
		} else if ec.ContractAddress == c.ContractAddress {
			log.Critical("Duplicated Contract AddressBytes:", c.ContractAddress)
			return
		}
	}
	m.contracts = append(m.contracts, c)
	m.afterLoad()
}

// Init initializes the manager by loading ERC-20 ABI and known contracts.
func (m *SmartContractsManager) Init() error {
	if m.storage == nil {
		return ErrConfigStorageEmpty
	}
	if !m.storage.IsExists() {
		err := m.ColdStart()
		if err != nil {
			return err
		}
	}
	err := json.Unmarshal([]byte(erc20tpl), &m.erc20abi)
	if err != nil {
		return err
	}
	return m.Load()
}
// Add adds a smart contract to the registry. Thread-safe.
func (m *SmartContractsManager) Add(c *SmartContractInfo) {
	m.mux.Lock()
	m.addUnsafe(c)
	m.mux.Unlock()
	m.Save()
}

// Load reads the known contracts from storage.
func (m *SmartContractsManager) Load() (err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	jsonBytes, err := m.storage.Load()
	if err != nil {
		return
	}
	err = json.Unmarshal(jsonBytes, &m.contracts)
	m.afterLoad()
	return
}

// Save persists the known contracts to storage.
func (m *SmartContractsManager) Save() (err error) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	data, err := json.MarshalIndent(m.contracts, "", " ")
	if err != nil {
		return err
	}
	return m.storage.Save(data)
}

// ColdStart initializes the contracts list with built-in defaults.
func (m *SmartContractsManager) ColdStart() (err error) {
	if m.storage == nil {
		return ErrConfigStorageEmpty
	}
	var contracts []*SmartContractInfo
	err = json.Unmarshal([]byte(rawKnownContractTpl), &contracts)
	if err != nil {
		return err
	}
	for _, c := range contracts {
		m.addUnsafe(c)
	}
	if err != nil {
		return err
	}
	m.afterLoad()
	return m.Save()
}

// Walk iterates over all known contracts with a read lock.
func (m *SmartContractsManager) Walk(view func(c *SmartContractInfo)) {
	m.mux.RLock()
	for _, c := range m.contracts {
		view(c)
	}
	m.mux.RUnlock()
}

// GetSmartContractAddressByName finds a contract address by its name.
func (m *SmartContractsManager) GetSmartContractAddressByName(contractName string) (contractAddress string, err error) {
	m.Walk(func(c *SmartContractInfo) {
		if c.Name == contractName {
			contractAddress = c.ContractAddress
		}
	})
	if contractAddress == "" {
		err = ErrUnknownContract
	}
	return contractAddress, err
}

// GetSmartContractAddressByToken finds a contract address by its token symbol.
func (m *SmartContractsManager) GetSmartContractAddressByToken(symbol string) (contractAddress string, err error) {
	//symbol = strings.ToLower(symbol)
	m.Walk(func(c *SmartContractInfo) {
		//log.Debug("Check", c.Symbol, symbol)
		if c.Symbol == symbol {
			contractAddress = c.ContractAddress
		}
	})
	if contractAddress == "" {
		err = ErrUnknownContract
	}
	return contractAddress, err
}

// GetSmartContractByToken finds a contract by its token symbol.
func (m *SmartContractsManager) GetSmartContractByToken(symbol string) (contract *SmartContractInfo, err error) {
	symbol = strings.ToLower(symbol)
	m.Walk(func(c *SmartContractInfo) {
		if c.Symbol == symbol {
			contract = c
		}
	})
	if contract == nil {
		err = ErrUnknownContract
	}
	return contract, err
}

// GetSmartContractByAddress finds a contract by its address.
func (m *SmartContractsManager) GetSmartContractByAddress(contractAddress string) (contract *SmartContractInfo, err error) {
	var found bool
	contractAddress = strings.ToLower(contractAddress)
	if contract, found = m.byAddress[contractAddress]; !found {
		return nil, ErrUnknownContract
	}
	return contract, nil
}

// GetSmartContractList returns a map of contract names to addresses.
func (m *SmartContractsManager) GetSmartContractList() (list map[string]string) {
	list = make(map[string]string)
	for _, c := range m.contracts {
		list[c.Name] = c.ContractAddress
	}
	return list
}
