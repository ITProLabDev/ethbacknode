package watchdog

import (
	"encoding/json"
	"github.com/ITProLabDev/ethbacknode/storage"
	"github.com/ITProLabDev/ethbacknode/tools/log"
)

// Config holds the watchdog service configuration.
// Controls polling behavior, confirmations, and debug logging.
type Config struct {
	storage             storage.BinStorage
	Run                 bool  `json:"run"`
	PullInterval        int   `json:"pullInterval"`
	PullByExternalEvent bool  `json:"pullByExternalEvent"`
	PullByTimer         bool  `json:"pullByTimer"`
	Confirmations       int64 `json:"confirmations"`
	Debug               bool  `json:"debug"`
}

// _configDefaultStorage returns the default file-based storage for configuration.
func _configDefaultStorage() storage.BinStorage {
	configStore, err := storage.NewBinFileStorage("Config", "data", "watchdog", "config.json")
	if err != nil {
		log.Error("Can not get default config storage:", err)
	}
	return configStore
}

// Load reads the configuration from storage.
// Performs cold start with defaults if no config exists.
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
// Sets Run=true, PullInterval=5 seconds, Confirmations=7.
func (c *Config) coldStart() (err error) {
	if c.storage == nil {
		return ErrConfigStorageEmpty
	}
	c.Run = true
	c.PullInterval = 5
	c.Confirmations = 7
	return c.Save()
}
