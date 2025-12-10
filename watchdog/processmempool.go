package watchdog

import (
	"github.com/ITProLabDev/ethbacknode/tools/log"
	"github.com/ITProLabDev/ethbacknode/types"
)

func (w *Service) processMemPool(pool []*types.TransferInfo) {
	//log.Dump("Process mempool:", pool)
	for _, tx := range pool {
		if w.config.Debug {
			log.Debug("Process transaction:", tx.TxID, "from:", tx.From, "to:", tx.To, "success:", tx.Success)
		}
		if !tx.Success {
			if w.config.Debug {
				log.Warning("Unsuccessful transaction:", tx.TxID, "from:", tx.From, "to:", tx.To, "skip...")
			}
			continue
		}
		_ = w.processTx(tx)
	}
}
