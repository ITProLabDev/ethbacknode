package abi

import "encoding/hex"

// DecodeCall identifies the method a transaction's calldata invokes on a
// registered contract and decodes its inputs.
//
// Unlike DecodeLog, an unknown selector is not an error: a contractTransaction
// subscriber wants to know a contract was called even when this registry
// cannot say how, so method carries the raw 4-byte selector as 0x-hex (e.g.
// "0xa9059cbb") with no decoded inputs. Calldata shorter than a selector --
// a plain value transfer with no call at all -- returns an empty method and
// no inputs. The only real error is an unregistered contract address; that
// should not happen in the contractTransaction flow, which only calls this
// for addresses that already have a subscription (which requires the ABI to
// be registered first).
func (m *SmartContractsManager) DecodeCall(contractAddress string, data []byte) (method string, inputs []DecodedValue, err error) {
	contract, err := m.GetSmartContractByAddress(contractAddress)
	if err != nil {
		return "", nil, err
	}
	if contract.Abi == nil || len(data) < 4 {
		return "", nil, nil
	}
	var sig [4]byte
	copy(sig[:], data[:4])

	entry, ferr := contract.Abi.GetMethodById(sig)
	if ferr != nil {
		return "0x" + hex.EncodeToString(sig[:]), nil, nil
	}
	decoded, derr := entry.DecodeInputsTyped(data)
	if derr != nil {
		return "0x" + hex.EncodeToString(sig[:]), nil, nil
	}
	return decoded.Method, decoded.Inputs, nil
}
