package main

import (
	"github.com/ITProLabDev/ethbacknode/clients/ethclient"
	"github.com/ITProLabDev/ethbacknode/storage"
)

// initResult reports what runInit did, so the caller can log it and tests can
// assert it without inspecting global state.
type initResult struct {
	WroteMain   bool
	WroteClient bool
}

// runInit bootstraps configuration for a target chain selected by --init.
//
// It resolves the chain profile (Eth | Arc), then writes the main config
// (node/endpoint defaults — chain-agnostic) and the client config (chain
// identity from the profile). Existing config files are NEVER overwritten:
// each is written only if absent, so re-running --init is safe. Returns an
// error for an unknown chain (and writes nothing in that case).
func runInit(chainName string, mainStorage, clientStorage storage.BinStorage) (initResult, error) {
	var res initResult

	// Resolve the profile FIRST so an unknown chain writes nothing.
	profile, err := ethclient.ChainProfileByName(chainName)
	if err != nil {
		return res, err
	}

	// Main config: node/endpoint defaults, independent of the chain.
	if !mainStorage.IsExists() {
		cfg := &Config{storage: mainStorage}
		if err := cfg.coldStart(); err != nil {
			return res, err
		}
		res.WroteMain = true
	}

	// Client config: chain identity from the selected profile.
	if !clientStorage.IsExists() {
		if err := ethclient.InitClientConfig(clientStorage, profile); err != nil {
			return res, err
		}
		res.WroteClient = true
	}

	return res, nil
}
