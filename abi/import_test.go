package abi

import (
	"math/big"
	"os"
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

func TestRegistry_ArbitraryContractRoundTrip(t *testing.T) {
	m := newTestManager(t) // mem storage + hex codec, from abi_test.go

	raw := `[
	  {"type":"function","name":"balanceOf","stateMutability":"view",
	   "inputs":[{"name":"account","type":"address"},{"name":"id","type":"uint256"}],
	   "outputs":[{"name":"","type":"uint256"}]},
	  {"type":"event","name":"TransferSingle","anonymous":false,
	   "inputs":[{"name":"operator","type":"address","indexed":true},
	             {"name":"from","type":"address","indexed":true},
	             {"name":"to","type":"address","indexed":true},
	             {"name":"id","type":"uint256","indexed":false},
	             {"name":"value","type":"uint256","indexed":false}]}
	]`
	addr := "0xabc0000000000000000000000000000000000111"
	c, err := NewContractFromABI("CTF", "CTF", addr, []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	m.Add(c)

	got, err := m.GetSmartContractByAddress(addr)
	if err != nil || got.Name != "CTF" {
		t.Fatalf("lookup by address: %v %v", got, err)
	}

	ev, err := got.Abi.GetMethodByName("TransferSingle")
	if err != nil {
		t.Fatal(err)
	}
	op := mustHex32(t, "000000000000000000000000"+"1111111111111111111111111111111111111111")
	from := mustHex32(t, "000000000000000000000000"+"2222222222222222222222222222222222222222")
	to := mustHex32(t, "000000000000000000000000"+"3333333333333333333333333333333333333333")
	topics := [][32]byte{ev.Topic0(), op, from, to}
	idsType, _ := parseType("uint256", nil)
	valType, _ := parseType("uint256", nil)
	data, err := encodeParams([]abiType{idsType, valType}, []any{big.NewInt(5), big.NewInt(9)})
	if err != nil {
		t.Fatal(err)
	}

	decoded, err := m.DecodeLog(addr, topics, data)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Name != "TransferSingle" || len(decoded.Inputs) != 5 {
		t.Fatalf("decoded=%+v", decoded)
	}
	if decoded.Inputs[3].Value.(*big.Int).Int64() != 5 {
		t.Fatalf("id=%v", decoded.Inputs[3].Value)
	}
	if decoded.Inputs[4].Value.(*big.Int).Int64() != 9 {
		t.Fatalf("value=%v", decoded.Inputs[4].Value)
	}
}

func TestImport_GenericContractClassFixtures(t *testing.T) {
	// Multi-token (ERC-1155) role: dynamic-array methods + transfer events.
	mt, err := os.ReadFile("testdata/multitoken_erc1155.json")
	if err != nil {
		t.Fatal(err)
	}
	a, err := ImportEthereumABI(mt)
	if err != nil {
		t.Fatalf("multitoken import: %v", err)
	}
	if _, err := a.GetMethodByName("balanceOfBatch"); err != nil {
		t.Fatal(err)
	}
	// TransferSingle event imported with the real mainnet ERC-1155 topic0.
	if _, err := a.GetEventByTopic0(mustHex32(t, "c3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62")); err != nil {
		t.Fatalf("TransferSingle topic0 not found: %v", err)
	}

	// Outcome-market role: bytes32 ids + dynamic arrays + indexed bytes32.
	om, err := os.ReadFile("testdata/outcome_market.json")
	if err != nil {
		t.Fatal(err)
	}
	oa, err := ImportEthereumABI(om)
	if err != nil {
		t.Fatalf("outcome_market import: %v", err)
	}
	if _, err := oa.GetMethodByName("splitPosition"); err != nil {
		t.Fatal(err)
	}
	if _, err := oa.GetMethodByName("PositionSplit"); err != nil { // GetMethodByName matches events by name too
		t.Fatal(err)
	}

	// Order-book (CLOB) role: a 9-field order tuple.
	ex, err := os.ReadFile("testdata/clob_exchange.json")
	if err != nil {
		t.Fatal(err)
	}
	ea, err := ImportEthereumABI(ex)
	if err != nil {
		t.Fatalf("clob_exchange import: %v", err)
	}
	fo, err := ea.GetMethodByName("fillOrder")
	if err != nil {
		t.Fatal(err)
	}
	want := "fillOrder((uint256,address,address,address,uint256,uint256,uint256,uint8,bytes),uint256)"
	if got := fo.canonicalSignature(); got != want {
		t.Fatalf("fillOrder canonicalSignature=\n%q\nwant\n%q", got, want)
	}
}
