package eventlog

import (
	"encoding/json"

	"github.com/ITProLabDev/ethbacknode/storage"
)

// persistedSub is the JSON form of a subscription (Scope as its string name).
type persistedSub struct {
	ServiceID       string `json:"serviceId"`
	ContractAddress string `json:"contractAddress"`
	Scope           string `json:"scope"`
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
		s.subs.add(&Subscription{ServiceID: p.ServiceID, ContractAddress: p.ContractAddress, Scope: scope})
	}
	return nil
}
