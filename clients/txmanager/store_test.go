package txmanager

import (
	"testing"

	"github.com/timshannon/badgerhold"
)

// newTestStore opens a real, temp-directory-backed badgerhold store. records
// reads through badgerhold.Query, which a hand-written fake cannot evaluate
// faithfully, so its tests run against the real thing rather than a fake.
func newTestStore(t *testing.T) Store {
	t.Helper()
	dir := t.TempDir()
	opts := badgerhold.DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir
	opts.Logger = nil
	db, err := badgerhold.Open(opts)
	if err != nil {
		t.Fatalf("open badgerhold: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestHighestNonceForReportsNotFoundOnAnEmptyStore(t *testing.T) {
	r := newRecords(newTestStore(t))

	_, found, err := r.highestNonceFor("0xaaa0000000000000000000000000000000000001")
	if err != nil {
		t.Fatalf("highestNonceFor: %v", err)
	}
	if found {
		t.Fatal("an empty store must report not found")
	}
}

func TestHighestNonceForReturnsTheHighestAcrossSeveralSaves(t *testing.T) {
	r := newRecords(newTestStore(t))
	const from = "0xaaa0000000000000000000000000000000000001"

	for i, nonce := range []int64{3, 7, 5} {
		if err := r.save(from, hashFor(i), nonce); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	got, found, err := r.highestNonceFor(from)
	if err != nil {
		t.Fatalf("highestNonceFor: %v", err)
	}
	if !found {
		t.Fatal("want found")
	}
	if got != 7 {
		t.Fatalf("highestNonceFor = %d, want 7", got)
	}
}

func TestHighestNonceForIsPerAddress(t *testing.T) {
	r := newRecords(newTestStore(t))
	if err := r.save("0xaaa0000000000000000000000000000000000001", hashFor(1), 9); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := r.save("0xbbb0000000000000000000000000000000000002", hashFor(2), 1); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, found, err := r.highestNonceFor("0xbbb0000000000000000000000000000000000002")
	if err != nil {
		t.Fatalf("highestNonceFor: %v", err)
	}
	if !found || got != 1 {
		t.Fatalf("highestNonceFor = (%d, %v), want (1, true)", got, found)
	}
}

func hashFor(i int) string {
	return "0xhash" + string(rune('a'+i))
}
