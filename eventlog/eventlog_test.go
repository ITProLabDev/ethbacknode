package eventlog

import (
	"sync"
	"testing"

	"github.com/ITProLabDev/ethbacknode/abi"
)

// fakeLogSource returns canned logs for GetLogs and canned receipts by hash.
type fakeLogSource struct {
	logs       []*ethLog
	lastFilter logFilter
}

func (f *fakeLogSource) GetLogs(filter logFilter) ([]*ethLog, error) {
	f.lastFilter = filter
	return f.logs, nil
}
func (f *fakeLogSource) GetTransactionReceipt(string) (*ethReceipt, error) { return nil, nil }

// fakeDecoder decodes a log into a fixed event keyed by the contract address.
type fakeDecoder struct{ events map[string]*abi.DecodedEvent }

func (d *fakeDecoder) DecodeLog(contract string, topics [][32]byte, data []byte) (*abi.DecodedEvent, error) {
	ev := d.events[contract]
	if ev == nil {
		return nil, errNoEvent
	}
	return ev, nil
}

func topicHex(last byte) string {
	b := make([]byte, 32)
	b[31] = last
	const hexdigits = "0123456789abcdef"
	out := make([]byte, 0, 66)
	out = append(out, '0', 'x')
	for _, by := range b {
		out = append(out, hexdigits[by>>4], hexdigits[by&0xf])
	}
	return string(out)
}

func TestService_OnBlock_GetLogs_DeliversWholeContract(t *testing.T) {
	addr := "0xabc0000000000000000000000000000000000001"
	src := &fakeLogSource{logs: []*ethLog{
		{Address: addr, Topics: []string{topicHex(0xaa)}, Data: nil, BlockNumber: 100, LogIndex: 0, TransactionHash: "0xtx1"},
	}}
	dec := &fakeDecoder{events: map[string]*abi.DecodedEvent{
		addr: {Name: "Transfer", Contract: addr, Inputs: nil},
	}}

	var mu sync.Mutex
	var got []*ContractEvent
	sink := func(ce *ContractEvent) { mu.Lock(); got = append(got, ce); mu.Unlock() }

	svc := New(
		WithLogSource(src),
		WithDecoder(dec),
		WithManaged(stubManaged{known: map[string]bool{}}),
		WithAddressCodec(stubCodec{}),
		WithSink(sink),
		WithConfig(DefaultConfig()),
	)
	svc.Subscribe(&Subscription{ServiceID: "s1", ContractAddress: addr, Scope: ScopeWholeContract})

	svc.OnBlock(100, "0xblock")

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 {
		t.Fatalf("delivered %d events want 1", len(got))
	}
	if got[0].Event.Name != "Transfer" || got[0].BlockNumber != 100 || got[0].TransactionHash != "0xtx1" {
		t.Fatalf("event context wrong: %+v", got[0])
	}
	if src.lastFilter.FromBlock != 100 || src.lastFilter.ToBlock != 100 {
		t.Fatalf("filter block bounds=%+v", src.lastFilter)
	}
	if len(src.lastFilter.Addresses) != 1 {
		t.Fatalf("filter addresses=%v want 1", src.lastFilter.Addresses)
	}
}

func TestService_OnBlock_NoSubscriptions_NoCalls(t *testing.T) {
	src := &fakeLogSource{}
	svc := New(WithLogSource(src), WithDecoder(&fakeDecoder{}), WithSink(func(*ContractEvent) {}), WithConfig(DefaultConfig()))
	svc.OnBlock(5, "0xb")
	if src.lastFilter.FromBlock != 0 {
		t.Fatal("with no subscriptions, GetLogs must not be called")
	}
}
