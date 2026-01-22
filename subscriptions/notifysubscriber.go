package subscriptions

import "github.com/ITProLabDev/ethbacknode/tools/log"

// NotifySubscriber sends a notification to a specific subscriber.
// Looks up the subscriber by ID and sends the data to their endpoint.
func (s *Manager) NotifySubscriber(serviceId ServiceId, subject string, data Signer) {
	s.subscribersMux.RLock()
	defer s.subscribersMux.RUnlock()
	subscriber, found := s.subscribers[serviceId]
	if !found {
		log.Error("Unknown serviceId: ", serviceId)
		return
	}
	subscriber.sendNotification(subject, data, s.config.Debug)
}
