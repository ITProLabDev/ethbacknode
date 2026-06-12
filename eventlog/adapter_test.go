package eventlog

import (
	"testing"

	"github.com/ITProLabDev/ethbacknode/abi"
)

func TestDecoderAdapter_DelegatesToManager(t *testing.T) {
	called := false
	adapter := NewDecoder(func(contract string, topics [][32]byte, data []byte) (*abi.DecodedEvent, error) {
		called = true
		return &abi.DecodedEvent{Name: "E", Contract: contract}, nil
	})
	ev, err := adapter.DecodeLog("0xabc", [][32]byte{{0x01}}, nil)
	if err != nil || !called || ev.Name != "E" {
		t.Fatalf("decoder adapter failed: ev=%v err=%v called=%v", ev, err, called)
	}
}

func TestRawLogSource_GetLogs(t *testing.T) {
	var gotFrom, gotTo int64
	var gotAddrs []string
	src := RawLogSource{
		GetLogsFn: func(fromBlock, toBlock int64, addresses, topics []string) ([]RawLog, error) {
			gotFrom, gotTo, gotAddrs = fromBlock, toBlock, addresses
			return []RawLog{
				{Address: "0xabc", Topics: []string{"0x01"}, Data: []byte{0x05}, BlockNumber: 9, TransactionHash: "0xtx", LogIndex: 3},
			}, nil
		},
	}
	logs, err := src.GetLogs(logFilter{FromBlock: 9, ToBlock: 9, Addresses: []string{"0xabc"}})
	if err != nil {
		t.Fatal(err)
	}
	if gotFrom != 9 || gotTo != 9 || len(gotAddrs) != 1 {
		t.Fatalf("filter not passed through: from=%d to=%d addrs=%v", gotFrom, gotTo, gotAddrs)
	}
	if len(logs) != 1 || logs[0].Address != "0xabc" || logs[0].LogIndex != 3 || len(logs[0].Data) != 1 {
		t.Fatalf("log not converted: %+v", logs)
	}
}

func TestRawLogSource_GetTransactionReceipt(t *testing.T) {
	src := RawLogSource{
		GetReceiptFn: func(txHash string) ([]RawLog, error) {
			return []RawLog{{Address: "0xdef", BlockNumber: 4, TransactionHash: txHash, LogIndex: 0}}, nil
		},
	}
	r, err := src.GetTransactionReceipt("0xt9")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Logs) != 1 || r.Logs[0].Address != "0xdef" || r.Logs[0].TransactionHash != "0xt9" {
		t.Fatalf("receipt not converted: %+v", r)
	}
}

func TestRawLogSource_NilFuncs(t *testing.T) {
	var src RawLogSource // both func fields nil
	if _, err := src.GetLogs(logFilter{}); err == nil {
		t.Fatal("nil GetLogsFn must error, not panic")
	}
	if _, err := src.GetTransactionReceipt("0xt"); err == nil {
		t.Fatal("nil GetReceiptFn must error, not panic")
	}
}
