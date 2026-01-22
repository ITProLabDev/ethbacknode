package watchdog

import "github.com/ITProLabDev/ethbacknode/tools/log"

// processBlock retrieves a block by number and processes all its transactions.
// Emits a block event and processes each transaction for managed addresses.
func (w *Service) processBlock(blockNum int64) (err error) {
	if w.config.Debug {
		log.Debug("Process block", blockNum)
	}
	block, err := w.client.BlockByNum(blockNum, true)
	if w.config.Debug {
		if err != nil {
			log.Error("Get Block Error: error", err)
		}
	}
	if err != nil {
		return err
	}
	w.events <- &event{
		blockEvent: true,
		blockNum:   blockNum,
		blockId:    block.BlockID,
		blockTime:  block.Timestamp,
	}
	for _, tx := range block.Transactions {
		if err := w.processTx(tx); err != nil {
			log.Error("process tx error", err)
		}
	}
	return nil
}
