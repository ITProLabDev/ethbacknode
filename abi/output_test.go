package abi

import (
	"math/big"
	"testing"
)

func TestDecodeOutputs_SingleUint(t *testing.T) {
	e := &SmartContractAbiEntry{
		Name: "balanceOf",
		Type: "function",
		Outputs: []*SmartContractAbiEntryOutput{
			{Type: "uint256"},
		},
	}
	data, err := encodeParams([]abiType{{kind: kindUint, size: 256}}, []any{big.NewInt(1000)})
	if err != nil {
		t.Fatal(err)
	}
	vals, err := e.DecodeOutputs(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 1 || vals[0].Value.(*big.Int).Int64() != 1000 {
		t.Fatalf("outputs=%+v", vals)
	}
}

func TestDecodeOutputs_MultipleWithNames(t *testing.T) {
	e := &SmartContractAbiEntry{
		Name: "getReserves",
		Type: "function",
		Outputs: []*SmartContractAbiEntryOutput{
			{Name: "reserve0", Type: "uint112"},
			{Name: "reserve1", Type: "uint112"},
		},
	}
	data, err := encodeParams(
		[]abiType{{kind: kindUint, size: 112}, {kind: kindUint, size: 112}},
		[]any{big.NewInt(7), big.NewInt(8)},
	)
	if err != nil {
		t.Fatal(err)
	}
	vals, err := e.DecodeOutputs(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 2 {
		t.Fatalf("outputs=%d", len(vals))
	}
	if vals[0].Name != "reserve0" || vals[0].Value.(*big.Int).Int64() != 7 {
		t.Fatalf("out0=%+v", vals[0])
	}
	if vals[1].Name != "reserve1" || vals[1].Value.(*big.Int).Int64() != 8 {
		t.Fatalf("out1=%+v", vals[1])
	}
}

func TestDecodeOutputs_DynamicArray(t *testing.T) {
	e := &SmartContractAbiEntry{
		Name: "balanceOfBatch",
		Type: "function",
		Outputs: []*SmartContractAbiEntryOutput{
			{Type: "uint256[]"},
		},
	}
	arr, _ := parseType("uint256[]", nil)
	data, err := encodeParams([]abiType{arr}, []any{[]any{big.NewInt(1), big.NewInt(2)}})
	if err != nil {
		t.Fatal(err)
	}
	vals, err := e.DecodeOutputs(data)
	if err != nil {
		t.Fatal(err)
	}
	got := vals[0].Value.([]DecodedValue)
	if len(got) != 2 || got[1].Value.(*big.Int).Int64() != 2 {
		t.Fatalf("array out=%+v", got)
	}
}

func TestDecodeOutputs_NoOutputs(t *testing.T) {
	e := &SmartContractAbiEntry{Name: "doThing", Type: "function"}
	vals, err := e.DecodeOutputs(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 0 {
		t.Fatalf("want 0 outputs, got %d", len(vals))
	}
}
