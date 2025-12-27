package security

import (
	"encoding/json"

	"github.com/ITProLabDev/ethbacknode/storage"
	"github.com/ITProLabDev/ethbacknode/tools/log"
)

const (
	KEY_FORMAT_HEX    = "hex"
	KEY_FORMAT_JSON   = "json"
	KEY_FORMAT_BASE58 = "base58"

	SIGNATURE_TYPE_SHA256 = "SHA256"
	SIGNATURE_TYPE_RIPEMD = "RIPEMD"
	SIGNATURE_TYPE_SHA512 = "SHA512"
)

type Config struct {
	storage       storage.BinStorage
	Debug         bool   `json:"debug"`
	KeyFormat     string `json:"defaultKeyFormat"`
	SignatureType string `json:"signatureType"`
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
		err = c.coldStart()
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
	if !c.storage.IsExists() {
		changed = true
	}
	if c.KeyFormat == "" {
		c.KeyFormat = KEY_FORMAT_HEX
		changed = true
	}
	if c.SignatureType == "" {
		c.SignatureType = SIGNATURE_TYPE_SHA256
		changed = true
	}
	if changed {
		return c.Save()
	}
	return nil
}
