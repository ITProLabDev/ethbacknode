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

// NotifySubscriberRaw sends a notification with an arbitrary JSON payload to a
// subscriber (no Signer required). Used for contractEvent delivery.
func (s *Manager) NotifySubscriberRaw(serviceId ServiceId, subject string, payload interface{}) {
	s.subscribersMux.RLock()
	subscriber, found := s.subscribers[serviceId]
	s.subscribersMux.RUnlock()
	if !found {
		log.Error("NotifySubscriberRaw: unknown serviceId:", serviceId)
		return
	}
	subscriber.sendNotification(subject, payload, s.config.Debug)
}
