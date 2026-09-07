package eventlog

import (
	"sync"
	"testing"
)

// memStore is an in-memory storage.BinStorage for tests.
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

func TestService_SubscribePersists(t *testing.T) {
	st := &memStore{}
	svc := New(WithSubscriptionStorage(st))
	if err := svc.SubscribeAndSave(&Subscription{ServiceID: "s1", ContractAddress: "0xAA", Scope: ScopeWholeContract}); err != nil {
		t.Fatal(err)
	}
	if !st.exists {
		t.Fatal("subscription should have been persisted")
	}

	// A fresh service loading the same storage must see the subscription.
	svc2 := New(WithSubscriptionStorage(st))
	if err := svc2.LoadSubscriptions(); err != nil {
		t.Fatal(err)
	}
	subs := svc2.subs.forAddress("0xaa")
	if len(subs) != 1 || subs[0].ServiceID != "s1" || subs[0].Scope != ScopeWholeContract {
		t.Fatalf("loaded subs=%+v", subs)
	}
}

func TestService_SubscribePersistsSelectors(t *testing.T) {
	st := &memStore{}
	svc := New(WithSubscriptionStorage(st))
	sel := [4]byte{0xa9, 0x05, 0x9c, 0xbb}
	if err := svc.SubscribeAndSave(&Subscription{
		ServiceID: "s1", ContractAddress: "0xAA", Scope: ScopeWholeContract,
		Selectors: [][4]byte{sel},
	}); err != nil {
		t.Fatal(err)
	}

	svc2 := New(WithSubscriptionStorage(st))
	if err := svc2.LoadSubscriptions(); err != nil {
		t.Fatal(err)
	}
	subs := svc2.subs.forAddress("0xaa")
	if len(subs) != 1 {
		t.Fatalf("loaded subs=%+v", subs)
	}
	if len(subs[0].Selectors) != 1 || subs[0].Selectors[0] != sel {
		t.Fatalf("selectors did not round-trip: got %x, want [%x]", subs[0].Selectors, sel)
	}
}

func TestService_SubscribePersistsNoSelectorsAsEmpty(t *testing.T) {
	st := &memStore{}
	svc := New(WithSubscriptionStorage(st))
	if err := svc.SubscribeAndSave(&Subscription{ServiceID: "s1", ContractAddress: "0xAA", Scope: ScopeWholeContract}); err != nil {
		t.Fatal(err)
	}

	svc2 := New(WithSubscriptionStorage(st))
	if err := svc2.LoadSubscriptions(); err != nil {
		t.Fatal(err)
	}
	subs := svc2.subs.forAddress("0xaa")
	if len(subs) != 1 || len(subs[0].Selectors) != 0 {
		t.Fatalf("expected no selectors to round-trip as empty, got %+v", subs)
	}
}

func TestService_LoadSubscriptions_EmptyStorage(t *testing.T) {
	st := &memStore{} // never written
	svc := New(WithSubscriptionStorage(st))
	if err := svc.LoadSubscriptions(); err != nil {
		t.Fatalf("loading empty storage must not error: %v", err)
	}
	if len(svc.subs.addresses()) != 0 {
		t.Fatal("empty storage should yield no subscriptions")
	}
}

func TestService_LoadSubscriptions_BadJSON(t *testing.T) {
	st := &memStore{data: []byte(`not valid json`), exists: true}
	svc := New(WithSubscriptionStorage(st))
	if err := svc.LoadSubscriptions(); err == nil {
		t.Fatal("malformed JSON must error")
	}
}

func TestService_LoadSubscriptions_BadScope(t *testing.T) {
	badJSON := `[{"serviceId":"s1","contractAddress":"0xAA","scope":"invalid_scope"}]`
	st := &memStore{data: []byte(badJSON), exists: true}
	svc := New(WithSubscriptionStorage(st))
	if err := svc.LoadSubscriptions(); err == nil {
		t.Fatal("bad scope in persisted data must error")
	}
}

func TestService_LoadSubscriptions_BadSelectorHex(t *testing.T) {
	badJSON := `[{"serviceId":"s1","contractAddress":"0xAA","scope":"whole_contract","selectors":["not-hex"]}]`
	st := &memStore{data: []byte(badJSON), exists: true}
	svc := New(WithSubscriptionStorage(st))
	if err := svc.LoadSubscriptions(); err == nil {
		t.Fatal("malformed selector hex in persisted data must error")
	}
}
