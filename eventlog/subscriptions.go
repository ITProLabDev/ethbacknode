// Package eventlog collects, decodes, and scope-filters smart-contract event
// logs per block, handing decoded events to a delivery sink. It plugs into the
// watchdog as a block listener and keeps the abi/ engine a pure library.
package eventlog

import (
	"fmt"
	"strings"
	"sync"
)

// Scope selects which of a contract's events a subscription receives.
type Scope int

const (
	// ScopeWholeContract delivers every event of the contract, regardless of
	// which addresses are involved.
	ScopeWholeContract Scope = iota
	// ScopeManagedOnly delivers only events that involve a managed address
	// (matched inside the event's address-typed parameters).
	ScopeManagedOnly
)

// ParseScope parses a scope name (case-insensitive): "whole_contract" or
// "managed_only".
func ParseScope(s string) (Scope, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "whole_contract":
		return ScopeWholeContract, nil
	case "managed_only":
		return ScopeManagedOnly, nil
	default:
		return 0, fmt.Errorf("unknown event scope %q (want whole_contract | managed_only)", s)
	}
}

// Subscription is an in-memory contract-event subscription. ServiceID
// identifies the subscriber (for M6 delivery); ContractAddress is the contract
// whose events are wanted; Scope selects the filtering mode.
type Subscription struct {
	ServiceID       string
	ContractAddress string
	Scope           Scope

	// Selectors filters contractTransaction delivery to only transactions
	// whose method selector (the first 4 bytes of calldata) is in this set.
	// Empty means no filter: every transaction to the contract matches, the
	// pre-Milestone-8.1 behavior. contractEvent is unaffected -- this only
	// gates contractTransaction.
	Selectors [][4]byte
}

// subscriptionSet is a concurrency-safe in-memory set of subscriptions keyed by
// lowercased contract address.
type subscriptionSet struct {
	mu     sync.RWMutex
	byAddr map[string][]*Subscription
}

func newSubscriptionSet() *subscriptionSet {
	return &subscriptionSet{byAddr: make(map[string][]*Subscription)}
}

func (s *subscriptionSet) add(sub *Subscription) {
	key := strings.ToLower(sub.ContractAddress)
	s.mu.Lock()
	s.byAddr[key] = append(s.byAddr[key], sub)
	s.mu.Unlock()
}

// forAddress returns the subscriptions registered for a contract address
// (case-insensitive). The returned slice is a copy safe to read concurrently.
func (s *subscriptionSet) forAddress(addr string) []*Subscription {
	key := strings.ToLower(addr)
	s.mu.RLock()
	defer s.mu.RUnlock()
	src := s.byAddr[key]
	out := make([]*Subscription, len(src))
	copy(out, src)
	return out
}

// addresses returns the unique (lowercased) contract addresses with at least
// one subscription — used to build the eth_getLogs address filter.
func (s *subscriptionSet) addresses() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.byAddr))
	for addr := range s.byAddr {
		out = append(out, addr)
	}
	return out
}

// all returns a flat copy of every subscription across all addresses.
func (s *subscriptionSet) all() []*Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Subscription
	for _, subs := range s.byAddr {
		out = append(out, subs...)
	}
	return out
}

// remove drops the subscription matching (serviceID, contractAddress). If the
// address has no subscriptions left, its map entry is deleted so addresses()
// no longer reports it. A missing match is a no-op.
func (s *subscriptionSet) remove(serviceID, contractAddress string) {
	key := strings.ToLower(contractAddress)
	s.mu.Lock()
	defer s.mu.Unlock()
	src := s.byAddr[key]
	if len(src) == 0 {
		return
	}
	kept := src[:0]
	for _, sub := range src {
		if sub.ServiceID != serviceID {
			kept = append(kept, sub)
		}
	}
	if len(kept) == 0 {
		delete(s.byAddr, key)
		return
	}
	s.byAddr[key] = kept
}

// scopeName returns the canonical string name for a scope (inverse of
// ParseScope), used for persistence.
func scopeName(sc Scope) string {
	if sc == ScopeManagedOnly {
		return "managed_only"
	}
	return "whole_contract"
}
