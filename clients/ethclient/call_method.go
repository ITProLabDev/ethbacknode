package ethclient

import (
	"github.com/ITProLabDev/ethbacknode/abi"
	"github.com/ITProLabDev/ethbacknode/common/hexnum"
)

// CallMethod invokes a contract view/pure method by name via eth_call and
// returns its decoded outputs. The contract must be registered in the ABI
// manager (so its ABI is known). args are the typed method inputs, matching
// the engine's per-kind Go types (address/bytesN/bytes -> []byte, bool -> bool,
// uint*/int* -> *big.Int|int|int64|uint64, string -> string, arrays/tuples ->
// []any). Returns the decoded return values in declaration order.
func (c *Client) CallMethod(contractAddress, methodName string, args ...any) ([]abi.DecodedValue, error) {
	contract, err := c.abi.GetSmartContractByAddress(contractAddress)
	if err != nil {
		return nil, err
	}
	if contract.Abi == nil {
		return nil, abi.ErrUnknownContract
	}
	method, err := contract.Abi.GetMethodByName(methodName)
	if err != nil {
		return nil, err
	}
	callData, err := method.EncodeInputsTyped(args...)
	if err != nil {
		return nil, err
	}
	resultHex, err := c.Call(contractAddress, hexnum.BytesToHex(callData))
	if err != nil {
		return nil, err
	}
	resultBytes, err := hexnum.ParseHexBytes(resultHex)
	if err != nil {
		return nil, err
	}
	return method.DecodeOutputs(resultBytes)
}
