package abi

import "testing"

func TestAddContractFromABI(t *testing.T) {
	m := newTestManager(t)
	raw := `[{"type":"event","name":"Transfer","inputs":[
	  {"name":"from","type":"address","indexed":true},
	  {"name":"to","type":"address","indexed":true},
	  {"name":"value","type":"uint256"}]}]`
	if err := m.AddContractFromABI("Tok", "TOK", "0xabc0000000000000000000000000000000000001", []byte(raw)); err != nil {
		t.Fatal(err)
	}
	if _, err := m.GetSmartContractByAddress("0xabc0000000000000000000000000000000000001"); err != nil {
		t.Fatalf("contract not registered: %v", err)
	}
	if err := m.AddContractFromABI("Bad", "BAD", "0xdef", []byte(`not json`)); err == nil {
		t.Fatal("invalid ABI must error")
	}
}

func TestIsContractKnown(t *testing.T) {
	m := newTestManager(t)
	raw := `[{"type":"event","name":"Transfer","inputs":[
	  {"name":"from","type":"address","indexed":true},
	  {"name":"to","type":"address","indexed":true},
	  {"name":"value","type":"uint256"}]}]`
	// Register with a checksummed address.
	const checksummed = "0xABC0000000000000000000000000000000000001"
	if err := m.AddContractFromABI("Tok", "TOK", checksummed, []byte(raw)); err != nil {
		t.Fatal(err)
	}
	// Known regardless of address casing (checksum-tolerant lookup).
	if !m.IsContractKnown(checksummed) {
		t.Fatal("registered checksummed address must be known")
	}
	if !m.IsContractKnown("0xabc0000000000000000000000000000000000001") {
		t.Fatal("registered address must be known when queried lowercase")
	}
	// Unregistered address is not known.
	if m.IsContractKnown("0x0000000000000000000000000000000000000999") {
		t.Fatal("unregistered address must not be known")
	}
}
