package eventlog

import (
	"bytes"
	"math/big"
	"sync"
	"testing"

	"github.com/ITProLabDev/ethbacknode/abi"
)

// fakeTransactionSource returns a canned transaction list for every block.
type fakeTransactionSource struct {
	txs          []*ethTransaction
	calls        int
	lastBlockNum int64
}

func (f *fakeTransactionSource) GetBlockTransactions(blockNum int64) ([]*ethTransaction, error) {
	f.calls++
	f.lastBlockNum = blockNum
	return f.txs, nil
}

// fakeTransactionDecoder returns a fixed method+inputs for a given contract
// address, regardless of calldata -- the decoding logic itself is abi's
// responsibility and is tested there (abi/decodecall_test.go).
type fakeTransactionDecoder struct {
	method string
	inputs []abi.DecodedValue
}

func (d *fakeTransactionDecoder) DecodeCall(contractAddress string, data []byte) (string, []abi.DecodedValue, error) {
	return d.method, d.inputs, nil
}

func TestService_OnBlock_DeliversContractTransactionForWholeContractScope(t *testing.T) {
	addr := "0xabc0000000000000000000000000000000000001"
	txSrc := &fakeTransactionSource{txs: []*ethTransaction{
		{Hash: "0xtx1", From: "0xfrom", To: addr, Value: big.NewInt(5), Gas: 100000, Input: []byte{0xa9, 0x05, 0x9c, 0xbb}},
	}}
	logSrc := &fakeLogSource{receipts: map[string]*ethReceipt{
		"0xtx1": {Status: true, GasUsed: 51000},
	}}
	dec := &fakeTransactionDecoder{method: "transfer", inputs: []abi.DecodedValue{{Name: "to", Type: "address"}}}

	var mu sync.Mutex
	var got []*ContractTransaction
	sink := func(ct *ContractTransaction) { mu.Lock(); got = append(got, ct); mu.Unlock() }

	svc := New(
		WithLogSource(logSrc),
		WithDecoder(&fakeDecoder{}),
		WithTransactionSource(txSrc),
		WithTransactionDecoder(dec),
		WithTransactionSink(sink),
		WithSink(func(*ContractEvent) {}),
		WithConfig(DefaultConfig()),
	)
	svc.Subscribe(&Subscription{ServiceID: "s1", ContractAddress: addr, Scope: ScopeWholeContract})

	svc.OnBlock(42, "0xblock")

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 {
		t.Fatalf("delivered %d contractTransactions, want 1", len(got))
	}
	ct := got[0]
	if ct.ServiceID != "s1" || ct.Contract != addr || ct.TransactionHash != "0xtx1" || ct.BlockNumber != 42 {
		t.Fatalf("context wrong: %+v", ct)
	}
	if ct.From != "0xfrom" || ct.To != addr || ct.Value.Cmp(big.NewInt(5)) != 0 || ct.Gas != 100000 {
		t.Fatalf("transaction fields wrong: %+v", ct)
	}
	if !ct.Success || ct.GasUsed != 51000 {
		t.Fatalf("receipt fields wrong: %+v", ct)
	}
	if !bytes.Equal(ct.Data, []byte{0xa9, 0x05, 0x9c, 0xbb}) {
		t.Fatalf("Data = %x, want the raw calldata (always included, decoded or not)", ct.Data)
	}
	if ct.Method != "transfer" || len(ct.Inputs) != 1 {
		t.Fatalf("decoded call wrong: %+v", ct)
	}
	if txSrc.calls != 1 || txSrc.lastBlockNum != 42 {
		t.Fatalf("GetBlockTransactions called %d times for block %d, want 1 call for block 42", txSrc.calls, txSrc.lastBlockNum)
	}
}

// A transaction whose selector matches no registered method still carries its
// raw calldata, so a subscriber can decode it on its own (e.g. against an ABI
// this registry does not have).
func TestService_OnBlock_ContractTransactionCarriesRawDataForAnUnknownSelector(t *testing.T) {
	addr := "0xabc0000000000000000000000000000000000001"
	calldata := []byte{0xde, 0xad, 0xbe, 0xef, 0x01, 0x02}
	txSrc := &fakeTransactionSource{txs: []*ethTransaction{
		{Hash: "0xtx1", To: addr, Value: big.NewInt(0), Input: calldata},
	}}
	logSrc := &fakeLogSource{receipts: map[string]*ethReceipt{"0xtx1": {Status: true}}}
	// Simulates abi.DecodeCall's real behavior for an unknown selector: the
	// hex selector as method, no decoded inputs, no error.
	dec := &fakeTransactionDecoder{method: "0xdeadbeef", inputs: nil}

	var got []*ContractTransaction
	sink := func(ct *ContractTransaction) { got = append(got, ct) }

	svc := New(
		WithLogSource(logSrc),
		WithDecoder(&fakeDecoder{}),
		WithTransactionSource(txSrc),
		WithTransactionDecoder(dec),
		WithTransactionSink(sink),
		WithSink(func(*ContractEvent) {}),
		WithConfig(DefaultConfig()),
	)
	svc.Subscribe(&Subscription{ServiceID: "s1", ContractAddress: addr, Scope: ScopeWholeContract})

	svc.OnBlock(1, "0xblock")

	if len(got) != 1 {
		t.Fatalf("delivered %d contractTransactions, want 1", len(got))
	}
	ct := got[0]
	if ct.Method != "0xdeadbeef" || ct.Inputs != nil {
		t.Fatalf("expected undecoded method+no inputs, got: %+v", ct)
	}
	if !bytes.Equal(ct.Data, calldata) {
		t.Fatalf("Data = %x, want %x (the raw calldata, since the selector was not decodable)", ct.Data, calldata)
	}
}

func TestService_OnBlock_DoesNotDeliverContractTransactionForManagedOnlyScope(t *testing.T) {
	addr := "0xabc0000000000000000000000000000000000001"
	txSrc := &fakeTransactionSource{txs: []*ethTransaction{
		{Hash: "0xtx1", From: "0xfrom", To: addr, Value: big.NewInt(0), Gas: 21000, Input: nil},
	}}
	logSrc := &fakeLogSource{receipts: map[string]*ethReceipt{"0xtx1": {Status: true}}}

	var got []*ContractTransaction
	sink := func(ct *ContractTransaction) { got = append(got, ct) }

	svc := New(
		WithLogSource(logSrc),
		WithDecoder(&fakeDecoder{}),
		WithTransactionSource(txSrc),
		WithTransactionDecoder(&fakeTransactionDecoder{}),
		WithTransactionSink(sink),
		WithSink(func(*ContractEvent) {}),
		WithConfig(DefaultConfig()),
	)
	// managed_only only -- contractTransaction is whole_contract-only in v1
	// (see todo/TASKS.md Milestone 8).
	svc.Subscribe(&Subscription{ServiceID: "s1", ContractAddress: addr, Scope: ScopeManagedOnly})

	svc.OnBlock(1, "0xblock")

	if len(got) != 0 {
		t.Fatalf("delivered %d contractTransactions to a managed_only-only subscription, want 0: %+v", len(got), got)
	}
}

func TestService_OnBlock_ContractTransactionIgnoresUnsubscribedRecipients(t *testing.T) {
	subscribed := "0xabc0000000000000000000000000000000000001"
	other := "0xdef0000000000000000000000000000000000002"
	txSrc := &fakeTransactionSource{txs: []*ethTransaction{
		{Hash: "0xtx1", To: other, Value: big.NewInt(0)},
	}}

	var got []*ContractTransaction
	sink := func(ct *ContractTransaction) { got = append(got, ct) }

	svc := New(
		WithLogSource(&fakeLogSource{}),
		WithDecoder(&fakeDecoder{}),
		WithTransactionSource(txSrc),
		WithTransactionDecoder(&fakeTransactionDecoder{}),
		WithTransactionSink(sink),
		WithSink(func(*ContractEvent) {}),
		WithConfig(DefaultConfig()),
	)
	svc.Subscribe(&Subscription{ServiceID: "s1", ContractAddress: subscribed, Scope: ScopeWholeContract})

	svc.OnBlock(1, "0xblock")

	if len(got) != 0 {
		t.Fatalf("delivered %d contractTransactions for an unsubscribed recipient, want 0: %+v", len(got), got)
	}
}

func TestService_OnBlock_ContractTransactionDeliversToEachSubscribedService(t *testing.T) {
	addr := "0xabc0000000000000000000000000000000000001"
	txSrc := &fakeTransactionSource{txs: []*ethTransaction{
		{Hash: "0xtx1", To: addr, Value: big.NewInt(0)},
	}}
	logSrc := &fakeLogSource{receipts: map[string]*ethReceipt{"0xtx1": {Status: true}}}

	var mu sync.Mutex
	var ids []string
	sink := func(ct *ContractTransaction) { mu.Lock(); ids = append(ids, ct.ServiceID); mu.Unlock() }

	svc := New(
		WithLogSource(logSrc),
		WithDecoder(&fakeDecoder{}),
		WithTransactionSource(txSrc),
		WithTransactionDecoder(&fakeTransactionDecoder{}),
		WithTransactionSink(sink),
		WithSink(func(*ContractEvent) {}),
		WithConfig(DefaultConfig()),
	)
	svc.Subscribe(&Subscription{ServiceID: "svcA", ContractAddress: addr, Scope: ScopeWholeContract})
	svc.Subscribe(&Subscription{ServiceID: "svcB", ContractAddress: addr, Scope: ScopeWholeContract})

	svc.OnBlock(1, "0xblock")

	mu.Lock()
	defer mu.Unlock()
	if len(ids) != 2 {
		t.Fatalf("delivered %d times, want 2 (one per subscribed service): %v", len(ids), ids)
	}
	seen := map[string]bool{ids[0]: true, ids[1]: true}
	if !seen["svcA"] || !seen["svcB"] {
		t.Fatalf("serviceIDs=%v want svcA+svcB", ids)
	}
}

func TestService_OnBlock_NoTransactionSinkDoesNoTransactionWork(t *testing.T) {
	addr := "0xabc0000000000000000000000000000000000001"
	txSrc := &fakeTransactionSource{}

	svc := New(
		WithLogSource(&fakeLogSource{}),
		WithDecoder(&fakeDecoder{}),
		WithTransactionSource(txSrc),
		// No WithTransactionDecoder / WithTransactionSink: contractTransaction
		// delivery must stay off entirely when unconfigured.
		WithSink(func(*ContractEvent) {}),
		WithConfig(DefaultConfig()),
	)
	svc.Subscribe(&Subscription{ServiceID: "s1", ContractAddress: addr, Scope: ScopeWholeContract})

	svc.OnBlock(1, "0xblock")

	if txSrc.calls != 0 {
		t.Fatalf("GetBlockTransactions called %d times with no TransactionSink wired, want 0", txSrc.calls)
	}
}
