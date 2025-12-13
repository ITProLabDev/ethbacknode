package authorization

import (
	"encoding/json"

	"github.com/ITProLabDev/ethbacknode/storage"
	"github.com/ITProLabDev/ethbacknode/tools/log"
)

type Config struct {
	storage storage.BinStorage
	Debug   bool `json:"debug"`
}

func _configDefaultStorage() storage.BinStorage {
	configStore, err := storage.NewBinFileStorage("Config", "data", "addresspool", "config.json")
	if err != nil {
		log.Error("Can not get default config storage:", err)
	}
	return configStore
}

func (c *Config) Load() (err error) {
	if !c.storage.IsExists() {
		c.coldStart()
	}
	jsonBytes, err := c.storage.Load()
	if err != nil {
		return
	}
	err = json.Unmarshal(jsonBytes, c)
	return c.checkDefaults()
}

func (c *Config) Save() (err error) {
	data, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return
	}
	err = c.storage.Save(data)
	return
}

func (c *Config) coldStart() (err error) {
	if c.storage == nil {
		return ErrConfigStorageEmpty
	}
	return c.checkDefaults()
}

func (c *Config) checkDefaults() error {
	changed := false
	//todo
	if changed {
		return c.Save()
	}
	return nil
}
