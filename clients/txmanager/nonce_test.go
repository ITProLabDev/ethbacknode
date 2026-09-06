package txmanager

import (
	"fmt"
	"sort"
	"sync"
	"testing"
)

// fakeChain is a ChainNonces double whose pending nonce is fixed per address.
type fakeChain struct {
	mu      sync.Mutex
	pending map[string]int64
}

func newFakeChain() *fakeChain { return &fakeChain{pending: make(map[string]int64)} }

func (f *fakeChain) PendingNonceAt(address string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.pending[address], nil
}

const addrA = "0xaaa0000000000000000000000000000000000001"

func newTestNonceManager(t *testing.T, chain ChainNonces) *NonceManager {
	t.Helper()
	return NewNonceManager(WithChain(chain), WithStore(newTestStore(t)))
}

func TestAllocateSeedsFromTheNodeWhenNoRecordsExist(t *testing.T) {
	chain := newFakeChain()
	chain.pending[addrA] = 5

	m := newTestNonceManager(t, chain)

	got, err := m.Allocate(addrA)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if got != 5 {
		t.Fatalf("Allocate = %d, want 5 (the node's pending count)", got)
	}
}

func TestAllocateSeedsFromRecordsWhenHigherThanTheNode(t *testing.T) {
	chain := newFakeChain()
	chain.pending[addrA] = 2 // the node has not caught up with what was actually sent

	store := newTestStore(t)
	if err := newRecords(store).save(addrA, "0xhash1", 7); err != nil {
		t.Fatalf("seed record: %v", err)
	}

	m := NewNonceManager(WithChain(chain), WithStore(store))

	got, err := m.Allocate(addrA)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if got != 8 {
		t.Fatalf("Allocate = %d, want 8 (one past the highest recorded nonce)", got)
	}
}

func TestAllocateIsSequentialAndDistinctUnderConcurrency(t *testing.T) {
	chain := newFakeChain()
	chain.pending[addrA] = 0
	m := newTestNonceManager(t, chain)

	const n = 20
	results := make([]int64, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			nonce, err := m.Allocate(addrA)
			if err != nil {
				t.Errorf("Allocate: %v", err)
				return
			}
			results[i] = nonce
		}(i)
	}
	wg.Wait()

	sort.Slice(results, func(i, j int) bool { return results[i] < results[j] })
	seen := make(map[int64]bool, n)
	for _, r := range results {
		if seen[r] {
			t.Fatalf("nonce %d handed out more than once: %v", r, results)
		}
		seen[r] = true
	}
	for i, r := range results {
		if r != int64(i) {
			t.Fatalf("results are not the contiguous set [0,%d): %v", n, results)
		}
	}
}

func TestReleaseOfTheMostRecentNonceRewindsTheCursor(t *testing.T) {
	chain := newFakeChain()
	m := newTestNonceManager(t, chain)

	first, err := m.Allocate(addrA)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	m.Release(addrA, first)

	got, err := m.Allocate(addrA)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if got != first {
		t.Fatalf("Allocate after Release = %d, want %d (the released nonce reused)", got, first)
	}
}

func TestReleaseOfAHoleIsHandedOutBeforeTheCursorAdvancesFurther(t *testing.T) {
	chain := newFakeChain()
	m := newTestNonceManager(t, chain)

	first, err := m.Allocate(addrA)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if _, err := m.Allocate(addrA); err != nil { // second, kept
		t.Fatalf("Allocate: %v", err)
	}
	third, err := m.Allocate(addrA)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}

	m.Release(addrA, first) // a hole below the cursor, not the most recent

	got, err := m.Allocate(addrA)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if got != first {
		t.Fatalf("Allocate after releasing a hole = %d, want %d (the hole), not %d (advancing past it)", got, first, third+1)
	}
}

func TestAllocateTreatsAnAddressAsOneCursorRegardlessOfCase(t *testing.T) {
	chain := newFakeChain()
	const checksummed = "0xAbC0000000000000000000000000000000000001"
	chain.pending[checksummed] = 0
	chain.pending["0xabc0000000000000000000000000000000000001"] = 0
	m := newTestNonceManager(t, chain)

	first, err := m.Allocate(checksummed)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	second, err := m.Allocate("0xabc0000000000000000000000000000000000001")
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if second != first+1 {
		t.Fatalf("Allocate(lowercase) = %d, want %d (continuing the same cursor as the checksummed form)", second, first+1)
	}
}

func TestSentPersistsSoARestartDoesNotReuseANonce(t *testing.T) {
	chain := newFakeChain()
	chain.pending[addrA] = 0
	store := newTestStore(t)

	m1 := NewNonceManager(WithChain(chain), WithStore(store))
	for i := 0; i < 3; i++ {
		nonce, err := m1.Allocate(addrA)
		if err != nil {
			t.Fatalf("Allocate: %v", err)
		}
		m1.Sent(addrA, fmt.Sprintf("0xhash%d", i), nonce)
	}

	// Simulate a restart: a fresh manager over the same store, with the node
	// still reporting its old pending count (it has not caught up yet either).
	m2 := NewNonceManager(WithChain(chain), WithStore(store))
	got, err := m2.Allocate(addrA)
	if err != nil {
		t.Fatalf("Allocate after restart: %v", err)
	}
	if got != 3 {
		t.Fatalf("Allocate after restart = %d, want 3 (one past the highest sent nonce)", got)
	}
}

func TestAllocateWithoutChainOrStoreReturnsAClearError(t *testing.T) {
	if _, err := NewNonceManager(WithStore(newTestStore(t))).Allocate(addrA); err != ErrChainNotSet {
		t.Fatalf("Allocate without a chain: got %v, want ErrChainNotSet", err)
	}
	if _, err := NewNonceManager(WithChain(newFakeChain())).Allocate(addrA); err != ErrStoreNotSet {
		t.Fatalf("Allocate without a store: got %v, want ErrStoreNotSet", err)
	}
}
