package eventlog

import "testing"

func TestParseMode(t *testing.T) {
	cases := map[string]Mode{
		"getlogs":  ModeGetLogs,
		"GetLogs":  ModeGetLogs,
		"receipts": ModeReceipts,
		"RECEIPTS": ModeReceipts,
		"":         ModeGetLogs, // empty defaults to getLogs
	}
	for in, want := range cases {
		got, err := ParseMode(in)
		if err != nil || got != want {
			t.Fatalf("ParseMode(%q)=%v,%v want %v", in, got, err, want)
		}
	}
	if _, err := ParseMode("carrier-pigeon"); err == nil {
		t.Fatal("unknown mode must error")
	}
}

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	if c.Mode != ModeGetLogs {
		t.Fatalf("default mode=%v want ModeGetLogs", c.Mode)
	}
	if c.ReceiptConcurrency < 1 {
		t.Fatalf("default receipt concurrency=%d must be >=1", c.ReceiptConcurrency)
	}
}
