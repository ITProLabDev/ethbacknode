package abi

import (
	"errors"
	"math/big"
	"testing"
)

const transferABI = `[{"type":"function","name":"transfer","stateMutability":"nonpayable",` +
	`"inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],` +
	`"outputs":[{"name":"","type":"bool"}]}]`

const decodeCallAddr = "0xdec0000000000000000000000000000000000001"

func newDecodeCallManager(t *testing.T) *SmartContractsManager {
	t.Helper()
	m := newTestManager(t)
	if err := m.AddContractFromABI("Token", "TOK", decodeCallAddr, []byte(transferABI)); err != nil {
		t.Fatalf("AddContractFromABI: %v", err)
	}
	return m
}

func TestDecodeCallDecodesAKnownMethod(t *testing.T) {
	m := newDecodeCallManager(t)
	contract, err := m.GetSmartContractByAddress(decodeCallAddr)
	if err != nil {
		t.Fatalf("GetSmartContractByAddress: %v", err)
	}
	entry, err := contract.Abi.GetMethodByName("transfer")
	if err != nil {
		t.Fatalf("GetMethodByName: %v", err)
	}
	to := make([]byte, 20)
	to[19] = 0x42
	data, err := entry.EncodeInputsTyped(to, big.NewInt(1000))
	if err != nil {
		t.Fatalf("EncodeInputsTyped: %v", err)
	}

	method, inputs, err := m.DecodeCall(decodeCallAddr, data)
	if err != nil {
		t.Fatalf("DecodeCall: %v", err)
	}
	if method != "transfer" {
		t.Fatalf("method = %q, want %q", method, "transfer")
	}
	if len(inputs) != 2 {
		t.Fatalf("got %d inputs, want 2: %+v", len(inputs), inputs)
	}
}

func TestDecodeCallReturnsTheHexSelectorForAnUnknownMethod(t *testing.T) {
	m := newDecodeCallManager(t)
	// A selector that matches no entry in the registered ABI, followed by
	// some arbitrary argument data.
	data := append([]byte{0xde, 0xad, 0xbe, 0xef}, make([]byte, 32)...)

	method, inputs, err := m.DecodeCall(decodeCallAddr, data)
	if err != nil {
		t.Fatalf("DecodeCall: %v", err)
	}
	if method != "0xdeadbeef" {
		t.Fatalf("method = %q, want %q", method, "0xdeadbeef")
	}
	if inputs != nil {
		t.Fatalf("inputs = %+v, want nil for an undecodable call", inputs)
	}
}

func TestDecodeCallReturnsEmptyMethodForCalldataShorterThanASelector(t *testing.T) {
	m := newDecodeCallManager(t)

	method, inputs, err := m.DecodeCall(decodeCallAddr, []byte{0x01, 0x02})
	if err != nil {
		t.Fatalf("DecodeCall: %v", err)
	}
	if method != "" {
		t.Fatalf("method = %q, want empty (no selector, e.g. a plain value transfer)", method)
	}
	if inputs != nil {
		t.Fatalf("inputs = %+v, want nil", inputs)
	}
}

func TestDecodeCallErrorsOnAnUnregisteredContract(t *testing.T) {
	m := newDecodeCallManager(t)

	_, _, err := m.DecodeCall("0x0000000000000000000000000000000000009999", []byte{0xde, 0xad, 0xbe, 0xef})
	if !errors.Is(err, ErrUnknownContract) {
		t.Fatalf("err = %v, want ErrUnknownContract", err)
	}
}
