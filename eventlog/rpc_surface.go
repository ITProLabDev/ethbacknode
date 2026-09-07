package eventlog

// SubscribeAndSaveStrings registers a subscription from string parameters
// (as received over RPC) and persists it. The scope string is validated via
// ParseScope. selectors optionally filters contractTransaction delivery to
// only those method selectors (nil/empty = no filter, every transaction to
// the contract matches -- contractEvent is unaffected either way). This is
// the surface the endpoint's EventSubscriber wires to.
func (s *Service) SubscribeAndSaveStrings(serviceID, contractAddress, scope string, selectors [][4]byte) error {
	sc, err := ParseScope(scope)
	if err != nil {
		return err
	}
	return s.SubscribeAndSave(&Subscription{
		ServiceID:       serviceID,
		ContractAddress: contractAddress,
		Scope:           sc,
		Selectors:       selectors,
	})
}

// UnsubscribeStrings removes the subscription for (serviceID, contractAddress)
// and persists the new set. Removing a non-existent subscription is a no-op
// (not an error) so the call is idempotent.
func (s *Service) UnsubscribeStrings(serviceID, contractAddress string) error {
	s.subs.remove(serviceID, contractAddress)
	return s.saveSubscriptions()
}

// ListSubscriptions returns all subscriptions as string-keyed maps
// (serviceId, address, scope, selectors) for RPC responses. selectors is
// always present as a []string (possibly empty) of 0x-hex selectors.
func (s *Service) ListSubscriptions() []map[string]any {
	all := s.subs.all()
	out := make([]map[string]any, len(all))
	for i, sub := range all {
		selectors := formatSelectors(sub.Selectors)
		if selectors == nil {
			selectors = []string{}
		}
		out[i] = map[string]any{
			"serviceId": sub.ServiceID,
			"address":   sub.ContractAddress,
			"scope":     scopeName(sub.Scope),
			"selectors": selectors,
		}
	}
	return out
}
