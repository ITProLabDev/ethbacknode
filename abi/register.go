package abi

// AddContractFromABI imports a canonical Ethereum JSON ABI and registers the
// contract in one step. Returns an error if the ABI is invalid.
func (m *SmartContractsManager) AddContractFromABI(name, symbol, address string, rawABI []byte) error {
	info, err := NewContractFromABI(name, symbol, address, rawABI)
	if err != nil {
		return err
	}
	m.Add(info)
	return nil
}

// IsContractKnown reports whether a contract with the given address is
// registered. The lookup is case-insensitive (checksum-tolerant), mirroring
// GetSmartContractByAddress, so callers may pass an EIP-55 checksummed or a
// lowercase address interchangeably.
func (m *SmartContractsManager) IsContractKnown(address string) bool {
	_, err := m.GetSmartContractByAddress(address)
	return err == nil
}
