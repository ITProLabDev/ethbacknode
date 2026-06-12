package abi

import (
	"bytes"
	"math/big"
	"testing"
)

func TestUpdateSignature_Erc20TransferUnchanged(t *testing.T) {
	// Regression: switching to canonical type strings must NOT change the
	// well-known ERC-20 transfer selector 0xa9059cbb.
	e := &SmartContractAbiEntry{
		Name: "transfer",
		Type: "Function",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "_to", Type: "address"},
			{Name: "_value", Type: "uint256"},
		},
	}
	sig := e.GetSignature()
	want := [4]byte{0xa9, 0x05, 0x9c, 0xbb}
	if sig != want {
		t.Fatalf("selector=%x want %x", sig, want)
	}
}

func TestUpdateSignature_TupleCanonical(t *testing.T) {
	// A method taking a tuple must hash the canonical "(uint256,address)"
	// form, not the literal "tuple".
	e := &SmartContractAbiEntry{
		Name: "order",
		Type: "Function",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "o", Type: "tuple", Components: []*SmartContractAbiEntryInput{
				{Name: "amt", Type: "uint256"},
				{Name: "maker", Type: "address"},
			}},
		},
	}
	got := e.canonicalSignature()
	if got != "order((uint256,address))" {
		t.Fatalf("canonicalSignature=%q", got)
	}
}

func TestDecodeInputsTyped_TransferWithTuple(t *testing.T) {
	e := &SmartContractAbiEntry{
		Name: "submit",
		Type: "Function",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "id", Type: "uint256"},
			{Name: "data", Type: "bytes"},
		},
	}
	sig := e.GetSignature()
	body, err := encodeParams(
		[]abiType{{kind: kindUint, size: 256}, {kind: kindBytes}},
		[]any{big.NewInt(5), []byte("hi")},
	)
	if err != nil {
		t.Fatal(err)
	}
	call := append(sig[:], body...)
	decoded, err := e.DecodeInputsTyped(call)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Method != "submit" || len(decoded.Inputs) != 2 {
		t.Fatalf("decoded=%+v", decoded)
	}
	if decoded.Inputs[0].Value.(*big.Int).Int64() != 5 {
		t.Fatalf("id=%v", decoded.Inputs[0].Value)
	}
	if string(decoded.Inputs[1].Value.([]byte)) != "hi" {
		t.Fatalf("data=%v", decoded.Inputs[1].Value)
	}
	if decoded.Inputs[0].Name != "id" {
		t.Fatalf("name not set: %+v", decoded.Inputs[0])
	}
}

func TestEncodeInputsTyped_RoundTrip(t *testing.T) {
	e := &SmartContractAbiEntry{
		Name: "transfer",
		Type: "Function",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "_to", Type: "address"},
			{Name: "_value", Type: "uint256"},
		},
	}
	addr := make([]byte, 20)
	addr[0], addr[19] = 0xde, 0xef
	amount := big.NewInt(1000)

	callData, err := e.EncodeInputsTyped(addr, amount)
	if err != nil {
		t.Fatal(err)
	}
	sig := e.GetSignature()
	if !bytes.Equal(callData[:4], sig[:]) {
		t.Fatalf("selector prefix=%x want %x", callData[:4], sig[:])
	}
	decoded, err := e.DecodeInputsTyped(callData)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded.Inputs[0].Value.([]byte), addr) {
		t.Fatalf("addr round-trip: %x", decoded.Inputs[0].Value)
	}
	if decoded.Inputs[1].Value.(*big.Int).Cmp(amount) != 0 {
		t.Fatalf("amount round-trip: %v", decoded.Inputs[1].Value)
	}
}

func TestDecodeInputsTyped_WithRealTuple(t *testing.T) {
	e := &SmartContractAbiEntry{
		Name: "order",
		Type: "Function",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "o", Type: "tuple", Components: []*SmartContractAbiEntryInput{
				{Name: "amt", Type: "uint256"},
				{Name: "maker", Type: "address"},
			}},
		},
	}
	maker := make([]byte, 20)
	maker[0], maker[19] = 0xaa, 0xbb
	callData, err := e.EncodeInputsTyped([]any{big.NewInt(77), maker})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := e.DecodeInputsTyped(callData)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Method != "order" || len(decoded.Inputs) != 1 {
		t.Fatalf("decoded=%+v", decoded)
	}
	fields := decoded.Inputs[0].Value.([]DecodedValue)
	if len(fields) != 2 || fields[0].Value.(*big.Int).Int64() != 77 {
		t.Fatalf("tuple fields=%+v", fields)
	}
	if !bytes.Equal(fields[1].Value.([]byte), maker) {
		t.Fatalf("maker=%x", fields[1].Value)
	}
	if fields[0].Name != "amt" || fields[1].Name != "maker" {
		t.Fatalf("tuple field names not populated: %+v", fields)
	}
}
