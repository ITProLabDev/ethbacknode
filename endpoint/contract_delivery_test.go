package endpoint

import (
	"math/big"
	"testing"

	"github.com/ITProLabDev/ethbacknode/abi"
	"github.com/ITProLabDev/ethbacknode/eventlog"
)

// fakeNotifier captures NotifySubscriber calls.
type fakeNotifier struct {
	serviceID int64
	subject   string
	payload   interface{}
	calls     int
}

func (f *fakeNotifier) NotifyContractEvent(serviceID int64, subject string, payload interface{}) {
	f.serviceID = serviceID
	f.subject = subject
	f.payload = payload
	f.calls++
}

func TestContractEventSink_RoutesToSubscriber(t *testing.T) {
	notifier := &fakeNotifier{}
	sink := NewContractEventSink(notifier)

	sink(&eventlog.ContractEvent{
		ServiceID:       "42",
		Event:           &abi.DecodedEvent{Name: "Transfer", Contract: "0xabc"},
		BlockNumber:     100,
		TransactionHash: "0xtx",
		LogIndex:        3,
	})

	if notifier.calls != 1 {
		t.Fatalf("notify calls=%d want 1", notifier.calls)
	}
	if notifier.serviceID != 42 {
		t.Fatalf("serviceID=%d want 42", notifier.serviceID)
	}
	if notifier.subject != "contractEvent" {
		t.Fatalf("subject=%q want contractEvent", notifier.subject)
	}
	p, ok := notifier.payload.(*contractEventPayload)
	if !ok {
		t.Fatalf("payload type=%T", notifier.payload)
	}
	if p.Event != "Transfer" || p.Contract != "0xabc" || p.BlockNum != 100 || p.TxHash != "0xtx" {
		t.Fatalf("payload=%+v", p)
	}
}

func TestContractEventSink_SkipsBadServiceID(t *testing.T) {
	notifier := &fakeNotifier{}
	sink := NewContractEventSink(notifier)
	// Non-numeric ServiceID must be skipped (logged), not delivered.
	sink(&eventlog.ContractEvent{ServiceID: "not-a-number", Event: &abi.DecodedEvent{Name: "X"}})
	if notifier.calls != 0 {
		t.Fatalf("bad serviceID must not deliver, calls=%d", notifier.calls)
	}
}

func TestContractTransactionSink_RoutesToSubscriber(t *testing.T) {
	notifier := &fakeNotifier{}
	sink := NewContractTransactionSink(notifier)

	sink(&eventlog.ContractTransaction{
		ServiceID:       "42",
		Contract:        "0xabc",
		BlockNumber:     100,
		TransactionHash: "0xtx",
		From:            "0xfrom",
		To:              "0xabc",
		Value:           big.NewInt(1000000000000000000),
		Gas:             100000,
		GasUsed:         51000,
		Success:         true,
		Method:          "transfer",
		Inputs:          []abi.DecodedValue{{Name: "to", Type: "address"}},
		Data:            []byte{0xa9, 0x05, 0x9c, 0xbb},
	})

	if notifier.calls != 1 {
		t.Fatalf("notify calls=%d want 1", notifier.calls)
	}
	if notifier.serviceID != 42 {
		t.Fatalf("serviceID=%d want 42", notifier.serviceID)
	}
	if notifier.subject != "contractTransaction" {
		t.Fatalf("subject=%q want contractTransaction", notifier.subject)
	}
	p, ok := notifier.payload.(*contractTransactionPayload)
	if !ok {
		t.Fatalf("payload type=%T", notifier.payload)
	}
	if p.Contract != "0xabc" || p.BlockNum != 100 || p.TxHash != "0xtx" || p.From != "0xfrom" || p.To != "0xabc" {
		t.Fatalf("payload context wrong: %+v", p)
	}
	if p.Value != "1000000000000000000" {
		t.Fatalf("value = %q, want a decimal string (JS-safe), not a raw number", p.Value)
	}
	if p.Gas != 100000 || p.GasUsed != 51000 || !p.Success {
		t.Fatalf("gas/status wrong: %+v", p)
	}
	if p.Method != "transfer" || len(p.Inputs) != 1 {
		t.Fatalf("decoded call wrong: %+v", p)
	}
	if p.Data != "0xa9059cbb" {
		t.Fatalf("data = %q, want the raw calldata as 0x-hex, always included", p.Data)
	}
}

func TestContractTransactionSink_SkipsBadServiceID(t *testing.T) {
	notifier := &fakeNotifier{}
	sink := NewContractTransactionSink(notifier)
	sink(&eventlog.ContractTransaction{ServiceID: "not-a-number", Contract: "0xabc"})
	if notifier.calls != 0 {
		t.Fatalf("bad serviceID must not deliver, calls=%d", notifier.calls)
	}
}

func TestContractTransactionSink_NilValueBecomesZero(t *testing.T) {
	notifier := &fakeNotifier{}
	sink := NewContractTransactionSink(notifier)
	sink(&eventlog.ContractTransaction{ServiceID: "1", Contract: "0xabc", Value: nil})
	p := notifier.payload.(*contractTransactionPayload)
	if p.Value != "0" {
		t.Fatalf("value = %q, want \"0\" for a nil Value", p.Value)
	}
}
