package eventlog

import (
	"bytes"
	"math/big"
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
		GetReceiptFn: func(txHash string) (*RawReceipt, error) {
			return &RawReceipt{
				Logs:    []RawLog{{Address: "0xdef", BlockNumber: 4, TransactionHash: txHash, LogIndex: 0}},
				Status:  true,
				GasUsed: 51000,
			}, nil
		},
	}
	r, err := src.GetTransactionReceipt("0xt9")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Logs) != 1 || r.Logs[0].Address != "0xdef" || r.Logs[0].TransactionHash != "0xt9" {
		t.Fatalf("receipt logs not converted: %+v", r)
	}
	if !r.Status || r.GasUsed != 51000 {
		t.Fatalf("receipt status/gasUsed not converted: %+v", r)
	}
}

func TestRawTransactionSource_GetBlockTransactions(t *testing.T) {
	src := RawTransactionSource{
		GetBlockTransactionsFn: func(blockNum int64) ([]RawTransaction, error) {
			return []RawTransaction{
				{Hash: "0xtx1", From: "0xfrom", To: "0xto", Value: big.NewInt(7), Gas: 21000, Input: []byte{0xde, 0xad}},
			}, nil
		},
	}
	txs, err := src.GetBlockTransactions(9)
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 1 {
		t.Fatalf("got %d transactions, want 1", len(txs))
	}
	tx := txs[0]
	if tx.Hash != "0xtx1" || tx.From != "0xfrom" || tx.To != "0xto" || tx.Gas != 21000 {
		t.Fatalf("transaction not converted: %+v", tx)
	}
	if tx.Value.Cmp(big.NewInt(7)) != 0 || !bytes.Equal(tx.Input, []byte{0xde, 0xad}) {
		t.Fatalf("value/input not converted: %+v", tx)
	}
}

func TestRawTransactionSource_NilFunc(t *testing.T) {
	var src RawTransactionSource
	if _, err := src.GetBlockTransactions(1); err == nil {
		t.Fatal("nil GetBlockTransactionsFn must error, not panic")
	}
}

func TestTransactionDecoderAdapter_DelegatesToManager(t *testing.T) {
	called := false
	adapter := NewTransactionDecoder(func(contract string, data []byte) (string, []abi.DecodedValue, error) {
		called = true
		return "transfer", []abi.DecodedValue{{Name: "to", Type: "address"}}, nil
	})
	method, inputs, err := adapter.DecodeCall("0xabc", []byte{0xa9, 0x05, 0x9c, 0xbb})
	if err != nil || !called || method != "transfer" || len(inputs) != 1 {
		t.Fatalf("transaction decoder adapter failed: method=%v inputs=%v err=%v called=%v", method, inputs, err, called)
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
