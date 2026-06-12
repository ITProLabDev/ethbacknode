package eventlog

import (
	"testing"

	"github.com/ITProLabDev/ethbacknode/abi"
)

// stubManaged implements the ManagedAddresses interface for tests.
type stubManaged struct{ known map[string]bool }

func (s stubManaged) IsAddressKnown(addr string) bool { return s.known[addr] }

// stubCodec encodes a 20-byte slice to a lowercase 0x-hex string.
type stubCodec struct{}

func (stubCodec) EncodeBytesToAddress(b []byte) (string, error) {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, 0, 2+len(b)*2)
	out = append(out, '0', 'x')
	for _, by := range b {
		out = append(out, hexdigits[by>>4], hexdigits[by&0xf])
	}
	return string(out), nil
}

func addr20(last byte) []byte {
	b := make([]byte, 20)
	b[19] = last
	return b
}

func TestEventMatchesScope_WholeContract(t *testing.T) {
	ev := &abi.DecodedEvent{Name: "X", Inputs: []abi.DecodedValue{
		{Name: "to", Type: "address", Value: addr20(0x11)},
	}}
	managed := stubManaged{known: map[string]bool{}}
	if !eventMatchesScope(ev, ScopeWholeContract, managed, stubCodec{}) {
		t.Fatal("whole_contract must always match")
	}
}

func TestEventMatchesScope_ManagedOnly(t *testing.T) {
	codec := stubCodec{}
	managedAddr, _ := codec.EncodeBytesToAddress(addr20(0x22))
	managed := stubManaged{known: map[string]bool{managedAddr: true}}

	hit := &abi.DecodedEvent{Name: "Transfer", Inputs: []abi.DecodedValue{
		{Name: "from", Type: "address", Value: addr20(0x11)},
		{Name: "to", Type: "address", Value: addr20(0x22)},
		{Name: "value", Type: "uint256", Value: nil},
	}}
	if !eventMatchesScope(hit, ScopeManagedOnly, managed, codec) {
		t.Fatal("managed_only must match when a managed address is involved")
	}

	miss := &abi.DecodedEvent{Name: "Transfer", Inputs: []abi.DecodedValue{
		{Name: "from", Type: "address", Value: addr20(0x11)},
		{Name: "to", Type: "address", Value: addr20(0x33)},
	}}
	if eventMatchesScope(miss, ScopeManagedOnly, managed, codec) {
		t.Fatal("managed_only must NOT match when no managed address is involved")
	}
}

func TestEventMatchesScope_ManagedOnly_IgnoresNonAddress(t *testing.T) {
	managed := stubManaged{known: map[string]bool{}}
	ev := &abi.DecodedEvent{Name: "X", Inputs: []abi.DecodedValue{
		{Name: "n", Type: "uint256", Value: nil},
		{Name: "s", Type: "string", Value: "hi"},
	}}
	if eventMatchesScope(ev, ScopeManagedOnly, managed, stubCodec{}) {
		t.Fatal("managed_only with no address params must not match")
	}
}
