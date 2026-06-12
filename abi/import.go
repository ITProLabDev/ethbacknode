package abi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// ImportEthereumABI converts a canonical Ethereum JSON ABI into the project's
// own SmartContractAbi. It accepts either the standard top-level array form
// (as emitted by solc / Etherscan / Polygonscan) or the project's own
// {"entries": [...]} object form. Every entry's input types are validated with
// the typed engine, so a malformed type fails fast at import time.
//
// Field casing is preserved as-is: canonical ABIs use lowercase type values
// ("function", "event"); the engine compares them case-insensitively.
//
// Returns ErrEmptyABI when there are no entries and ErrInvalidABIJSON for
// malformed JSON, null entries, invalid types, or tuple outputs.
func ImportEthereumABI(raw []byte) (*SmartContractAbi, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, ErrInvalidABIJSON
	}

	abi := &SmartContractAbi{}
	switch trimmed[0] {
	case '[':
		var entries []*SmartContractAbiEntry
		if err := json.Unmarshal(trimmed, &entries); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidABIJSON, err)
		}
		abi.Entries = entries
	case '{':
		if err := json.Unmarshal(trimmed, abi); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidABIJSON, err)
		}
	default:
		return nil, ErrInvalidABIJSON
	}

	if len(abi.Entries) == 0 {
		return nil, ErrEmptyABI
	}

	// Validate every entry's input/output types parse with the typed engine.
	for _, e := range abi.Entries {
		if e == nil {
			return nil, fmt.Errorf("%w: null entry", ErrInvalidABIJSON)
		}
		for _, in := range e.Inputs {
			if _, err := parseType(in.Type, in.Components); err != nil {
				return nil, fmt.Errorf("%w: entry %q input %q: %v", ErrInvalidABIJSON, e.Name, in.Type, err)
			}
		}
		for _, out := range e.Outputs {
			// The output model carries no components, so tuple outputs cannot be
			// faithfully represented yet — reject them loudly rather than import
			// an empty tuple. (Tuple-output support is deferred to the M4 call layer.)
			if out.Type == "tuple" || strings.HasPrefix(out.Type, "tuple[") {
				return nil, fmt.Errorf("%w: entry %q tuple output not supported", ErrInvalidABIJSON, e.Name)
			}
			if _, err := parseType(out.Type, nil); err != nil {
				return nil, fmt.Errorf("%w: entry %q output %q: %v", ErrInvalidABIJSON, e.Name, out.Type, err)
			}
		}
	}

	return abi, nil
}
