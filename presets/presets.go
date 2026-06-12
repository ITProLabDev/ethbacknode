// Package presets ships built-in smart-contract bundles embedded in the binary
// and applies them to the abi registry at startup, matched by numeric chainId.
// This is the data/registration-driven preset path (PROJECT_STATUS §6): no
// contract-specific Go logic — only ABI data + registration.
package presets

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/ITProLabDev/ethbacknode/abi"
)

//go:embed arc.json
var arcBundleRaw []byte

// presetContract is one contract in a bundle: identity + its canonical ABI.
type presetContract struct {
	Name     string          `json:"name"`
	Symbol   string          `json:"symbol"`
	Decimals int             `json:"decimals"`
	Address  string          `json:"address"`
	ABI      json.RawMessage `json:"abi"` // canonical Ethereum ABI array, verbatim
}

// bundle is the set of contracts deployed to one chain.
type bundle struct {
	ChainID     int64            `json:"chainId"`
	Source      string           `json:"source"`
	GeneratedAt string           `json:"generatedAt"`
	Contracts   []presetContract `json:"contracts"`
}

// ContractAdder is the narrow registry surface Apply needs. Satisfied by
// *abi.SmartContractsManager.
type ContractAdder interface {
	Add(c *abi.SmartContractInfo)
}

// allBundles parses every embedded preset file into bundles.
func allBundles() ([]bundle, error) {
	var bundles []bundle
	if err := json.Unmarshal(arcBundleRaw, &bundles); err != nil {
		return nil, fmt.Errorf("presets: parse arc.json: %w", err)
	}
	return bundles, nil
}

// Apply registers every preset contract whose bundle chainId equals chainID.
// It builds each contract via abi.NewContractFromABI (validating the ABI) and
// hands it to adder.Add. Returns the number registered and the first error.
// A chainID with no matching bundle registers nothing and returns (0, nil).
func Apply(chainID int64, adder ContractAdder) (loaded int, err error) {
	bundles, err := allBundles()
	if err != nil {
		return 0, err
	}
	for _, b := range bundles {
		if b.ChainID != chainID {
			continue
		}
		for _, pc := range b.Contracts {
			info, ierr := abi.NewContractFromABI(pc.Name, pc.Symbol, pc.Address, []byte(pc.ABI))
			if ierr != nil {
				return loaded, fmt.Errorf("presets: contract %q (chain %d): %w", pc.Name, chainID, ierr)
			}
			info.Decimals = pc.Decimals
			adder.Add(info)
			loaded++
		}
	}
	return loaded, nil
}
