package ethclient

import (
	"encoding/json"
	"github.com/ITProLabDev/ethbacknode/storage"
	"github.com/ITProLabDev/ethbacknode/tools/log"
	"github.com/ITProLabDev/ethbacknode/types"
)

type Config struct {
	storage       storage.BinStorage
	ChainName     string
	ChainId       string
	ChainSymbol   string
	Decimals      int
	Confirmations int  `json:"confirmations"`
	Debug         bool `json:"debug"`
	Tokens        []*types.TokenInfo
}

func _configDefaultStorage() storage.BinStorage {
	configStore, err := storage.NewBinFileStorage("Config", "data", "client", "config.json")
	if err != nil {
		log.Error("Can not get default config storage:", err)
	}
	return configStore
}

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
	// Default cold-start identity is the Ethereum profile (preserves the
	// historical behavior). `--init <chain>` selects a different profile.
	c.ApplyProfile(ethProfile())
	return c.Save()
}
