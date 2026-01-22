package address

import (
	"fmt"

	"github.com/ITProLabDev/ethbacknode/tools/log"
)

// DevCheckMemPool is a development utility that verifies the fast pool consistency.
// Checks that all addresses in allAddresses can be found in the fast lookup pool.
func (p *Manager) DevCheckMemPool() {
	for _, row := range p.allAddresses {
		fromFastPool, ok := p.fastPool.LookupString(row.Address)
		if ok {
			fmt.Println("address:", row.Address, "found in fast pool ok, is subscribed:", fromFastPool.Subscribed)
		}
	}
}

// DevDumpMemPool is a development utility that logs all addresses in the pool.
// Outputs address string, subscription state, service/user/invoice IDs, and watch state.
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
