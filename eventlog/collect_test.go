package eventlog

import (
	"sort"
	"testing"
)

// recProvider serves receipts by tx hash and a block's tx-hash list.
type recProvider struct {
	receipts map[string]*ethReceipt
}

func (r *recProvider) GetLogs(logFilter) ([]*ethLog, error) { return nil, nil }
func (r *recProvider) GetTransactionReceipt(txHash string) (*ethReceipt, error) {
	return r.receipts[txHash], nil
}

func TestCollectModeReceipts_GathersAllLogs(t *testing.T) {
	addr := "0xabc0000000000000000000000000000000000001"
	prov := &recProvider{
		receipts: map[string]*ethReceipt{
			"0xt1": {Logs: []*ethLog{{Address: addr, BlockNumber: 7, LogIndex: 0, TransactionHash: "0xt1"}}},
			"0xt2": {Logs: []*ethLog{
				{Address: addr, BlockNumber: 7, LogIndex: 1, TransactionHash: "0xt2"},
				{Address: addr, BlockNumber: 7, LogIndex: 2, TransactionHash: "0xt2"},
			}},
		},
	}
	svc := New(
		WithLogSource(prov),
		WithConfig(Config{Mode: ModeReceipts, ReceiptConcurrency: 4}),
	)
	svc.blockTxHashes = func(blockNum int64) ([]string, error) {
		return []string{"0xt1", "0xt2"}, nil
	}

	logs, err := svc.collectModeReceipts(7, []string{addr})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 3 {
		t.Fatalf("collected %d logs want 3", len(logs))
	}
	idx := []int64{logs[0].LogIndex, logs[1].LogIndex, logs[2].LogIndex}
	sort.Slice(idx, func(i, j int) bool { return idx[i] < idx[j] })
	if idx[0] != 0 || idx[1] != 1 || idx[2] != 2 {
		t.Fatalf("log indices=%v", idx)
	}
}

func TestCollectModeReceipts_FiltersByAddress(t *testing.T) {
	want := "0xabc0000000000000000000000000000000000001"
	other := "0x9990000000000000000000000000000000000009"
	prov := &recProvider{
		receipts: map[string]*ethReceipt{
			"0xt1": {Logs: []*ethLog{
				{Address: want, BlockNumber: 7, TransactionHash: "0xt1"},
				{Address: other, BlockNumber: 7, TransactionHash: "0xt1"},
			}},
		},
	}
	svc := New(WithLogSource(prov), WithConfig(Config{Mode: ModeReceipts, ReceiptConcurrency: 2}))
	svc.blockTxHashes = func(int64) ([]string, error) { return []string{"0xt1"}, nil }

	logs, err := svc.collectModeReceipts(7, []string{want})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].Address != want {
		t.Fatalf("address filter failed: %+v", logs)
	}
}

func TestCollectModeReceipts_NoSeam(t *testing.T) {
	// Without a blockTxHashes seam wired, receipts mode must error (not panic).
	svc := New(WithLogSource(&recProvider{}), WithConfig(Config{Mode: ModeReceipts, ReceiptConcurrency: 2}))
	if _, err := svc.collectModeReceipts(1, []string{"0xabc"}); err == nil {
		t.Fatal("missing blockTxHashes seam must error")
	}
}
