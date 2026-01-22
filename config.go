package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/ITProLabDev/ethbacknode/storage"
	"github.com/ITProLabDev/ethbacknode/tools/log"
)

// Error definitions for configuration operations.
var (
	// ErrConfigStorageEmpty is returned when attempting to save config without storage.
	ErrConfigStorageEmpty = errors.New("config storage not set")
)

// Config holds the global application configuration.
// It supports both HCL (primary) and JSON (legacy) formats.
type Config struct {
	storage           storage.BinStorage `json:"-"`
	NodeUrl           string             `json:"nodeUrl" hcl:"nodeUrl,attr"`
	NodePort          string             `json:"nodePort" hcl:"nodePort,attr"`
	NodeUseSSl        bool               `json:"nodeUseSSL" hcl:"nodeUseSSL,attr"`
	NodeUseIPC        bool               `json:"nodeUseIPC" hcl:"nodeUseIPC,attr"`
	NodeIPCSocket     string             `json:"nodeIPCSocket" hcl:"nodeIPCSocket,attr"`
	RpcAddress        string             `json:"rpcAddress" hcl:"rpcAddress,attr"`
	RpcPort           string             `json:"rpcPort" hcl:"rpcPort,attr"`
	DataPath          string             `json:"dataPath" hcl:"dataPath,attr"`
	DebugMode         bool               `json:"debug_mode" hcl:"debugMode,attr"`
	ParamsFlags       map[string]bool    `json:"flags" hcl:"flags,optional"`
	ParamsString      map[string]string  `json:"paramsString" hcl:"paramsString,optional"`
	ParamsInt         map[string]int     `json:"paramsInt" hcl:"paramsInt,optional"`
	AdditionalHeaders map[string]string  `json:"additionalHeaders" hcl:"additionalHeaders,optional"`
	BurnAddress       string             `json:"burnAddress" hcl:"burnAddress,attr"`
}

// _configDefaultStorage creates and returns the default configuration storage.
// It uses BinFileStorage with the global config path.
func _configDefaultStorage() storage.BinStorage {
	configStore, err := storage.NewBinFileStorage("Config", ".", ".", globalConfigPath)
	if err != nil {
		log.Error("Can not get default config storage:", err)
	}
	return configStore
}

// Load reads and parses the configuration from storage.
// It auto-detects the format (HCL or JSON) by examining the first character.
// JSON files are supported for backward compatibility but will be converted
// to HCL format on the next save operation.
func (c *Config) Load() (err error) {
	if !c.storage.IsExists() {
		err = c.coldStart()
		if err != nil {
			return err
		}
	}
	configBytes, err := c.storage.Load()
	if err != nil {
		return
	}

	// Detect format by content (first non-whitespace character)
	// JSON files typically start with '{' or '['
	trimmedBytes := trimWhitespace(configBytes)
	if len(trimmedBytes) > 0 && (trimmedBytes[0] == '{' || trimmedBytes[0] == '[') {
		// JSON format detected - parse as JSON for backward compatibility
		log.Warning("JSON config format detected. Please migrate to HCL format.")
		err = json.Unmarshal(configBytes, c)
		if err != nil {
			return fmt.Errorf("failed to parse JSON config: %w", err)
		}
		// Auto-convert to HCL on next save
		log.Info("Config loaded from JSON. Will be saved as HCL on next save.")
		return nil
	}

	// Try to parse as HCL
	err = hclsimple.Decode("config.hcl", configBytes, nil, c)
	if err != nil {
		return fmt.Errorf("failed to parse HCL config: %w", err)
	}

	return
}

// trimWhitespace removes leading whitespace characters (space, tab, newline, carriage return)
// from the beginning of a byte slice. Used for format detection.
func trimWhitespace(data []byte) []byte {
	for i, b := range data {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			return data[i:]
		}
	}
	return data
}

// Save persists the configuration to storage in HCL format.
// It writes all configuration fields including optional maps (flags, params, headers).
// Returns ErrConfigStorageEmpty if storage is not initialized.
func (c *Config) Save() (err error) {
	if c.storage == nil {
		return ErrConfigStorageEmpty
	}

	f := hclwrite.NewEmptyFile()
	body := f.Body()

	// Set scalar attributes
	body.SetAttributeValue("nodeUrl", cty.StringVal(c.NodeUrl))
	body.SetAttributeValue("nodePort", cty.StringVal(c.NodePort))
	body.SetAttributeValue("nodeUseSSL", cty.BoolVal(c.NodeUseSSl))
	body.SetAttributeValue("nodeUseIPC", cty.BoolVal(c.NodeUseIPC))
	body.SetAttributeValue("nodeIPCSocket", cty.StringVal(c.NodeIPCSocket))
	body.SetAttributeValue("rpcAddress", cty.StringVal(c.RpcAddress))
	body.SetAttributeValue("rpcPort", cty.StringVal(c.RpcPort))
	body.SetAttributeValue("dataPath", cty.StringVal(c.DataPath))
	body.SetAttributeValue("debugMode", cty.BoolVal(c.DebugMode))
	body.SetAttributeValue("burnAddress", cty.StringVal(c.BurnAddress))

	// Set optional maps
	if len(c.ParamsFlags) > 0 {
		flagMap := make(map[string]cty.Value)
		for k, v := range c.ParamsFlags {
			flagMap[k] = cty.BoolVal(v)
		}
		body.SetAttributeValue("flags", cty.MapVal(flagMap))
	}
	if len(c.ParamsString) > 0 {
		stringMap := make(map[string]cty.Value)
		for k, v := range c.ParamsString {
			stringMap[k] = cty.StringVal(v)
		}
		body.SetAttributeValue("paramsString", cty.MapVal(stringMap))
	}
	if len(c.ParamsInt) > 0 {
		intMap := make(map[string]cty.Value)
		for k, v := range c.ParamsInt {
			intMap[k] = cty.NumberIntVal(int64(v))
		}
		body.SetAttributeValue("paramsInt", cty.MapVal(intMap))
	}
	if len(c.AdditionalHeaders) > 0 {
		headerMap := make(map[string]cty.Value)
		for k, v := range c.AdditionalHeaders {
			headerMap[k] = cty.StringVal(v)
		}
		body.SetAttributeValue("additionalHeaders", cty.MapVal(headerMap))
	}

	data := hclwrite.Format(f.Bytes())
	err = c.storage.Save(data)
	return
}

// coldStart initializes the configuration with default values when no config file exists.
// It sets up default connection parameters and saves the initial configuration.
// Default settings: localhost:8545 (HTTP-RPC), localhost:21080 (endpoint).
func (c *Config) coldStart() (err error) {
	if c.storage == nil {
		return ErrConfigStorageEmpty
	}
	c.NodeUrl = "localhost"
	c.NodePort = "8545"
	c.NodeUseIPC = false
	c.NodeIPCSocket = ""
	c.RpcAddress = "localhost"
	c.RpcPort = "21080"
	c.DataPath = "data"
	c.AdditionalHeaders = map[string]string{
		"X-Client": APP_NAME + "/" + APP_VERSION,
	}
	return c.Save()
}

// Flag retrieves a boolean flag value from configuration by name.
// If the flag doesn't exist, it initializes it to false and auto-saves the config.
// This allows dynamic flag registration at runtime.
func (c *Config) Flag(name string) bool {
	var changed bool
	defer func() {
		if changed {
			err := c.Save()
			if err != nil {
				log.Error("Can not save config:", err)
			}
		}
	}()
	if c.ParamsFlags == nil {
		changed = true
		c.ParamsFlags = make(map[string]bool)
		c.ParamsFlags[name] = false
		return false
	}
	if value, ok := c.ParamsFlags[name]; !ok {
		c.ParamsFlags[name] = false
		changed = true
		return false
	} else {
		return value
	}
}

// String retrieves a string parameter value from configuration by name.
// If the parameter doesn't exist, it initializes it with defaultValue and auto-saves.
// This allows dynamic parameter registration at runtime.
func (c *Config) String(flagName string, defaultValue string) string {
	var changed bool
	defer func() {
		if changed {
			err := c.Save()
			if err != nil {
				log.Error("Can not save config:", err)
			}
		}
	}()
	if c.ParamsString == nil {
		changed = true
		c.ParamsString = make(map[string]string)
		c.ParamsString[flagName] = defaultValue
		return defaultValue
	}
	if value, ok := c.ParamsString[flagName]; !ok {
		c.ParamsString[flagName] = defaultValue
		changed = true
		return defaultValue
	} else {
		return value
	}
}

// Int retrieves an integer parameter value from configuration by name.
// If the parameter doesn't exist, it initializes it with defaultValue and auto-saves.
// This allows dynamic parameter registration at runtime.
func (c *Config) Int(flagName string, defaultValue int) int {
	var changed bool
	defer func() {
		if changed {
			err := c.Save()
			if err != nil {
				log.Error("Can not save config:", err)
			}
		}
	}()
	if c.ParamsInt == nil {
		changed = true
		c.ParamsInt = make(map[string]int)
		c.ParamsInt[flagName] = defaultValue
		return defaultValue
	}
	if value, ok := c.ParamsInt[flagName]; !ok {
		c.ParamsInt[flagName] = defaultValue
		changed = true
		return defaultValue
	} else {
		return value
	}
}
