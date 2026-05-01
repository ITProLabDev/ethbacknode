package abi

import (
	"math/big"
)

// Erc20CallGetBalance encodes a balanceOf call for the given address.
func (m *SmartContractsManager) Erc20CallGetBalance(address string) (callData string, err error) {
	if m.erc20abi == nil {
		return "", ErrSmartContractUnknownMethod
	}
	if m.addressCodec == nil {
		return "", ErrInvalidParamsData
	}
	method, err := m.erc20abi.GetMethodByName("balanceOf")
	if err != nil {
		return "", err
	}
	addressBytes, err := m.addressCodec.DecodeAddressToBytes(address)
	if err != nil {
		return "", err
	}
	return method.encodeInputs(addressBytes)
}

// Erc20IsTransfer checks if the call data is an ERC-20 transfer method.
func (m *SmartContractsManager) Erc20IsTransfer(callData []byte) bool {
	if m.erc20abi == nil {
		return false
	}
	method, err := m.erc20abi.GetMethodByName("transfer")
	if err != nil {
		return false
	}
	callDataMethod, ok := _extractMethodId(callData)
	if !ok {
		return false
	}
	return method.checkSignature(callDataMethod)
}

// Erc20DecodeAmount decodes a uint256 amount from call data.
func (m *SmartContractsManager) Erc20DecodeAmount(callData []byte) (amount *big.Int) {
	param := paramInput{
		Type: "uint256",
		Data: callData,
	}
	return param.GetBigInt()
}

// Erc20DecodeIfTransfer decodes a transfer method's recipient and amount.
// Returns ErrNotTransferMethod if the call data is not a transfer.
func (m *SmartContractsManager) Erc20DecodeIfTransfer(callData []byte) (address string, amount *big.Int, err error) {
	if m.erc20abi == nil {
		return "", nil, ErrSmartContractUnknownMethod
	}
	method, err := m.erc20abi.GetMethodByName("transfer")
	if err != nil {
		return "", nil, err
	}
	callDataMethod, ok := _extractMethodId(callData)
	if !ok {
		return "", nil, ErrNotTransferMethod
	}
	if !method.checkSignature(callDataMethod) {
		return "", nil, ErrNotTransferMethod
	}
	params, err := method.DecodeInputs(callData)
	if err != nil {
		return "", nil, err
	}
	for _, param := range params {
		switch param.Type {
		case "address":
			address, err = m.addressCodec.EncodeBytesToAddress(param.GetAddressBytes())
			if err != nil {
				return "", nil, err
			}
		case "uint256":
			amount = param.GetBigInt()
		default:
			return "", nil, ErrInvalidParamsData
		}
	}
	return address, amount, nil
}

/*

function name() public view returns (string)
function symbol() public view returns (string)
function decimals() public view returns (uint8)
function totalSupply() public view returns (uint256)
function balanceOf(address _owner) public view returns (uint256 balance)
function transfer(address _to, uint256 _value) public returns (bool success)
function transferFrom(address _from, address _to, uint256 _value) public returns (bool success)

*/
