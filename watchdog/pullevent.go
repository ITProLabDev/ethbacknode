package watchdog

type PullEvent struct {
	txEvent    bool
	txId       string
	blockEvent bool
	blockId    string
}

//TODO: subscription to events for external modules

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

func (w *Service) processPullEvent(event *PullEvent) {
	//todo
}
