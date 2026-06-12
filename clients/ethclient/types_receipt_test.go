package ethclient

import (
	"encoding/json"
	"testing"
)

const sampleLogJSON = `{
  "address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
  "topics": [
    "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
    "0x0000000000000000000000001111111111111111111111111111111111111111",
    "0x0000000000000000000000002222222222222222222222222222222222222222"
  ],
  "data": "0x00000000000000000000000000000000000000000000000000000000000003e8",
  "blockNumber": "0x10d4f",
  "transactionHash": "0xabc0000000000000000000000000000000000000000000000000000000000001",
  "transactionIndex": "0x2",
  "logIndex": "0x5",
  "removed": false
}`

func TestLog_UnmarshalJSON(t *testing.T) {
	var lg Log
	if err := json.Unmarshal([]byte(sampleLogJSON), &lg); err != nil {
		t.Fatal(err)
	}
	if lg.Address != "0xdac17f958d2ee523a2206206994597c13d831ec7" {
		t.Fatalf("address=%q", lg.Address)
	}
	if len(lg.Topics) != 3 {
		t.Fatalf("topics=%d want 3", len(lg.Topics))
	}
	if lg.BlockNumber != 0x10d4f {
		t.Fatalf("blockNumber=%d want %d", lg.BlockNumber, 0x10d4f)
	}
	if lg.TransactionIndex != 2 || lg.LogIndex != 5 {
		t.Fatalf("txIndex=%d logIndex=%d", lg.TransactionIndex, lg.LogIndex)
	}
	if lg.Removed {
		t.Fatal("removed should be false")
	}
	if len(lg.Data) != 32 {
		t.Fatalf("data len=%d want 32", len(lg.Data))
	}
}

func TestLog_Topics32(t *testing.T) {
	var lg Log
	if err := json.Unmarshal([]byte(sampleLogJSON), &lg); err != nil {
		t.Fatal(err)
	}
	t32, err := lg.Topics32()
	if err != nil {
		t.Fatal(err)
	}
	if len(t32) != 3 {
		t.Fatalf("t32 len=%d", len(t32))
	}
	if t32[0][0] != 0xdd || t32[0][31] != 0xef {
		t.Fatalf("topic0=%x", t32[0])
	}
	// All topics must be decoded, not just topic0: topic1/topic2 are
	// left-padded addresses ending in 0x11.../0x22...
	if t32[1][31] != 0x11 || t32[2][31] != 0x22 {
		t.Fatalf("topic1=%x topic2=%x", t32[1], t32[2])
	}
}

func TestLog_Topics32_RejectsBadLength(t *testing.T) {
	lg := Log{Topics: []string{"0x1234"}} // 2 bytes, not 32
	if _, err := lg.Topics32(); err == nil {
		t.Fatal("a non-32-byte topic must error")
	}
}

const sampleReceiptJSON = `{
  "transactionHash": "0xabc0000000000000000000000000000000000000000000000000000000000001",
  "transactionIndex": "0x2",
  "blockHash": "0xbbb0000000000000000000000000000000000000000000000000000000000002",
  "blockNumber": "0x10d4f",
  "from": "0x1111111111111111111111111111111111111111",
  "to": "0xdac17f958d2ee523a2206206994597c13d831ec7",
  "cumulativeGasUsed": "0x5208",
  "gasUsed": "0x5208",
  "contractAddress": null,
  "status": "0x1",
  "logs": [
    {
      "address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
      "topics": ["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"],
      "data": "0x00000000000000000000000000000000000000000000000000000000000003e8",
      "blockNumber": "0x10d4f",
      "transactionHash": "0xabc0000000000000000000000000000000000000000000000000000000000001",
      "transactionIndex": "0x2",
      "logIndex": "0x0",
      "removed": false
    }
  ]
}`

func TestReceipt_UnmarshalJSON(t *testing.T) {
	var r Receipt
	if err := json.Unmarshal([]byte(sampleReceiptJSON), &r); err != nil {
		t.Fatal(err)
	}
	if r.Status != 1 {
		t.Fatalf("status=%d want 1", r.Status)
	}
	if r.BlockNumber != 0x10d4f {
		t.Fatalf("blockNumber=%d", r.BlockNumber)
	}
	if r.GasUsed != 0x5208 {
		t.Fatalf("gasUsed=%d", r.GasUsed)
	}
	if r.From != "0x1111111111111111111111111111111111111111" {
		t.Fatalf("from=%q", r.From)
	}
	if len(r.Logs) != 1 {
		t.Fatalf("logs=%d want 1", len(r.Logs))
	}
	if len(r.Logs[0].Topics) != 1 {
		t.Fatalf("log topics=%d", len(r.Logs[0].Topics))
	}
}

func TestReceipt_Success(t *testing.T) {
	var r Receipt
	if err := json.Unmarshal([]byte(sampleReceiptJSON), &r); err != nil {
		t.Fatal(err)
	}
	if !r.Success() {
		t.Fatal("status 0x1 should be Success")
	}
	r.Status = 0
	if r.Success() {
		t.Fatal("status 0x0 should not be Success")
	}
}
