package security

import (
	"encoding/json"

	"github.com/ITProLabDev/ethbacknode/storage"
	"github.com/ITProLabDev/ethbacknode/tools/log"
)

// Key format and signature type constants.
const (
	// KEY_FORMAT_HEX indicates hex-encoded keys.
	KEY_FORMAT_HEX = "hex"
	// KEY_FORMAT_JSON indicates JSON-formatted keys.
	KEY_FORMAT_JSON = "json"
	// KEY_FORMAT_BASE58 indicates Base58-encoded keys.
	KEY_FORMAT_BASE58 = "base58"

	// SIGNATURE_TYPE_SHA256 indicates SHA-256 hashing.
	SIGNATURE_TYPE_SHA256 = "SHA256"
	// SIGNATURE_TYPE_RIPEMD indicates RIPEMD-160 hashing.
	SIGNATURE_TYPE_RIPEMD = "RIPEMD"
	// SIGNATURE_TYPE_SHA512 indicates SHA-512 hashing.
	SIGNATURE_TYPE_SHA512 = "SHA512"
)

// Config holds the security manager configuration.
type Config struct {
	storage       storage.BinStorage
	Debug         bool   `json:"debug"`
	KeyFormat     string `json:"defaultKeyFormat"`
	SignatureType string `json:"signatureType"`
}

// _configDefaultStorage returns the default file-based storage for configuration.
func _configDefaultStorage() storage.BinStorage {
	configStore, err := storage.NewBinFileStorage("Config", "data", "addresspool", "config.json")
	if err != nil {
		log.Error("Can not get default config storage:", err)
	}
	return configStore
}

// Load reads the configuration from storage.
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
func (c *Config) coldStart() (err error) {
	if c.storage == nil {
		return ErrConfigStorageEmpty
	}
	return c.checkDefaults()
}

// checkDefaults validates and applies default values for missing configuration.
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
