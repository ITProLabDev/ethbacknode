package watchdog

import (
	"github.com/ITProLabDev/ethbacknode/tools/log"
	"time"
)

// runLoop is the main monitoring loop that polls the blockchain.
// Checks mempool content and processes new blocks at configured intervals.
// Handles block catch-up when multiple blocks have been missed.
func (w *Service) runLoop() {
	blockchain := w.client.GetChainName()
	lastSeenBlock := w.state.LastBlockNum
	if lastSeenBlock == 0 {
		log.Warning("Restart from block: 0")
	} else {
		log.Info("Restart from block:", lastSeenBlock)
	}
	for {
		w.mux.Lock()
		memPoolContent, err := w.client.MemPoolContent()
		if err != nil {
			log.Error("Can not get mempool content:", err)
			w.mux.Unlock()
			time.Sleep(time.Duration(w.checkInterval) * time.Second)
			continue
		}
		if len(memPoolContent) != 0 {
			if w.config.Debug {
				log.Debug("MemPool contain", len(memPoolContent), "transactions, process...")
			}
			w.processMemPool(memPoolContent)
		} else {
			if w.config.Debug {
				log.Debug("MemPool is empty")
			}
		}
		currentBlock, err := w.client.BlockNum()
		if err != nil {
			log.Error("Can not get current block:", err)
			time.Sleep(time.Duration(w.checkInterval) * time.Second)
			w.mux.Unlock()
			continue
		}
		if currentBlock > lastSeenBlock {
			log.Info("Current", blockchain, "block:", currentBlock)
			if currentBlock-lastSeenBlock > 1 {
				log.Warning("Blocks ahead:", currentBlock-lastSeenBlock, "overtake or missed blocks")
				for processBlock := lastSeenBlock + 1; processBlock <= currentBlock; processBlock++ {
					if w.config.Debug {
						log.Debug("Process block:", processBlock)
					}
					err = w.processBlock(processBlock)
					if err != nil {
						log.Error("Can not process block:", processBlock, err)
						// Stop this tick's catch-up here rather than skipping
						// the failed block: lastSeenBlock (and so the
						// persisted state below) must stay at the last block
						// that actually succeeded, so the next tick retries
						// from this one instead of silently treating it as
						// processed.
						break
					}
					w.state.UpdateState(processBlock)
					lastSeenBlock = processBlock
				}
			} else {
				err = w.processBlock(currentBlock)
				if err != nil {
					log.Error("Can not process block:", currentBlock, err)
				} else {
					lastSeenBlock = currentBlock
				}
			}
		} else {
			if w.config.Debug {
				log.Debug("No new blocks, skip...")
			}
		}
		_ = w.state.UpdateState(lastSeenBlock)
		w.mux.Unlock()
		time.Sleep(time.Duration(w.checkInterval) * time.Second)
	}
}
