package ethclient

import (
	"encoding/json"
	"fmt"

	"github.com/ITProLabDev/ethbacknode/common/hexnum"
)

// Log is a single event log entry as returned by eth_getLogs /
// eth_getTransactionReceipt. Numeric fields arrive as 0x-hex strings and are
// decoded via the proxy-map idiom used across this package.
type Log struct {
	Address          string   // 20-byte contract address (0x...)
	Topics           []string // 0x-hex 32-byte topics; Topics[0] is the event signature
	Data             []byte   // non-indexed event data (ABI-encoded)
	BlockNumber      int64    // block containing the log
	TransactionHash  string   // 0x-hex tx hash
	TransactionIndex int64    // tx position within the block
	LogIndex         int64    // log position within the block
	Removed          bool     // true if the log was reverted by a chain reorg
}

// Topics32 converts the 0x-hex topics into the [32]byte form the ABI decoder
// (abi.DecodeLog) expects. It errors if any topic is not exactly 32 bytes.
func (l *Log) Topics32() ([][32]byte, error) {
	out := make([][32]byte, len(l.Topics))
	for i, t := range l.Topics {
		b, err := hexnum.ParseHexBytes(t)
		if err != nil {
			return nil, err
		}
		if len(b) != 32 {
			return nil, fmt.Errorf("topic %d is %d bytes, want 32", i, len(b))
		}
		copy(out[i][:], b)
	}
	return out, nil
}

// UnmarshalJSON decodes a log from geth's 0x-hex JSON representation.
func (l *Log) UnmarshalJSON(data []byte) error {
	proxy := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &proxy); err != nil {
		return err
	}
	if v, ok := proxy["address"]; ok {
		if err := json.Unmarshal(v, &l.Address); err != nil {
			return err
		}
	}
	if v, ok := proxy["topics"]; ok {
		if err := json.Unmarshal(v, &l.Topics); err != nil {
			return err
		}
	}
	if v, ok := proxy["data"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return err
		}
		if s != "" && s != "0x" {
			b, err := hexnum.ParseHexBytes(s)
			if err != nil {
				return err
			}
			l.Data = b
		}
	}
	if v, ok := proxy["blockNumber"]; ok {
		if err := unmarshalHexInt64(v, &l.BlockNumber); err != nil {
			return err
		}
	}
	if v, ok := proxy["transactionHash"]; ok {
		if err := json.Unmarshal(v, &l.TransactionHash); err != nil {
			return err
		}
	}
	if v, ok := proxy["transactionIndex"]; ok {
		if err := unmarshalHexInt64(v, &l.TransactionIndex); err != nil {
			return err
		}
	}
	if v, ok := proxy["logIndex"]; ok {
		if err := unmarshalHexInt64(v, &l.LogIndex); err != nil {
			return err
		}
	}
	if v, ok := proxy["removed"]; ok {
		if err := json.Unmarshal(v, &l.Removed); err != nil {
			return err
		}
	}
	return nil
}

// unmarshalHexInt64 decodes a JSON 0x-hex string into an int64. Empty / "0x"
// values decode to 0.
func unmarshalHexInt64(raw json.RawMessage, dst *int64) error {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return err
	}
	if s == "" || s == "0x" {
		*dst = 0
		return nil
	}
	n, err := hexnum.ParseHexInt64(s)
	if err != nil {
		return err
	}
	*dst = n
	return nil
}
