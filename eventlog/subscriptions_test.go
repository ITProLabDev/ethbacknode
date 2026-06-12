package eventlog

import "testing"

func TestScope_Parse(t *testing.T) {
	cases := map[string]Scope{
		"whole_contract": ScopeWholeContract,
		"WHOLE_CONTRACT": ScopeWholeContract,
		"managed_only":   ScopeManagedOnly,
		"Managed_Only":   ScopeManagedOnly,
	}
	for in, want := range cases {
		got, err := ParseScope(in)
		if err != nil || got != want {
			t.Fatalf("ParseScope(%q)=%v,%v want %v", in, got, err, want)
		}
	}
	if _, err := ParseScope("nonsense"); err == nil {
		t.Fatal("unknown scope must error")
	}
}

func TestSubscriptionSet_RegisterAndMatch(t *testing.T) {
	set := newSubscriptionSet()
	addr := "0xAbC0000000000000000000000000000000000001"
	set.add(&Subscription{ServiceID: "svc1", ContractAddress: addr, Scope: ScopeWholeContract})
	set.add(&Subscription{ServiceID: "svc2", ContractAddress: addr, Scope: ScopeManagedOnly})

	subs := set.forAddress("0xabc0000000000000000000000000000000000001")
	if len(subs) != 2 {
		t.Fatalf("forAddress=%d want 2", len(subs))
	}
	if len(set.forAddress("0x9999999999999999999999999999999999999999")) != 0 {
		t.Fatal("unknown address must match nothing")
	}
}

func TestSubscriptionSet_Addresses(t *testing.T) {
	set := newSubscriptionSet()
	set.add(&Subscription{ServiceID: "a", ContractAddress: "0xAA", Scope: ScopeWholeContract})
	set.add(&Subscription{ServiceID: "b", ContractAddress: "0xaa", Scope: ScopeManagedOnly})
	set.add(&Subscription{ServiceID: "c", ContractAddress: "0xBB", Scope: ScopeWholeContract})
	addrs := set.addresses()
	if len(addrs) != 2 {
		t.Fatalf("addresses=%v want 2 unique (0xaa,0xbb lowercased)", addrs)
	}
}
