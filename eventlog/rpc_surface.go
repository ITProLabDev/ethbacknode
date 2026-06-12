package eventlog

// SubscribeAndSaveStrings registers a subscription from string parameters
// (as received over RPC) and persists it. The scope string is validated via
// ParseScope. This is the surface the endpoint's EventSubscriber wires to.
func (s *Service) SubscribeAndSaveStrings(serviceID, contractAddress, scope string) error {
	sc, err := ParseScope(scope)
	if err != nil {
		return err
	}
	return s.SubscribeAndSave(&Subscription{
		ServiceID:       serviceID,
		ContractAddress: contractAddress,
		Scope:           sc,
	})
}

// UnsubscribeStrings removes the subscription for (serviceID, contractAddress)
// and persists the new set. Removing a non-existent subscription is a no-op
// (not an error) so the call is idempotent.
func (s *Service) UnsubscribeStrings(serviceID, contractAddress string) error {
	s.subs.remove(serviceID, contractAddress)
	return s.saveSubscriptions()
}

// ListSubscriptions returns all subscriptions as string maps (serviceId,
// address, scope) for RPC responses.
func (s *Service) ListSubscriptions() []map[string]string {
	all := s.subs.all()
	out := make([]map[string]string, len(all))
	for i, sub := range all {
		out[i] = map[string]string{
			"serviceId": sub.ServiceID,
			"address":   sub.ContractAddress,
			"scope":     scopeName(sub.Scope),
		}
	}
	return out
}
