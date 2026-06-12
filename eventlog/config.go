package eventlog

import (
	"fmt"
	"strings"
)

// Mode selects how the service collects a block's logs.
type Mode int

const (
	// ModeGetLogs fetches a block's logs with a single eth_getLogs call,
	// filtered by the subscribed contract addresses. Independent of tx count.
	ModeGetLogs Mode = iota
	// ModeReceipts fetches each relevant transaction's receipt and reads its
	// logs. Precise per-tx attribution at the cost of N calls per block.
	ModeReceipts
)

// ParseMode parses a mode name (case-insensitive). Empty defaults to getLogs.
func ParseMode(s string) (Mode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "getlogs":
		return ModeGetLogs, nil
	case "receipts":
		return ModeReceipts, nil
	default:
		return 0, fmt.Errorf("unknown eventlog mode %q (want getLogs | receipts)", s)
	}
}

// Config configures the eventlog service.
type Config struct {
	// Mode selects the per-block log collection strategy.
	Mode Mode
	// ReceiptConcurrency bounds parallel receipt fetches in ModeReceipts.
	ReceiptConcurrency int
}

// DefaultConfig returns the default configuration: getLogs mode, 8-way receipt
// concurrency.
func DefaultConfig() Config {
	return Config{Mode: ModeGetLogs, ReceiptConcurrency: 8}
}
