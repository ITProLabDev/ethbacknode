package address

func (p *Manager) updatePool() {
	p.mux.Lock()
	if len(p.allAddresses) != 0 {
		p.fastPool = newAddressMemStore(rawPool(p.allAddresses))
	}
	p.mux.Unlock()
}
