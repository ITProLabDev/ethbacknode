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
	mux sync.RWMutex

	// saveMu serializes persistence. mux is an RWMutex, so two concurrent Saves
	// would both hold it for reading and write the file at the same time; this
	// makes the writes sequential and each one whole.
	saveMu sync.Mutex

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
	m.byName = make(map[string]*SmartContractInfo, len(m.contracts))
	m.byAddress = make(map[string]*SmartContractInfo, len(m.contracts))
	m.bySymbol = make(map[string]*SmartContractInfo, len(m.contracts))
	for _, c := range m.contracts {
		m.byName[c.Name] = c
		m.byAddress[strings.ToLower(c.ContractAddress)] = c
		m.bySymbol[c.Symbol] = c
	}
}

// addUnsafe registers c unless it collides with something already present.
// The caller must hold mux for writing.
//
// Address matching is case-insensitive, matching how byAddress is keyed and
// how GetSmartContractByAddress looks up: an EIP-55 checksummed address and
// its lowercase form name the same contract, and registering both must not
// produce two entries.
func (m *SmartContractsManager) addUnsafe(c *SmartContractInfo) {
	addressKey := strings.ToLower(c.ContractAddress)

	if existing, found := m.byAddress[addressKey]; found {
		if existing.Name == c.Name {
			return // already registered; re-adding is a no-op
		}
		log.Critical("Duplicated Contract AddressBytes:", c.ContractAddress)
		return
	}
	if _, found := m.byName[c.Name]; found {
		log.Critical("Duplicated Contract Name:", c.Name)
		return
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
	if m.storage == nil {
		return ErrConfigStorageEmpty
	}
	jsonBytes, err := m.storage.Load()
	if err != nil {
		return
	}
	var loaded []*SmartContractInfo
	if err = json.Unmarshal(jsonBytes, &loaded); err != nil {
		return err
	}
	m.contracts = loaded
	m.afterLoad()
	return nil
}

// Save persists the known contracts to storage. Concurrent calls are
// serialized, so the stored file is always one complete snapshot rather than
// two interleaved ones.
func (m *SmartContractsManager) Save() (err error) {
	if m.storage == nil {
		return ErrConfigStorageEmpty
	}
	m.saveMu.Lock()
	defer m.saveMu.Unlock()

	// Snapshot under the read lock, write outside it: encoding must see a
	// stable slice, but the write itself does not need to block readers.
	m.mux.RLock()
	data, err := json.MarshalIndent(m.contracts, "", " ")
	m.mux.RUnlock()
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
	m.mux.Lock()
	for _, c := range contracts {
		m.addUnsafe(c)
	}
	m.afterLoad()
	m.mux.Unlock()
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

// The three lookups below go through the byName / bySymbol indexes that
// afterLoad maintains, rather than walking the whole contracts slice under a
// read lock. Semantics are unchanged, including which entry wins for a
// repeated name/symbol: afterLoad populates the maps in slice order, so the
// later entry overwrites the earlier one either way.

// GetSmartContractAddressByName finds a contract address by its name.
func (m *SmartContractsManager) GetSmartContractAddressByName(contractName string) (contractAddress string, err error) {
	m.mux.RLock()
	c, found := m.byName[contractName]
	m.mux.RUnlock()
	if !found {
		return "", ErrUnknownContract
	}
	return c.ContractAddress, nil
}

// GetSmartContractAddressByToken finds a contract address by its token symbol.
func (m *SmartContractsManager) GetSmartContractAddressByToken(symbol string) (contractAddress string, err error) {
	m.mux.RLock()
	c, found := m.bySymbol[symbol]
	m.mux.RUnlock()
	if !found {
		return "", ErrUnknownContract
	}
	return c.ContractAddress, nil
}

// GetSmartContractByToken finds a contract by its token symbol.
func (m *SmartContractsManager) GetSmartContractByToken(symbol string) (contract *SmartContractInfo, err error) {
	m.mux.RLock()
	c, found := m.bySymbol[symbol]
	m.mux.RUnlock()
	if !found {
		return nil, ErrUnknownContract
	}
	return c, nil
}

// GetSmartContractByAddress finds a contract by its address.
func (m *SmartContractsManager) GetSmartContractByAddress(contractAddress string) (contract *SmartContractInfo, err error) {
	contractAddress = strings.ToLower(contractAddress)
	m.mux.RLock()
	contract, found := m.byAddress[contractAddress]
	m.mux.RUnlock()
	if !found {
		return nil, ErrUnknownContract
	}
	return contract, nil
}

// GetSmartContractList returns a map of contract names to addresses.
func (m *SmartContractsManager) GetSmartContractList() (list map[string]string) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	list = make(map[string]string, len(m.contracts))
	for _, c := range m.contracts {
		list[c.Name] = c.ContractAddress
	}
	return list
}
