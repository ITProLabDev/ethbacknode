package address

// updatePool rebuilds the fast lookup index from all addresses.
// Should be called after bulk modifications to the address pool.
// Thread-safe operation.
func (p *Manager) updatePool() {
	p.mux.Lock()
	if len(p.allAddresses) != 0 {
		p.fastPool = newAddressMemStore(rawPool(p.allAddresses))
	}
	p.mux.Unlock()
}
