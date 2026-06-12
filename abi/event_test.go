package abi

import (
	"encoding/hex"
	"strings"
	"testing"
)

func mustHex32(t *testing.T, s string) [32]byte {
	t.Helper()
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil || len(b) != 32 {
		t.Fatalf("bad 32-byte hex %q: %v", s, err)
	}
	var out [32]byte
	copy(out[:], b)
	return out
}

func TestTopic0_Erc20Transfer(t *testing.T) {
	// Transfer(address,address,uint256) — the canonical ERC-20 Transfer topic0.
	e := &SmartContractAbiEntry{
		Name: "Transfer",
		Type: "Event",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "from", Type: "address", Indexed: true},
			{Name: "to", Type: "address", Indexed: true},
			{Name: "value", Type: "uint256"},
		},
	}
	got := e.Topic0()
	want := mustHex32(t, "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	if got != want {
		t.Fatalf("topic0=%x want %x", got, want)
	}
}

func TestTopic0_Erc1155TransferSingle(t *testing.T) {
	// TransferSingle(address,address,address,uint256,uint256)
	e := &SmartContractAbiEntry{
		Name: "TransferSingle",
		Type: "Event",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "operator", Type: "address", Indexed: true},
			{Name: "from", Type: "address", Indexed: true},
			{Name: "to", Type: "address", Indexed: true},
			{Name: "id", Type: "uint256"},
			{Name: "value", Type: "uint256"},
		},
	}
	got := e.Topic0()
	want := mustHex32(t, "c3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62")
	if got != want {
		t.Fatalf("topic0=%x want %x", got, want)
	}
}
