package subscriptions

func (s *Manager) eventLoop() {
	for {
		eventProcessor := <-s.eventPipe
		eventProcessor()
	}
}
