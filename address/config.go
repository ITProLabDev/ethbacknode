package address

import (
	"backnode/storage"
	"backnode/tools/log"
	"encoding/json"
)

type Config struct {
	storage                 storage.BinStorage
	Debug                   bool `json:"debug"`
	defaultBip44Support     bool
	defaultBip44CoinType    string
	defaultBip36MnemonicLen int
	EnableAddressGenerate   bool   `json:"enableAddressGenerate"`
	MinFreePoolSize         int    `json:"minFreePoolSize"`
	GeneratePoolUpTo        int    `json:"generatePoolUpTo"`
	Bip39Support            bool   `json:"bip39Support"`
	Bip36MnemonicLen        int    `json:"bip36MnemonicLen"`
	Bip44CoinType           string `json:"bip44CoinType"`
	Bip32DerivationPath     string `json:"bip32DerivationPath"`
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
	c.MinFreePoolSize = 100
	c.GeneratePoolUpTo = 110
	c.EnableAddressGenerate = true
	c.Bip36MnemonicLen = 12
	if c.defaultBip44CoinType == "" {
		c.Bip44CoinType = "Ether"
	} else {
		c.Bip44CoinType = c.defaultBip44CoinType
	}
	if c.defaultBip36MnemonicLen == 0 {
		c.Bip36MnemonicLen = 12
	} else {
		c.Bip36MnemonicLen = c.defaultBip36MnemonicLen
	}
	c.Bip39Support = c.defaultBip44Support
	c.Bip32DerivationPath = "m/44'/60'/0'/0/0"
	return c.Save()
}

func (c *Config) checkDefaults() error {
	changed := false
	if c.Bip36MnemonicLen == 0 {
		if c.defaultBip36MnemonicLen == 0 {
			c.Bip36MnemonicLen = 12
		} else {
			c.Bip36MnemonicLen = c.defaultBip36MnemonicLen
		}
		changed = true
	}
	if c.Bip44CoinType == "" {
		if c.defaultBip44CoinType == "" {
			c.Bip44CoinType = "Ether"
		} else {
			c.Bip44CoinType = c.defaultBip44CoinType
		}
		changed = true
	}
	if c.Bip32DerivationPath == "" {
		c.Bip32DerivationPath = "m/44'/60'/0'/0/0"
		changed = true
	}
	if changed {
		return c.Save()
	}
	return nil
}
