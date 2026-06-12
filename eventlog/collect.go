package eventlog

import (
	"encoding/hex"
	"fmt"
	"strings"
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

// collectModeReceipts is implemented in Task 5 (receipts mode). Stub so the
// package compiles after Task 4; replaced with the real parallel-fetch version.
func (s *Service) collectModeReceipts(blockNum int64, addrs []string) ([]*ethLog, error) {
	return nil, fmt.Errorf("eventlog: receipts mode not yet implemented")
}
