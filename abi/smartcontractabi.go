package abi

import (
	"backnode/common/hexnum"
	"backnode/crypto"
	"backnode/tools/log"
	"fmt"
	"math/big"
	"strings"
)

const (
	ParamTypeUint256 = "uint256"
)

type SmartContractInfo struct {
	Name            string            `json:"name"`
	Symbol          string            `json:"symbol"`
	ContractAddress string            `json:"contract_address"`
	OriginAddress   string            `json:"origin_address"`
	Decimals        int               `json:"decimals"`
	OriginGasLimit  int64             `json:"origin_gas_limit"`
	Abi             *SmartContractAbi `json:"abi"`
}

type SmartContractAbi struct {
	prepared bool
	Entries  []*SmartContractAbiEntry `json:"entries"`
}

func (a *SmartContractAbi) AddEntry(entry *SmartContractAbiEntry) {
	a.Entries = append(a.Entries, entry)
}

func NewEntry() *SmartContractAbiEntry {
	return &SmartContractAbiEntry{}
}

type SmartContractAbiEntry struct {
	Constant        bool                           `json:"constant,omitempty"`
	Signature       [4]byte                        `json:"-"`
	Name            string                         `json:"name,omitempty"`
	StateMutability string                         `json:"stateMutability,omitempty"`
	Type            string                         `json:"type"`
	Inputs          []*SmartContractAbiEntryInput  `json:"inputs,omitempty"`
	Outputs         []*SmartContractAbiEntryOutput `json:"outputs,omitempty"`
}

type SmartContractAbiEntryInput struct {
	Name    string `json:"name,omitempty"`
	Type    string `json:"type"`
	Indexed bool   `json:"indexed,omitempty"`
	data    []byte
}

type SmartContractAbiEntryOutput struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

func (e *SmartContractAbiEntry) DecodeInputs(data []byte) (paramsParsed []*paramInput, err error) {
	//skip method signature
	data = data[4:]
	paramsParsed = make([]*paramInput, len(e.Inputs))
	for i, param := range e.Inputs {
		if len(data) == 0 {
			log.Error("invalid params Data")
			return nil, ErrInvalidParamsData
		}
		nextParamParsed := &paramInput{
			Name: param.Name,
			Type: param.Type,
		}
		next := _parseParam(nextParamParsed, param.Type, data)
		paramsParsed[i] = nextParamParsed
		if len(data) >= next {
			data = data[next:]
		} else {
			return nil, ErrInvalidParamsData
		}
	}
	return paramsParsed, nil
}

func (e *SmartContractAbiEntry) encodeInputs(params ...interface{}) (data string, err error) {
	signature := e.GetSignature()
	dataBytes := make([]byte, 4)
	copy(dataBytes, signature[0:4])
	if len(e.Inputs) != len(params) {
		return "", ErrSmartContractMethodParamsCountMismatch
	}
	for i, paramEntry := range e.Inputs {
		inputParam := new(paramInput)
		param := params[i]
		switch paramEntry.Type {
		case "address":
			addr := param.([]byte)
			inputParam.SetAddress(addr)
		case "uint256":
			inputParam.Data = param.([]byte)
		case "bool":
			val := param.(bool)
			inputParam.SetBool(val)
		}
		dataBytes = append(dataBytes, inputParam.Data...)
	}
	data = hexnum.BytesToHex(dataBytes)
	return data, nil
}

func (e *SmartContractAbiEntry) encodeInputsBytes(params ...interface{}) (dataBytes []byte, err error) {
	signature := e.GetSignature()
	dataBytes = make([]byte, 4)
	copy(dataBytes, signature[0:4])
	if len(e.Inputs) != len(params) {
		return nil, ErrSmartContractMethodParamsCountMismatch
	}
	for i, paramEntry := range e.Inputs {
		inputParam := new(paramInput)
		param := params[i]
		switch paramEntry.Type {
		case "address":
			addr := param.([]byte)
			inputParam.SetAddress(addr)
		case "uint256":
			inputParam.Data = param.([]byte)
		case "bool":
			val := param.(bool)
			inputParam.SetBool(val)
		}
		dataBytes = append(dataBytes, inputParam.Data...)
	}
	return dataBytes, nil
}

func (e *SmartContractAbiEntry) GetSignature() [4]byte {
	if e._isSignatureEmpty() {
		e.updateSignature()
	}
	return e.Signature
}

func (e *SmartContractAbiEntry) checkSignature(signature [4]byte) bool {
	for i, b := range e.Signature {
		if signature[i] != b {
			return false
		}
	}
	return true
}

func (e *SmartContractAbiEntry) _isSignatureEmpty() bool {
	for _, b := range e.Signature {
		if b != 0 {
			return false
		}
	}
	return true
}

func (e *SmartContractAbiEntry) String() string {
	var params = make([]string, len(e.Inputs))
	var output = make([]string, len(e.Outputs))
	for i, in := range e.Inputs {
		params[i] = in.Name + " " + in.Type
	}
	for i, out := range e.Outputs {
		output[i] = out.Name + " " + out.Type
	}
	return e.Type + ": " + e.Name + "(" + strings.Join(params, ",") + ")" + strings.Join(output, ",") + ", methodId: 0x" + fmt.Sprintf("%x", e.Signature)
}

func (e *SmartContractAbiEntry) updateSignature() {
	var params = make([]string, len(e.Inputs))
	for i, in := range e.Inputs {
		params[i] = in.Type
	}
	h := crypto.Keccak256([]byte(e.Name + "(" + strings.Join(params, ",") + ")"))
	copy(e.Signature[:], h[:4])
}

func (a *SmartContractAbi) _prepare() {
	for _, e := range a.Entries {
		e.updateSignature()
	}
}

func (a *SmartContractAbi) dumpMethods() {
	if !a.prepared {
		a._prepare()
	}
	for i, m := range a.Entries {
		log.Debug(i, ":", m)
	}
}

func (a *SmartContractAbi) GetMethodById(signature [4]byte) (entry *SmartContractAbiEntry, err error) {
	if !a.prepared {
		a._prepare()
	}
	for _, entry = range a.Entries {
		if entry.checkSignature(signature) {
			return entry, nil
		}
	}
	return nil, ErrSmartContractUnknownMethod
}

func (a *SmartContractAbi) GetMethodByName(name string) (entry *SmartContractAbiEntry, err error) {
	if !a.prepared {
		a._prepare()
	}
	for _, entry = range a.Entries {
		if entry.Name == name {
			return entry, nil
		}
	}
	return nil, ErrSmartContractUnknownMethod
}

type paramInput struct {
	Name string
	Type string
	Data []byte
}

func (p *paramInput) GetAddressBytes() (address []byte) {
	return p.Data
}

func (p *paramInput) GetBigInt() *big.Int {
	return new(big.Int).SetBytes(p.Data)
}
func (p *paramInput) GetInt64() (num int64) {
	return _byteToInt64(p.Data)
}

func (p *paramInput) GetBool() bool {
	if p.Data[0] == 0 {
		return false
	}
	return true
}

func (p *paramInput) SetAddress(addrBytes []byte) (err error) {
	addressBytes := bytePad(addrBytes, 32, 0)
	p.Data = addressBytes
	return nil
}

func (p *paramInput) SetInt64(amount int64) (err error) {
	p.Data = bytePad(_int64ToByte(amount), 32, 0)
	return
}

func (p *paramInput) SetBigInt(amount *big.Int) {
	p.Data = bytePad(amount.Bytes(), 32, 0)
}
func (p *paramInput) SetBool(val bool) {
	if val {
		p.Data[0] = 1
	}
}

func _parseParam(param *paramInput, paramType string, data []byte) (nextParam int) {
	switch paramType {
	case "uint256":
		param.Data = make([]byte, 32)
		copy(param.Data, data[0:32])
		return 32
	case "int256":
		param.Data = make([]byte, 32)
		copy(param.Data, data[0:32])
		return 32
	case "address":
		param.Data = make([]byte, 20)
		copy(param.Data, data[12:32])
		return 32
	case "bool":
		param.Data = make([]byte, 1)
		copy(param.Data, data[31:32])
		return 32
	}
	return
}

func _extractMethodId(data []byte) [4]byte {
	var signature [4]byte
	copy(signature[:], data[:4])
	return signature
}
