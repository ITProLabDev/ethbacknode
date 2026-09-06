package abi

import (
	"strings"
	"sync"
	"testing"
)

// Regression tests for four registry defects plus a signature lazy-init race,
// all pre-existing in addUnsafe/Save/the three index lookups/GetSignature.
// See docs/PROJECT_STATUS.md §5 ("abi.addUnsafe duplicate check is
// case-sensitive"). Root cause and fix verified against a sibling project
// (git.voodoocore.io/core/mdex) that forked this package and fixed the same
// defects.

// addUnsafe compared addresses byte-for-byte while byAddress is keyed
// lowercase, so two different contracts registered at the same address in
// different letter casing were NOT recognized as a collision: both were
// appended to m.contracts, and afterLoad's lowercase-keyed byAddress map kept
// only whichever was added last, stranding the other unreachable by address
// lookup even though it still shows up in GetSmartContractList.
func TestAddUnsafeTreatsCaseVariantAddressAsCollision(t *testing.T) {
	m := newTestManager(t)
	before := len(m.GetSmartContractList())

	const checksummed = "0xAbC0000000000000000000000000000000000001"
	m.Add(&SmartContractInfo{Name: "Token", Symbol: "TOK", ContractAddress: checksummed})
	if got := len(m.GetSmartContractList()); got != before+1 {
		t.Fatalf("after first Add: got %d contracts, want %d", got, before+1)
	}

	// A different contract at the SAME address, spelled lowercase, must be
	// rejected as an address collision rather than silently appended.
	m.Add(&SmartContractInfo{Name: "Other", Symbol: "OTH", ContractAddress: strings.ToLower(checksummed)})

	list := m.GetSmartContractList()
	if got := len(list); got != before+1 {
		t.Fatalf("after colliding Add: got %d contracts, want %d (list: %v)", got, before+1, list)
	}
	if _, ok := list["Token"]; !ok {
		t.Fatalf("the original registration was lost: %v", list)
	}
	if _, ok := list["Other"]; ok {
		t.Fatalf("the colliding registration should have been rejected, not appended: %v", list)
	}

	// The surviving entry must still be reachable by address, in either case.
	if _, err := m.GetSmartContractByAddress(checksummed); err != nil {
		t.Errorf("GetSmartContractByAddress(checksummed): %v", err)
	}
	if _, err := m.GetSmartContractByAddress(strings.ToLower(checksummed)); err != nil {
		t.Errorf("GetSmartContractByAddress(lowercase): %v", err)
	}
}

// unsyncStorage is a BinStorage fake with no internal locking of its own, so a
// missing lock in SmartContractsManager.Save shows up as a data race on this
// type's fields under `go test -race`, rather than being masked by a lock the
// fake happens to take (unlike memStorage, used elsewhere in this package).
type unsyncStorage struct {
	data   []byte
	exists bool
}

func (s *unsyncStorage) IsExists() bool        { return s.exists }
func (s *unsyncStorage) Load() ([]byte, error) { return s.data, nil }
func (s *unsyncStorage) Save(raw []byte) error {
	s.data = raw
	s.exists = true
	return nil
}

// Save took mux only for reading, which two goroutines may do at the same
// time by definition, so nothing serialized the storage writes themselves:
// two concurrent Saves could both marshal and both call storage.Save at once.
// Run with -race; a missing saveMu is reported as a data race on
// unsyncStorage's fields, not caught by a functional assertion.
func TestConcurrentSaveDoesNotRace(t *testing.T) {
	st := &unsyncStorage{}
	m := NewManager(WithStorage(st), WithAddressCodec(&hexCodec{}))
	m.Add(&SmartContractInfo{Name: "Race", Symbol: "RC", ContractAddress: "0x1111111111111111111111111111111111111111"})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = m.Save()
		}()
	}
	wg.Wait()
}

// GetSignature (and Topic0, which shares the same lazy init) used an
// all-zero-bytes check to decide whether to compute the signature, which two
// goroutines can pass at the same time: the write to e.Signature was
// unsynchronized. Run with -race.
func TestSignatureLazyInitIsRaceFree(t *testing.T) {
	entry := &SmartContractAbiEntry{
		Type: "function",
		Name: "totalSupply",
		Outputs: []*SmartContractAbiEntryOutput{
			{Type: "uint256"},
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = entry.GetSignature()
		}()
	}
	wg.Wait()
}
