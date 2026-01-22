package watchdog

// jsVmInit initializes the JavaScript VM for custom scripting.
// Currently disabled (goja commented out).
func (w *Service) jsVmInit() {
	//w.jsVm = goja.New()
}
