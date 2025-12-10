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
	c.ChainName = "Ethereum"
	c.ChainId = "ethereum"
	c.ChainSymbol = "ETH"
	c.Decimals = 18
	c.Confirmations = 20
	c.Tokens = []*types.TokenInfo{
		{
			Name:            "TetherToken",
			Symbol:          "USDT",
			Decimals:        6,
			ContractAddress: "0xdAC17F958D2ee523a2206206994597C13D831ec7",
			Protocol:        "TRC20",
		},
		{
			Name:            "USD Coin",
			Symbol:          "USDC",
			Decimals:        6,
			ContractAddress: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
			Protocol:        "TRC20",
		},
	}
	return c.Save()
}
