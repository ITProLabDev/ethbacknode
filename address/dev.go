package address

import (
	"fmt"
	"module github.com/ITProLabDev/ethbacknode/tools/log"
)

func (p *Manager) DevCheckMemPool() {
	for _, row := range p.allAddresses {
		fromFastPool, ok := p.fastPool.LookupString(row.Address)
		if ok {
			fmt.Println("address:", row.Address, "found in fast pool ok, is subscribed:", fromFastPool.Subscribed)
		}
	}
}

func (p *Manager) DevDumpMemPool() {
	for _, row := range p.allAddresses {
		var subscribed, watchState string
		if row.Subscribed {
			subscribed = "subscribed"
		} else {
			subscribed = "free"
		}
		if row.WatchOnly {
			watchState = "watch only"
		} else {
			watchState = "normal"
		}
		log.Warning("*", row.Address, "state:", subscribed, row.ServiceId, row.UserId, row.InvoiceId, watchState)
	}
}
