package subscriptions

// eventLoop processes events from the event pipe sequentially.
// Runs as a goroutine, executing event handlers one at a time.
func (s *Manager) eventLoop() {
	for {
		eventProcessor := <-s.eventPipe
		eventProcessor()
	}
}
