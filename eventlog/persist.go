package eventlog

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ITProLabDev/ethbacknode/storage"
)

// persistedSub is the JSON form of a subscription (Scope as its string name,
// Selectors as 0x-hex strings).
type persistedSub struct {
	ServiceID       string   `json:"serviceId"`
	ContractAddress string   `json:"contractAddress"`
	Scope           string   `json:"scope"`
	Selectors       []string `json:"selectors,omitempty"`
}

// formatSelectors renders selectors as 0x-hex strings for persistence.
func formatSelectors(sels [][4]byte) []string {
	if len(sels) == 0 {
		return nil
	}
	out := make([]string, len(sels))
	for i, s := range sels {
		out[i] = "0x" + hex.EncodeToString(s[:])
	}
	return out
}

// parseSelectors parses persisted 0x-hex selector strings back into [4]byte
// values, rejecting anything that is not exactly 4 bytes of hex.
func parseSelectors(hexes []string) ([][4]byte, error) {
	if len(hexes) == 0 {
		return nil, nil
	}
	out := make([][4]byte, len(hexes))
	for i, h := range hexes {
		b, err := hex.DecodeString(strings.TrimPrefix(h, "0x"))
		if err != nil {
			return nil, fmt.Errorf("eventlog: selector %q: %w", h, err)
		}
		if len(b) != 4 {
			return nil, fmt.Errorf("eventlog: selector %q is %d bytes, want 4", h, len(b))
		}
		copy(out[i][:], b)
	}
	return out, nil
}

// WithSubscriptionStorage sets the storage backend used to persist event
// subscriptions.
func WithSubscriptionStorage(st storage.BinStorage) Option {
	return func(svc *Service) { svc.subStorage = st }
}

// SubscribeAndSave registers a subscription and persists the full set.
func (s *Service) SubscribeAndSave(sub *Subscription) error {
	s.subs.add(sub)
	return s.saveSubscriptions()
}

// saveSubscriptions writes all subscriptions to storage (no-op if no storage).
func (s *Service) saveSubscriptions() error {
	if s.subStorage == nil {
		return nil
	}
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	all := s.subs.all()
	out := make([]persistedSub, len(all))
	for i, sub := range all {
		out[i] = persistedSub{
			ServiceID:       sub.ServiceID,
			ContractAddress: sub.ContractAddress,
			Scope:           scopeName(sub.Scope),
			Selectors:       formatSelectors(sub.Selectors),
		}
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return s.subStorage.Save(data)
}

// LoadSubscriptions loads persisted subscriptions from storage into the set.
// A missing/empty store is not an error. Subscriptions are APPENDED to the
// current set (safe for the one-time startup load; calling it twice duplicates).
func (s *Service) LoadSubscriptions() error {
	if s.subStorage == nil || !s.subStorage.IsExists() {
		return nil
	}
	data, err := s.subStorage.Load()
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	var loaded []persistedSub
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}
	for _, p := range loaded {
		scope, err := ParseScope(p.Scope)
		if err != nil {
			return err
		}
		selectors, err := parseSelectors(p.Selectors)
		if err != nil {
			return err
		}
		s.subs.add(&Subscription{
			ServiceID: p.ServiceID, ContractAddress: p.ContractAddress, Scope: scope,
			Selectors: selectors,
		})
	}
	return nil
}
