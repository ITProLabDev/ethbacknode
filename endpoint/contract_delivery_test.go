package endpoint

import (
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
