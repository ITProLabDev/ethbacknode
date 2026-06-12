package eventlog

import "testing"

func TestSubscribeAndSaveStrings(t *testing.T) {
	st := &memStore{}
	svc := New(WithSubscriptionStorage(st))
	if err := svc.SubscribeAndSaveStrings("7", "0xABC", "managed_only"); err != nil {
		t.Fatal(err)
	}
	subs := svc.subs.forAddress("0xabc")
	if len(subs) != 1 || subs[0].ServiceID != "7" || subs[0].Scope != ScopeManagedOnly {
		t.Fatalf("subs=%+v", subs)
	}
	if !st.exists {
		t.Fatal("must persist")
	}
}

func TestSubscribeAndSaveStrings_BadScope(t *testing.T) {
	svc := New()
	if err := svc.SubscribeAndSaveStrings("7", "0xABC", "bogus"); err == nil {
		t.Fatal("bad scope must error")
	}
}

func TestListSubscriptions(t *testing.T) {
	svc := New()
	_ = svc.SubscribeAndSaveStrings("1", "0xAA", "whole_contract")
	_ = svc.SubscribeAndSaveStrings("2", "0xBB", "managed_only")
	list := svc.ListSubscriptions()
	if len(list) != 2 {
		t.Fatalf("list=%v", list)
	}
	for _, m := range list {
		if m["serviceId"] == "" || m["address"] == "" || m["scope"] == "" {
			t.Fatalf("incomplete entry: %v", m)
		}
	}
}

func TestUnsubscribeStrings(t *testing.T) {
	st := &memStore{}
	svc := New(WithSubscriptionStorage(st))
	_ = svc.SubscribeAndSaveStrings("7", "0xAA", "whole_contract")
	_ = svc.SubscribeAndSaveStrings("8", "0xAA", "managed_only")
	if err := svc.UnsubscribeStrings("7", "0xAA"); err != nil {
		t.Fatal(err)
	}
	subs := svc.subs.forAddress("0xaa")
	if len(subs) != 1 || subs[0].ServiceID != "8" {
		t.Fatalf("after unsubscribe subs=%+v", subs)
	}
	if err := svc.UnsubscribeStrings("8", "0xAA"); err != nil {
		t.Fatal(err)
	}
	if len(svc.subs.forAddress("0xaa")) != 0 {
		t.Fatal("address should have no subscriptions left")
	}
}
