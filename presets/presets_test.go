package presets

import (
	"errors"
	"testing"

	"github.com/ITProLabDev/ethbacknode/abi"
)

// fakeAdder records every contract handed to Add and can simulate a failure.
type fakeAdder struct {
	added  []*abi.SmartContractInfo
	failOn string // contract name to fail validation for (unused here)
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

var _ = errors.New
