package eventlog

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ITProLabDev/ethbacknode/tools/flow"
)

// collect gathers the block's logs according to the configured mode.
func (s *Service) collect(blockNum int64, addrs []string) ([]*ethLog, error) {
	switch s.cfg.Mode {
	case ModeReceipts:
		return s.collectModeReceipts(blockNum, addrs)
	default:
		return s.collectModeGetLogs(blockNum, addrs)
	}
}

// collectModeGetLogs fetches the block's logs with one eth_getLogs call,
// filtered to the subscribed contract addresses.
func (s *Service) collectModeGetLogs(blockNum int64, addrs []string) ([]*ethLog, error) {
	return s.source.GetLogs(logFilter{
		FromBlock: blockNum,
		ToBlock:   blockNum,
		Addresses: addrs,
	})
}

// parseTopic decodes a 0x-hex 32-byte topic string.
func parseTopic(s string) ([32]byte, error) {
	var out [32]byte
	h := strings.TrimPrefix(s, "0x")
	b, err := hex.DecodeString(h)
	if err != nil {
		return out, err
	}
	if len(b) != 32 {
		return out, fmt.Errorf("topic is %d bytes, want 32", len(b))
	}
	copy(out[:], b)
	return out, nil
}

// txReceipt pairs a transaction hash with its fetched receipt (for FanOut).
type txReceipt struct {
	hash    string
	receipt *ethReceipt
}

// collectModeReceipts fetches each of the block's transaction receipts (bounded
// parallel) and returns the logs emitted by the subscribed contract addresses.
func (s *Service) collectModeReceipts(blockNum int64, addrs []string) ([]*ethLog, error) {
	if s.blockTxHashes == nil {
		return nil, fmt.Errorf("eventlog: receipts mode requires blockTxHashes")
	}
	hashes, err := s.blockTxHashes(blockNum)
	if err != nil {
		return nil, err
	}
	if len(hashes) == 0 {
		return nil, nil
	}

	wanted := make(map[string]bool, len(addrs))
	for _, a := range addrs {
		wanted[strings.ToLower(a)] = true
	}

	items := make([]txReceipt, len(hashes))
	for i, h := range hashes {
		items[i] = txReceipt{hash: h}
	}

	results, err := flow.FanOut(context.Background(), items, s.cfg.ReceiptConcurrency,
		func(_ context.Context, tr txReceipt) (txReceipt, error) {
			r, err := s.source.GetTransactionReceipt(tr.hash)
			if err != nil {
				return tr, err
			}
			tr.receipt = r
			return tr, nil
		})
	if err != nil {
		return nil, err
	}

	var out []*ethLog
	for _, tr := range results {
		if tr.receipt == nil {
			continue
		}
		for _, lg := range tr.receipt.Logs {
			if wanted[strings.ToLower(lg.Address)] {
				out = append(out, lg)
			}
		}
	}
	return out, nil
}
