package abi

import (
	"encoding/json"
	"github.com/ITProLabDev/ethbacknode/address"
	"github.com/ITProLabDev/ethbacknode/storage"
	"github.com/ITProLabDev/ethbacknode/tools/log"
	"strings"
	"sync"
)

type Option func(*SmartContractsManager)

func WithStorage(storage storage.BinStorage) Option {
	return func(m *SmartContractsManager) {
		m.storage = storage
	}
}
func WithAddressCodec(codec address.AddressCodec) Option {
	return func(m *SmartContractsManager) {
		m.addressCodec = codec
	}
}

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
func (m *SmartContractsManager) Add(c *SmartContractInfo) {
	m.mux.Lock()
	m.addUnsafe(c)
	m.mux.Unlock()
	m.Save()
}

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

func (m *SmartContractsManager) Save() (err error) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	data, err := json.MarshalIndent(m.contracts, "", " ")
	if err != nil {
		return err
	}
	return m.storage.Save(data)
}

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

func (m *SmartContractsManager) Walk(view func(c *SmartContractInfo)) {
	m.mux.RLock()
	for _, c := range m.contracts {
		view(c)
	}
	m.mux.RUnlock()
}

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

func (m *SmartContractsManager) GetSmartContractByAddress(contractAddress string) (contract *SmartContractInfo, err error) {
	var found bool
	contractAddress = strings.ToLower(contractAddress)
	if contract, found = m.byAddress[contractAddress]; !found {
		return nil, ErrUnknownContract
	}
	return contract, nil
}

func (m *SmartContractsManager) GetSmartContractList() (list map[string]string) {
	list = make(map[string]string)
	for _, c := range m.contracts {
		list[c.Name] = c.ContractAddress
	}
	return list
}
