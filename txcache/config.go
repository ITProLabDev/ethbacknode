package txcache

import (
	"encoding/json"
	"github.com/ITProLabDev/ethbacknode/storage"
	"github.com/ITProLabDev/ethbacknode/tools/log"
)

// NewConfig creates a new configuration with default storage.
func NewConfig() *Config {
	return &Config{
		storage: _configDefaultStorage(),
	}
}

// Config holds the transaction cache configuration.
// Controls confirmation thresholds and storage behavior.
type Config struct {
	storage               storage.BinStorage
	Debug                 bool `json:"debug"`
	Confirmations         int  `json:"confirmations"`
	RegisterConfirmations int  `json:"registerConfirmations"`
	StoreIncomingTx       bool `json:"storeIncomingTx"`
	StoreOutgoingTx       bool `json:"storeOutgoingTx"`
}

// _configDefaultStorage returns the default file-based storage for configuration.
func _configDefaultStorage() storage.BinStorage {
	configStore, err := storage.NewBinFileStorage("Config", "data", "txcache", "config.json")
	if err != nil {
		log.Error("Can not get default config storage:", err)
	}
	return configStore
}

// Load reads the configuration from storage.
func (c *Config) Load() (err error) {
	if !c.storage.IsExists() {
		err = c.coldStart()
		if err != nil {
			return err
		}
	}
	jsonBytes, err := c.storage.Load()
	if err != nil {
		return
	}
	err = json.Unmarshal(jsonBytes, c)
	return
}

// Save persists the configuration to storage as JSON.
func (c *Config) Save() (err error) {
	data, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return
	}
	err = c.storage.Save(data)
	return
}

// coldStart initializes the configuration with default values.
// Sets Confirmations=20 and RegisterConfirmations=50.
func (c *Config) coldStart() (err error) {
	if c.storage == nil {
		return ErrConfigStorageEmpty
	}
	c.Debug = false
	c.StoreIncomingTx = true
	c.StoreOutgoingTx = true
	c.Confirmations = 20
	c.RegisterConfirmations = 50
	return c.Save()
}
