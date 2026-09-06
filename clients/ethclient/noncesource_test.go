package ethclient

import (
	"errors"
	"testing"
)

type fakeNonceSource struct {
	allocateNonce int64
	allocateErr   error

	sentCalls int
	sentFrom  string
	sentHash  string
	sentNonce int64

	releaseCalls  int
	releasedFrom  string
	releasedNonce int64
}

func (f *fakeNonceSource) Allocate(from string) (int64, error) {
	return f.allocateNonce, f.allocateErr
}

func (f *fakeNonceSource) Sent(from, txHash string, nonce int64) {
	f.sentCalls++
	f.sentFrom, f.sentHash, f.sentNonce = from, txHash, nonce
}

func (f *fakeNonceSource) Release(from string, nonce int64) {
	f.releaseCalls++
	f.releasedFrom, f.releasedNonce = from, nonce
}

func TestWithAllocatedNonceRecordsSentOnSuccess(t *testing.T) {
	ns := &fakeNonceSource{allocateNonce: 7}
	c := &Client{nonceSource: ns}

	hash, err := c.withAllocatedNonce("0xfrom", func(nonce int64) (string, error) {
		if nonce != 7 {
			t.Fatalf("fn got nonce %d, want 7", nonce)
		}
		return "0xhash", nil
	})
	if err != nil {
		t.Fatalf("withAllocatedNonce: %v", err)
	}
	if hash != "0xhash" {
		t.Fatalf("hash = %q, want 0xhash", hash)
	}
	if ns.sentCalls != 1 || ns.sentFrom != "0xfrom" || ns.sentHash != "0xhash" || ns.sentNonce != 7 {
		t.Fatalf("Sent not recorded correctly: %+v", ns)
	}
	if ns.releaseCalls != 0 {
		t.Fatalf("Release must not be called on success, got %d calls", ns.releaseCalls)
	}
}

func TestWithAllocatedNonceReleasesOnFailure(t *testing.T) {
	ns := &fakeNonceSource{allocateNonce: 9}
	c := &Client{nonceSource: ns}

	sendErr := errors.New("broadcast rejected")
	_, err := c.withAllocatedNonce("0xfrom", func(nonce int64) (string, error) {
		return "", sendErr
	})
	if !errors.Is(err, sendErr) {
		t.Fatalf("err = %v, want %v", err, sendErr)
	}
	if ns.releaseCalls != 1 || ns.releasedFrom != "0xfrom" || ns.releasedNonce != 9 {
		t.Fatalf("Release not called correctly: %+v", ns)
	}
	if ns.sentCalls != 0 {
		t.Fatalf("Sent must not be called on failure, got %d calls", ns.sentCalls)
	}
}

func TestWithAllocatedNoncePropagatesAllocateError(t *testing.T) {
	allocErr := errors.New("no chain wired")
	ns := &fakeNonceSource{allocateErr: allocErr}
	c := &Client{nonceSource: ns}

	called := false
	_, err := c.withAllocatedNonce("0xfrom", func(nonce int64) (string, error) {
		called = true
		return "", nil
	})
	if !errors.Is(err, allocErr) {
		t.Fatalf("err = %v, want %v", err, allocErr)
	}
	if called {
		t.Fatal("fn must not run when allocation fails")
	}
	if ns.sentCalls != 0 || ns.releaseCalls != 0 {
		t.Fatalf("neither Sent nor Release should fire when Allocate fails: %+v", ns)
	}
}

func TestAllocateNonceFallsBackToTheNodeWhenNoSourceIsWired(t *testing.T) {
	c, _ := newFakeClient(`"0x5"`)
	nonce, err := c.allocateNonce("0xfrom")
	if err != nil {
		t.Fatalf("allocateNonce: %v", err)
	}
	if nonce != 5 {
		t.Fatalf("allocateNonce fallback = %d, want 5 (from PendingNonceAt)", nonce)
	}
}
