package abi

import (
	"testing"
)

const erc20ABIArray = `[
  {"type":"function","name":"transfer","stateMutability":"nonpayable",
   "inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],
   "outputs":[{"name":"","type":"bool"}]},
  {"type":"event","name":"Transfer","anonymous":false,
   "inputs":[{"name":"from","type":"address","indexed":true},
             {"name":"to","type":"address","indexed":true},
             {"name":"value","type":"uint256","indexed":false}]}
]`

func TestImportEthereumABI_ArrayForm(t *testing.T) {
	a, err := ImportEthereumABI([]byte(erc20ABIArray))
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Entries) != 2 {
		t.Fatalf("entries=%d want 2", len(a.Entries))
	}
	transfer, err := a.GetMethodByName("transfer")
	if err != nil {
		t.Fatal(err)
	}
	if len(transfer.Inputs) != 2 || transfer.Inputs[0].Type != "address" {
		t.Fatalf("transfer inputs=%+v", transfer.Inputs)
	}
	// The lowercase "function"/"event" types must be preserved as-is.
	if transfer.Type != "function" {
		t.Fatalf("type=%q want lowercase function", transfer.Type)
	}
	// Event lookup by the canonical mainnet topic0 proves the event imported
	// correctly (Transfer(address,address,uint256)).
	ev, err := a.GetEventByTopic0(mustHex32(t, "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"))
	if err != nil || ev.Name != "Transfer" {
		t.Fatalf("event by topic0: %v %v", ev, err)
	}
}

func TestImportEthereumABI_ObjectForm(t *testing.T) {
	// Also accept the project's own { "entries": [...] } wrapper form.
	obj := `{"entries":` + erc20ABIArray + `}`
	a, err := ImportEthereumABI([]byte(obj))
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Entries) != 2 {
		t.Fatalf("entries=%d want 2", len(a.Entries))
	}
}

func TestImportEthereumABI_Errors(t *testing.T) {
	cases := map[string]string{
		"empty array":     `[]`,
		"empty object":    `{"entries":[]}`,
		"not json":        `not json at all`,
		"bad type":        `[{"type":"function","name":"x","inputs":[{"name":"a","type":"uint999"}]}]`,
		"null entry":      `[null]`,
		"nested bad type": `[{"type":"function","name":"x","inputs":[{"type":"tuple","components":[{"type":"uint999"}]}]}]`,
		"tuple output":    `[{"type":"function","name":"x","inputs":[],"outputs":[{"type":"tuple","components":[{"type":"uint256"}]}]}]`,
	}
	for name, raw := range cases {
		if _, err := ImportEthereumABI([]byte(raw)); err == nil {
			t.Fatalf("%s: expected error", name)
		}
	}
}

func TestImportEthereumABI_Tuple(t *testing.T) {
	// A function taking a struct (tuple with components) must import and its
	// canonical signature must render the tuple form.
	raw := `[
	  {"type":"function","name":"submit","stateMutability":"nonpayable",
	   "inputs":[{"name":"order","type":"tuple",
	     "components":[{"name":"amt","type":"uint256"},{"name":"maker","type":"address"}]}],
	   "outputs":[]}
	]`
	a, err := ImportEthereumABI([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	m, err := a.GetMethodByName("submit")
	if err != nil {
		t.Fatal(err)
	}
	if got := m.canonicalSignature(); got != "submit((uint256,address))" {
		t.Fatalf("canonicalSignature=%q", got)
	}
}

func TestNewContractFromABI(t *testing.T) {
	c, err := NewContractFromABI("MyToken", "MTK", "0xAbC1230000000000000000000000000000000001", []byte(erc20ABIArray))
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "MyToken" || c.Symbol != "MTK" {
		t.Fatalf("contract meta=%+v", c)
	}
	if c.ContractAddress != "0xAbC1230000000000000000000000000000000001" {
		t.Fatalf("address not preserved: %q", c.ContractAddress)
	}
	if c.Abi == nil || len(c.Abi.Entries) != 2 {
		t.Fatalf("abi not attached: %+v", c.Abi)
	}
}

func TestNewContractFromABI_RejectsBadABI(t *testing.T) {
	if _, err := NewContractFromABI("X", "X", "0x01", []byte(`[]`)); err == nil {
		t.Fatal("empty abi must be rejected")
	}
	if _, err := NewContractFromABI("X", "X", "0x01", []byte(`garbage`)); err == nil {
		t.Fatal("bad json must be rejected")
	}
}
