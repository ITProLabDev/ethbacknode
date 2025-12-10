package watchdog

import "backnode/tools/log"

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
