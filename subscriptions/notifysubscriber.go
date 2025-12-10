package subscriptions

import "backnode/tools/log"

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
