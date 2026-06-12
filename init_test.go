package main

import (
	"sync"
	"testing"
)

// memStore is an in-memory storage.BinStorage for testing runInit.
type memStore struct {
	mu     sync.Mutex
	data   []byte
	exists bool
}

func (s *memStore) IsExists() bool { return s.exists }
func (s *memStore) Save(raw []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = append(s.data[:0], raw...)
	s.exists = true
	return nil
}
func (s *memStore) Load() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]byte, len(s.data))
	copy(out, s.data)
	return out, nil
}

func TestRunInit_WritesBothWhenAbsent(t *testing.T) {
	mainSt := &memStore{}
	clientSt := &memStore{}
	res, err := runInit("Arc", mainSt, clientSt)
	if err != nil {
		t.Fatal(err)
	}
	if !mainSt.exists || !clientSt.exists {
		t.Fatalf("both configs should be written: main=%v client=%v", mainSt.exists, clientSt.exists)
	}
	if !res.WroteMain || !res.WroteClient {
		t.Fatalf("result should report both written: %+v", res)
	}
}

func TestRunInit_SkipsExisting(t *testing.T) {
	mainSt := &memStore{exists: true, data: []byte("existing")}
	clientSt := &memStore{exists: true, data: []byte("existing")}
	res, err := runInit("Eth", mainSt, clientSt)
	if err != nil {
		t.Fatal(err)
	}
	if res.WroteMain || res.WroteClient {
		t.Fatalf("nothing should be overwritten: %+v", res)
	}
	if string(mainSt.data) != "existing" || string(clientSt.data) != "existing" {
		t.Fatal("existing config must be left untouched")
	}
}

func TestRunInit_PartialSkip(t *testing.T) {
	// main exists, client absent → only client written.
	mainSt := &memStore{exists: true, data: []byte("existing")}
	clientSt := &memStore{}
	res, err := runInit("Eth", mainSt, clientSt)
	if err != nil {
		t.Fatal(err)
	}
	if res.WroteMain {
		t.Fatal("main should be skipped (already exists)")
	}
	if !res.WroteClient || !clientSt.exists {
		t.Fatal("client should be written (was absent)")
	}
	if string(mainSt.data) != "existing" {
		t.Fatal("existing main config must be untouched")
	}
}

func TestRunInit_UnknownChain(t *testing.T) {
	res, err := runInit("dogecoin", &memStore{}, &memStore{})
	if err == nil {
		t.Fatal("unknown chain must error")
	}
	if res.WroteMain || res.WroteClient {
		t.Fatal("nothing should be written on unknown chain")
	}
}

func TestRunInit_ClientConfigCarriesChainIdentity(t *testing.T) {
	clientSt := &memStore{}
	if _, err := runInit("Arc", &memStore{}, clientSt); err != nil {
		t.Fatal(err)
	}
	// Arc identity must be in the written client config.
	raw, _ := clientSt.Load()
	if want := "Arc Testnet"; !containsSub(raw, want) {
		t.Fatalf("client config should contain %q, got: %s", want, raw)
	}
	if !containsSub(raw, "USDC") {
		t.Fatalf("client config should carry USDC symbol, got: %s", raw)
	}
}

func containsSub(haystack []byte, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && bytesContains(haystack, []byte(needle))
}

func bytesContains(h, n []byte) bool {
	for i := 0; i+len(n) <= len(h); i++ {
		if string(h[i:i+len(n)]) == string(n) {
			return true
		}
	}
	return false
}
