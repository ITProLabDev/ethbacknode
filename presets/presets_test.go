package presets

import (
	"testing"

	"github.com/ITProLabDev/ethbacknode/abi"
)

// fakeAdder records every contract handed to Add.
type fakeAdder struct {
	added []*abi.SmartContractInfo
}

func (f *fakeAdder) Add(c *abi.SmartContractInfo) { f.added = append(f.added, c) }

func TestApply_ArcChainRegistersAllEight(t *testing.T) {
	adder := &fakeAdder{}
	n, err := Apply(5042002, adder)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if n != 8 {
		t.Fatalf("loaded=%d want 8", n)
	}
	if len(adder.added) != 8 {
		t.Fatalf("added=%d want 8", len(adder.added))
	}
	// Spot-check one contract: MockUSDC carries decimals=6 and a parsed ABI.
	var usdc *abi.SmartContractInfo
	for _, c := range adder.added {
		if c.Name == "MockUSDC" {
			usdc = c
		}
	}
	if usdc == nil {
		t.Fatal("MockUSDC not registered")
	}
	if usdc.Decimals != 6 {
		t.Fatalf("MockUSDC decimals=%d want 6", usdc.Decimals)
	}
	if usdc.Symbol != "MockUSDC" {
		t.Fatalf("MockUSDC symbol=%q", usdc.Symbol)
	}
	if usdc.Abi == nil || len(usdc.Abi.Entries) == 0 {
		t.Fatal("MockUSDC ABI not parsed")
	}
}

func TestApply_UnknownChainRegistersNothing(t *testing.T) {
	adder := &fakeAdder{}
	n, err := Apply(1, adder) // Ethereum mainnet — no bundle
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if n != 0 || len(adder.added) != 0 {
		t.Fatalf("loaded=%d added=%d want 0/0", n, len(adder.added))
	}
}

func TestApply_IsIdempotentAcrossCalls(t *testing.T) {
	// Applying twice with the SAME adder must not error and must report 8 each
	// time. (The real registry dedups by address; the fake just records, so we
	// assert Apply itself is stable and side-effect-free beyond Add.)
	adder := &fakeAdder{}
	if n, err := Apply(5042002, adder); err != nil || n != 8 {
		t.Fatalf("first apply n=%d err=%v", n, err)
	}
	if n, err := Apply(5042002, adder); err != nil || n != 8 {
		t.Fatalf("second apply n=%d err=%v", n, err)
	}
	if len(adder.added) != 16 {
		t.Fatalf("added=%d want 16 (8+8; dedup is the registry's job)", len(adder.added))
	}
}

func TestApply_AllEmbeddedAbisAreValid(t *testing.T) {
	// Guards that every embedded ABI parses — a malformed bundle would make
	// Apply return an error mid-way.
	adder := &fakeAdder{}
	n, err := Apply(5042002, adder)
	if err != nil {
		t.Fatalf("an embedded ABI failed to import: %v", err)
	}
	if n != 8 {
		t.Fatalf("registered %d want 8", n)
	}
}

func TestApply_PreservesContractAddresses(t *testing.T) {
	adder := &fakeAdder{}
	if _, err := Apply(5042002, adder); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"JustifyAccessControl": "0xD45678caddB0301E8783D875a317C0cFEDa189Aa",
		"MockUSDC":             "0x8AAbb743F59dD725Cf16B06F53C1aDfB660969D8",
		"MarketFactory":        "0xd7f35035F6E6B42a1CB52e4AFc982896933cB20C",
		"MarketAMM":            "0xAbED8cB716ee30985df01b9641C1A01666b6c734",
	}
	got := map[string]string{}
	for _, c := range adder.added {
		got[c.Name] = c.ContractAddress
	}
	for name, addr := range want {
		if got[name] != addr {
			t.Fatalf("%s address=%q want %q", name, got[name], addr)
		}
	}
}
