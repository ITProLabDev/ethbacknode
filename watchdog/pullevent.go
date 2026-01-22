package watchdog

// PullEvent represents an external event request for transaction or block lookup.
type PullEvent struct {
	txEvent    bool
	txId       string
	blockEvent bool
	blockId    string
}

//TODO: subscription to events for external modules

// pullEventWatcher listens for external pull event requests.
// Processes events from the pull channel until the service is stopped.
func (w *Service) pullEventWatcher() {
	for {
		select {
		case event := <-w.pullEventChannel:
			w.processPullEvent(event)
		case _, closed := <-w.quit:
			if closed {
				return
			}
		}
	}
}

// processPullEvent handles an external pull event request.
// TODO: Implementation pending.
func (w *Service) processPullEvent(event *PullEvent) {
	//todo
}
