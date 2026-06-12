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
