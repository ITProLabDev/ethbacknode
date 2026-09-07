package abi

import "testing"

// The well-known ERC-20 transfer selector, keccak256("transfer(address,uint256)")[:4]
// -- reproduced everywhere from Etherscan to Solidity tooling, so a mismatch
// here means the hashing itself is wrong, not just this helper.
func TestSelectorMatchesTheWellKnownErc20TransferSelector(t *testing.T) {
	got := Selector("transfer(address,uint256)")
	want := [4]byte{0xa9, 0x05, 0x9c, 0xbb}
	if got != want {
		t.Fatalf("Selector(transfer(address,uint256)) = %x, want %x", got, want)
	}
}

func TestSelectorMatchesEntrySignature(t *testing.T) {
	entry := &SmartContractAbiEntry{
		Type: "function",
		Name: "transfer",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "to", Type: "address"},
			{Name: "amount", Type: "uint256"},
		},
	}
	if got, want := Selector("transfer(address,uint256)"), entry.GetSignature(); got != want {
		t.Fatalf("Selector = %x, entry.GetSignature() = %x, want equal", got, want)
	}
}

func TestSelectorDiffersForOverloadedNames(t *testing.T) {
	a := Selector("transfer(address,uint256)")
	b := Selector("transfer(address,uint256,bytes)")
	if a == b {
		t.Fatal("two different signatures with the same bare name must not collide")
	}
}
