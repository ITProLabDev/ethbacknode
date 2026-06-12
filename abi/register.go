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
