# M6 — Delivery & API (contractEvent + contract/subscription RPC) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver decoded contract events to subscribers as a `contractEvent` JSON-RPC callback, and add RPC methods to register a contract+ABI, list registered contracts, subscribe/unsubscribe to a contract's events with a `scope` (whole_contract | managed_only), and call a view method (`contractCall`) — persisting registrations and subscriptions in existing storage.

**Architecture:** M5's eventlog `Sink` is replaced (in main.go) by a delivery function that routes each `ContractEvent` to the owning subscriber via the existing `subscriptions.Manager` HTTP-callback mechanism. The eventlog `ContractEvent`/`Subscription` carry the `ServiceID` so the sink knows the recipient. New RPC processors (registered through the existing `RegisterProcessor` table — already an extensible map, no dispatch-core change) cover contract registration (via M3 `NewContractFromABI` + abi manager), event subscription (via eventlog `Subscribe`), listing, and `contractCall` (via M4 `CallMethod`). Registrations/subscriptions persist as JSON in the existing module storage.

**Tech Stack:** Go 1.24, the existing `endpoint` JSON-RPC server (`RegisterProcessor`/`RegisterSecuredProcessor`, `RpcRequest`/`RpcResponse`), `subscriptions.Manager` (HTTP callback delivery), M5 `eventlog` (`Service`, `Subscription`, `Scope`, `ContractEvent`), M3 `abi` (`NewContractFromABI`, `SmartContractsManager`), M4 `ethclient.CallMethod`, the `storage` package, standard `testing`.

---

## Background & Constraints

**Hard rule:** self-built stack only — NO go-ethereum / external ABI/RPC library. `abi/` stays pure.

**The delivery gap M6 must close (verified against M5 code):**
- M5's `eventlog.Service.decodeAndDeliver` fires the `Sink` once per *matching subscription*, but `eventlog.ContractEvent` does NOT carry which subscriber matched. M6 adds `ServiceID` to `ContractEvent` and sets it from the matched `Subscription` so the sink can route delivery. (This is a small M5 change, justified: the sink contract is M5's seam and M6 is its first real consumer.)
- The sink is called concurrently (documented in M5). The delivery sink uses `subscriptions.Manager.NotifySubscriber`, which is already concurrency-safe (RLock).

**Integration points (verified):**
- `endpoint.BackRpc` holds deps (`addressPool`, `chainClient types.ChainClient`, `subscriptions *subscriptions.Manager`, ...) and a `rpcProcessors map[RpcMethod]RpcProcessor`. `RegisterProcessor(method, fn)` / `RegisterSecuredProcessor(method, fn)` populate the map; `InitProcessors()` wires built-ins. The map IS the extensible dispatch table — adding methods needs no core change.
- `NewBackRpc(addressPool, chainClient, subscriptions, watchdog, txCache, ...options)` constructs it. M6 adds the abi manager + eventlog service via `BackRpcOption`s (so the signature stays stable).
- RPC processor signature: `func(ctx RequestContext, request RpcRequest, response RpcResponse)`. `request.ParseParams(&struct)`, `response.SetResult(v)`, `response.SetError(code, msg)`. See `methods_subscribers.go` / `methods_balance.go` for the pattern.
- `subscriptions.Manager.NotifySubscriber(serviceId ServiceId, subject string, data Signer)` delivers to a subscriber's HTTP callback. `Signer` is `interface{ Sign(apiKey string) }`. `subscriptions.ServiceId` is an `int`.
- `(*Subscription).sendNotification(method string, message interface{}, debug bool)` is the lower-level path (message is any JSON-serializable value).
- M5 `eventlog.Service`: `Subscribe(*Subscription)`, `Subscribe`d events flow through `OnBlock`→sink. `eventlog.Subscription{ServiceID string, ContractAddress string, Scope Scope}`. `ParseScope(string) (Scope, error)`.
- M3 `abi`: `NewContractFromABI(name, symbol, address string, rawABI []byte) (*SmartContractInfo, error)`, `(*SmartContractsManager).Add(*SmartContractInfo)`, `.GetSmartContractList() map[string]string`, `.GetSmartContractByAddress(addr)`.
- M4 `ethclient`: `(*Client).CallMethod(contract, method string, args ...any) ([]abi.DecodedValue, error)`. NOTE: `BackRpc.chainClient` is typed `types.ChainClient` (interface) which does NOT include `CallMethod`. M6 adds `CallMethod` to a NEW narrow interface the contractCall processor uses, wired from the concrete client (mirroring M5's LogSource decoupling). See Task 5.

**Decisions locked for M6:**
- `ServiceID` on eventlog `ContractEvent`/`Subscription` is already a `string` (M5). The subscriptions `Manager` uses `ServiceId int`. The contractEvent subscriber is identified by the same numeric serviceId as other subscriptions; M6 stores it as the string form in eventlog and converts at the delivery boundary.
- Contract registration and event subscriptions are persisted as JSON via the existing module storage (`data/abi/` already holds `known_contracts.json` through the abi manager's own Save; eventlog subscriptions get a new `data/eventlog/subscriptions.json`). The abi manager already persists contracts on `Add` — so registering a contract is durable for free. Eventlog subscriptions need their own small persistence (Task 6).
- `contractCall` is a READ-only RPC (eth_call); no signing. Consistent with the "read + decode + encode call-data only" milestone scope.
- Extensibility: new event types are delivered through the same `sendNotification(subject, payload)` path — adding a type is a new subject string + payload struct, no dispatch switch. New RPC methods are new `RegisterProcessor` calls. This satisfies the extensibility requirement without a registry rewrite.

---

## File Structure

- **Modify `eventlog/event.go`** — add `ServiceID` to `ContractEvent`.
- **Modify `eventlog/eventlog.go`** — set `ContractEvent.ServiceID = sub.ServiceID` in `decodeAndDeliver`.
- **Create `endpoint/methods_contracts.go`** — RPC processors: `contractRegister`, `contractList`, `contractSubscribe`, `contractUnsubscribe`, `contractCall`.
- **Modify `endpoint/rpc_init.go`** — register the new methods in `InitProcessors`.
- **Modify `endpoint/rpc_handler.go`** — add `abiManager`, `eventLog`, `contractCaller` deps + `BackRpcOption`s.
- **Create `endpoint/contract_delivery.go`** — the eventlog→subscriptions delivery sink + its `contractEventPayload`.
- **Create `eventlog/persist.go`** — load/save eventlog subscriptions as JSON via `storage.BinStorage`.
- **Modify `main.go`** — replace the logging sink with the delivery sink; wire abi manager + eventlog + caller into `NewBackRpc`; load persisted subscriptions.
- Test files alongside each.

No changes to `abi/` internals, `watchdog/`, or the codec.

---

### Task 1: Thread ServiceID through ContractEvent

**Files:**
- Modify: `eventlog/event.go`
- Modify: `eventlog/eventlog.go`
- Test: `eventlog/eventlog_test.go` (append)

- [ ] **Step 1: Write the failing test** — append to `eventlog/eventlog_test.go`:

```go
func TestService_OnBlock_SetsServiceIDPerSubscription(t *testing.T) {
	addr := "0xabc0000000000000000000000000000000000001"
	src := &fakeLogSource{logs: []*ethLog{
		{Address: addr, Topics: []string{topicHex(0xaa)}, BlockNumber: 1, LogIndex: 0, TransactionHash: "0xtx"},
	}}
	dec := &fakeDecoder{events: map[string]*abi.DecodedEvent{
		addr: {Name: "E", Contract: addr},
	}}
	var mu sync.Mutex
	var ids []string
	sink := func(ce *ContractEvent) { mu.Lock(); ids = append(ids, ce.ServiceID); mu.Unlock() }

	svc := New(WithLogSource(src), WithDecoder(dec),
		WithManaged(stubManaged{known: map[string]bool{}}), WithAddressCodec(stubCodec{}),
		WithSink(sink), WithConfig(DefaultConfig()))
	svc.Subscribe(&Subscription{ServiceID: "svcA", ContractAddress: addr, Scope: ScopeWholeContract})
	svc.Subscribe(&Subscription{ServiceID: "svcB", ContractAddress: addr, Scope: ScopeWholeContract})

	svc.OnBlock(1, "0xb")

	mu.Lock()
	defer mu.Unlock()
	if len(ids) != 2 {
		t.Fatalf("delivered %d want 2", len(ids))
	}
	// Each subscriber must receive an event tagged with its own ServiceID.
	seen := map[string]bool{ids[0]: true, ids[1]: true}
	if !seen["svcA"] || !seen["svcB"] {
		t.Fatalf("serviceIDs=%v want svcA+svcB", ids)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./eventlog/ -run TestService_OnBlock_SetsServiceIDPerSubscription -v`
Expected: FAIL — `ce.ServiceID` undefined (field missing).

- [ ] **Step 3: Add the field** — in `eventlog/event.go`, add `ServiceID` to `ContractEvent`:

```go
// ContractEvent is a decoded contract event plus the block/tx context needed
// for delivery. It is the unit handed to a Sink.
type ContractEvent struct {
	Event            *abi.DecodedEvent // decoded event (name, contract, inputs)
	ServiceID        string            // the matched subscription's subscriber ID
	BlockNumber      int64
	TransactionHash  string
	TransactionIndex int64
	LogIndex         int64
	// Removed is true if the source log was reverted by a chain reorg.
	Removed bool
}
```

- [ ] **Step 4: Set it in decodeAndDeliver** — in `eventlog/eventlog.go`, set `ServiceID` from the matched subscription inside the delivery loop. Change the `s.sink(&ContractEvent{...})` construction to include `ServiceID: sub.ServiceID`:

```go
	for _, sub := range subs {
		if eventMatchesScope(ev, sub.Scope, s.managed, s.codec) {
			s.sink(&ContractEvent{
				Event:            ev,
				ServiceID:        sub.ServiceID,
				BlockNumber:      lg.BlockNumber,
				TransactionHash:  lg.TransactionHash,
				TransactionIndex: lg.TransactionIndex,
				LogIndex:         lg.LogIndex,
				Removed:          lg.Removed,
			})
		}
	}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./eventlog/ -run TestService_OnBlock_SetsServiceIDPerSubscription -v` then `go test ./eventlog/ -count=1`
Expected: PASS; full eventlog package `ok` (no regressions).

- [ ] **Step 6: Commit**

```bash
git add eventlog/event.go eventlog/eventlog.go eventlog/eventlog_test.go
git commit -m "feat(eventlog): tag ContractEvent with matched subscription ServiceID"
```

---

### Task 2: Persist eventlog subscriptions (load/save JSON)

**Files:**
- Create: `eventlog/persist.go`
- Test: `eventlog/persist_test.go`

- [ ] **Step 1: Write the failing test** — create `eventlog/persist_test.go`:

```go
package eventlog

import (
	"sync"
	"testing"
)

// memStore is an in-memory storage.BinStorage for tests.
type memStore struct {
	mu     sync.Mutex
	data   []byte
	exists bool
}

func (s *memStore) IsExists() bool { return s.exists }
func (s *memStore) Save(raw []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = append(s.data[:0], raw...)
	s.exists = true
	return nil
}
func (s *memStore) Load() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]byte, len(s.data))
	copy(out, s.data)
	return out, nil
}

func TestService_SubscribePersists(t *testing.T) {
	st := &memStore{}
	svc := New(WithSubscriptionStorage(st))
	if err := svc.SubscribeAndSave(&Subscription{ServiceID: "s1", ContractAddress: "0xAA", Scope: ScopeWholeContract}); err != nil {
		t.Fatal(err)
	}
	if !st.exists {
		t.Fatal("subscription should have been persisted")
	}

	// A fresh service loading the same storage must see the subscription.
	svc2 := New(WithSubscriptionStorage(st))
	if err := svc2.LoadSubscriptions(); err != nil {
		t.Fatal(err)
	}
	subs := svc2.subs.forAddress("0xaa")
	if len(subs) != 1 || subs[0].ServiceID != "s1" || subs[0].Scope != ScopeWholeContract {
		t.Fatalf("loaded subs=%+v", subs)
	}
}

func TestService_LoadSubscriptions_EmptyStorage(t *testing.T) {
	st := &memStore{} // never written
	svc := New(WithSubscriptionStorage(st))
	if err := svc.LoadSubscriptions(); err != nil {
		t.Fatalf("loading empty storage must not error: %v", err)
	}
	if len(svc.subs.addresses()) != 0 {
		t.Fatal("empty storage should yield no subscriptions")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./eventlog/ -run 'TestService_SubscribePersists|TestService_LoadSubscriptions' -v`
Expected: FAIL — `WithSubscriptionStorage` / `SubscribeAndSave` / `LoadSubscriptions` undefined.

- [ ] **Step 3: Write minimal implementation** — create `eventlog/persist.go`:

```go
package eventlog

import (
	"encoding/json"

	"github.com/ITProLabDev/ethbacknode/storage"
)

// persistedSub is the JSON form of a subscription (Scope as its string name).
type persistedSub struct {
	ServiceID       string `json:"serviceId"`
	ContractAddress string `json:"contractAddress"`
	Scope           string `json:"scope"`
}

// WithSubscriptionStorage sets the storage backend used to persist event
// subscriptions.
func WithSubscriptionStorage(st storage.BinStorage) Option {
	return func(svc *Service) { svc.subStorage = st }
}

// SubscribeAndSave registers a subscription and persists the full set.
func (s *Service) SubscribeAndSave(sub *Subscription) error {
	s.subs.add(sub)
	return s.saveSubscriptions()
}

// saveSubscriptions writes all subscriptions to storage (no-op if no storage).
func (s *Service) saveSubscriptions() error {
	if s.subStorage == nil {
		return nil
	}
	all := s.subs.all()
	out := make([]persistedSub, len(all))
	for i, sub := range all {
		out[i] = persistedSub{
			ServiceID:       sub.ServiceID,
			ContractAddress: sub.ContractAddress,
			Scope:           scopeName(sub.Scope),
		}
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return s.subStorage.Save(data)
}

// LoadSubscriptions loads persisted subscriptions from storage into the set.
// A missing/empty store is not an error.
func (s *Service) LoadSubscriptions() error {
	if s.subStorage == nil || !s.subStorage.IsExists() {
		return nil
	}
	data, err := s.subStorage.Load()
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	var loaded []persistedSub
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}
	for _, p := range loaded {
		scope, err := ParseScope(p.Scope)
		if err != nil {
			return err
		}
		s.subs.add(&Subscription{ServiceID: p.ServiceID, ContractAddress: p.ContractAddress, Scope: scope})
	}
	return nil
}
```

Add the `subStorage` field to the `Service` struct in `eventlog/eventlog.go` (after `blockTxHashes`):

```go
	// subStorage persists event subscriptions (optional).
	subStorage storage.BinStorage
```

and add the `storage` import to `eventlog/eventlog.go`'s import block:

```go
	"github.com/ITProLabDev/ethbacknode/storage"
```

Add the `all()` method and `scopeName` helper. Append `all()` to `eventlog/subscriptions.go`:

```go
// all returns a flat copy of every subscription across all addresses.
func (s *subscriptionSet) all() []*Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Subscription
	for _, subs := range s.byAddr {
		out = append(out, subs...)
	}
	return out
}

// scopeName returns the canonical string name for a scope (inverse of
// ParseScope), used for persistence.
func scopeName(sc Scope) string {
	if sc == ScopeManagedOnly {
		return "managed_only"
	}
	return "whole_contract"
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./eventlog/ -run 'TestService_SubscribePersists|TestService_LoadSubscriptions' -v` then `go test ./eventlog/ -count=1`
Expected: PASS; full package `ok`.

- [ ] **Step 5: Commit**

```bash
git add eventlog/persist.go eventlog/persist_test.go eventlog/eventlog.go eventlog/subscriptions.go
git commit -m "feat(eventlog): persist subscriptions to storage (load/save JSON)"
```

---

### Task 3: contractEvent delivery sink (eventlog → subscriptions)

**Files:**
- Create: `endpoint/contract_delivery.go`
- Test: `endpoint/contract_delivery_test.go`

- [ ] **Step 1: Write the failing test** — create `endpoint/contract_delivery_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./endpoint/ -run TestContractEventSink -v`
Expected: FAIL — `NewContractEventSink` / `contractEventPayload` undefined.

- [ ] **Step 3: Write minimal implementation** — create `endpoint/contract_delivery.go`:

```go
package endpoint

import (
	"strconv"

	"github.com/ITProLabDev/ethbacknode/abi"
	"github.com/ITProLabDev/ethbacknode/eventlog"
	"github.com/ITProLabDev/ethbacknode/tools/log"
)

// ContractEventNotifier delivers a contract-event payload to a subscriber.
// Satisfied by an adapter over subscriptions.Manager (wired in main.go).
type ContractEventNotifier interface {
	NotifyContractEvent(serviceID int64, subject string, payload interface{})
}

// contractEventPayload is the JSON-RPC notification body for a contractEvent.
type contractEventPayload struct {
	Event    string             `json:"event"`            // event name
	Contract string             `json:"contract"`         // contract address
	BlockNum int64              `json:"blockNum"`         // block number
	TxHash   string             `json:"txHash"`           // transaction hash
	TxIndex  int64              `json:"txIndex"`          // tx index in block
	LogIndex int64              `json:"logIndex"`         // log index in block
	Removed  bool               `json:"removed,omitempty"`// true if reverted by reorg
	Inputs   []abi.DecodedValue `json:"inputs"`           // decoded event parameters
}

// NewContractEventSink builds an eventlog.Sink that routes each decoded
// contract event to its subscriber via the notifier. It is concurrency-safe as
// long as the notifier is (subscriptions.Manager is).
func NewContractEventSink(notifier ContractEventNotifier) eventlog.Sink {
	return func(ce *eventlog.ContractEvent) {
		serviceID, err := strconv.ParseInt(ce.ServiceID, 10, 64)
		if err != nil {
			log.Error("contractEvent: bad serviceID", ce.ServiceID, ":", err)
			return
		}
		payload := &contractEventPayload{
			Event:    ce.Event.Name,
			Contract: ce.Event.Contract,
			BlockNum: ce.BlockNumber,
			TxHash:   ce.TransactionHash,
			TxIndex:  ce.TransactionIndex,
			LogIndex: ce.LogIndex,
			Removed:  ce.Removed,
			Inputs:   ce.Event.Inputs,
		}
		notifier.NotifyContractEvent(serviceID, "contractEvent", payload)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./endpoint/ -run TestContractEventSink -v`
Expected: PASS (both).

- [ ] **Step 5: Commit**

```bash
git add endpoint/contract_delivery.go endpoint/contract_delivery_test.go
git commit -m "feat(endpoint): contractEvent delivery sink routing to subscribers"
```

---

### Task 4: contract registration + listing + subscribe RPC methods

**Files:**
- Create: `endpoint/methods_contracts.go`
- Modify: `endpoint/rpc_handler.go` (add deps + options)
- Modify: `endpoint/rpc_init.go` (register methods)
- Test: `endpoint/methods_contracts_test.go`

- [ ] **Step 1: Add deps to BackRpc** — in `endpoint/rpc_handler.go`, add fields to the `BackRpc` struct (after `burnAddress`):

```go
	abiManager   ContractRegistry
	eventLog     EventSubscriber
}
```

and define the two narrow interfaces + their options at the end of `rpc_handler.go`:

```go
// ContractRegistry is the subset of abi.SmartContractsManager the contract RPC
// methods need. Wired from *abi.SmartContractsManager in main.go.
type ContractRegistry interface {
	GetSmartContractList() map[string]string
}

// ContractAdder registers a new contract. Split out so registration can be
// wired without exposing the whole manager. Satisfied by *abi.SmartContractsManager.
type ContractAdder interface {
	AddContractFromABI(name, symbol, address string, rawABI []byte) error
}

// EventSubscriber is the subset of eventlog.Service the subscribe RPC needs.
type EventSubscriber interface {
	SubscribeAndSave(serviceID, contractAddress, scope string) error
	ListSubscriptions() []map[string]string
}

// WithAbiManager wires the contract registry/adder used by contract RPC methods.
func WithAbiManager(reg ContractRegistry, adder ContractAdder) BackRpcOption {
	return func(r *BackRpc) { r.abiManager = reg; r.contractAdder = adder }
}

// WithEventSubscriber wires the eventlog subscription surface.
func WithEventSubscriber(es EventSubscriber) BackRpcOption {
	return func(r *BackRpc) { r.eventLog = es }
}
```

Also add `contractAdder ContractAdder` and `contractCaller ContractCaller` fields to the struct (ContractCaller is defined in Task 5; declare the field now and the interface in Task 5). To keep this task self-contained, add only `contractAdder ContractAdder` here; the `contractCaller` field + interface are added in Task 5.

Final struct addition for THIS task (place inside the struct):

```go
	abiManager    ContractRegistry
	contractAdder ContractAdder
	eventLog      EventSubscriber
```

- [ ] **Step 2: Write the failing test** — create `endpoint/methods_contracts_test.go`:

```go
package endpoint

import (
	"encoding/json"
	"testing"
)

// --- fakes for the contract RPC deps ---

type fakeRegistry struct{ list map[string]string }

func (f *fakeRegistry) GetSmartContractList() map[string]string { return f.list }

type fakeAdder struct {
	name, symbol, address string
	rawABI                []byte
	err                   error
}

func (f *fakeAdder) AddContractFromABI(name, symbol, address string, rawABI []byte) error {
	f.name, f.symbol, f.address, f.rawABI = name, symbol, address, rawABI
	return f.err
}

type fakeSubscriber struct {
	gotService, gotAddr, gotScope string
	err                           error
	subs                          []map[string]string
}

func (f *fakeSubscriber) SubscribeAndSave(serviceID, contractAddress, scope string) error {
	f.gotService, f.gotAddr, f.gotScope = serviceID, contractAddress, scope
	return f.err
}
func (f *fakeSubscriber) ListSubscriptions() []map[string]string { return f.subs }

// fakeReq/fakeResp implement RpcRequest/RpcResponse for processor tests.
type fakeReq struct{ params interface{} }

func (r *fakeReq) GetMethod() RpcMethod                           { return "" }
func (r *fakeReq) ParseParams(p interface{}) error                { b, _ := json.Marshal(r.params); return json.Unmarshal(b, p) }
func (r *fakeReq) GetParamString(string) (string, error)          { return "", nil }
func (r *fakeReq) GetParamInt(string) (int64, error)              { return 0, nil }
func (r *fakeReq) GetParamBool(string) (bool, error)              { return false, nil }

type fakeResp struct {
	result   interface{}
	errCode  int
	errMsg   string
	hasError bool
}

func (r *fakeResp) SetResult(v interface{})                  { r.result = v }
func (r *fakeResp) SetError(code int, msg string)            { r.errCode = code; r.errMsg = msg; r.hasError = true }
func (r *fakeResp) SetErrorWithData(code int, msg, d string) { r.errCode = code; r.errMsg = msg; r.hasError = true }

func TestContractRegister_AddsContract(t *testing.T) {
	adder := &fakeAdder{}
	r := &BackRpc{contractAdder: adder}
	req := &fakeReq{params: map[string]interface{}{
		"name": "Tok", "symbol": "TOK", "address": "0xabc",
		"abi": `[{"type":"event","name":"E","inputs":[]}]`,
	}}
	resp := &fakeResp{}
	r.rpcProcessContractRegister(nil, req, resp)
	if resp.hasError {
		t.Fatalf("unexpected error: %s", resp.errMsg)
	}
	if adder.name != "Tok" || adder.address != "0xabc" || len(adder.rawABI) == 0 {
		t.Fatalf("adder got name=%q addr=%q abilen=%d", adder.name, adder.address, len(adder.rawABI))
	}
}

func TestContractList_ReturnsRegistry(t *testing.T) {
	r := &BackRpc{abiManager: &fakeRegistry{list: map[string]string{"Tok": "0xabc"}}}
	resp := &fakeResp{}
	r.rpcProcessContractList(nil, &fakeReq{}, resp)
	if resp.hasError {
		t.Fatal("unexpected error")
	}
	m, ok := resp.result.(map[string]string)
	if !ok || m["Tok"] != "0xabc" {
		t.Fatalf("result=%v", resp.result)
	}
}

func TestContractSubscribe_RegistersSubscription(t *testing.T) {
	sub := &fakeSubscriber{}
	r := &BackRpc{eventLog: sub}
	req := &fakeReq{params: map[string]interface{}{
		"serviceId": "7", "address": "0xabc", "scope": "managed_only",
	}}
	resp := &fakeResp{}
	r.rpcProcessContractSubscribe(nil, req, resp)
	if resp.hasError {
		t.Fatalf("unexpected error: %s", resp.errMsg)
	}
	if sub.gotService != "7" || sub.gotAddr != "0xabc" || sub.gotScope != "managed_only" {
		t.Fatalf("subscriber got %q %q %q", sub.gotService, sub.gotAddr, sub.gotScope)
	}
}

func TestContractSubscribe_RejectsBadScope(t *testing.T) {
	sub := &fakeSubscriber{err: errBadScope}
	r := &BackRpc{eventLog: sub}
	req := &fakeReq{params: map[string]interface{}{"serviceId": "7", "address": "0xabc", "scope": "nonsense"}}
	resp := &fakeResp{}
	r.rpcProcessContractSubscribe(nil, req, resp)
	if !resp.hasError {
		t.Fatal("bad scope must error")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./endpoint/ -run 'TestContractRegister|TestContractList|TestContractSubscribe' -v`
Expected: FAIL — processors / `errBadScope` undefined.

- [ ] **Step 4: Write minimal implementation** — create `endpoint/methods_contracts.go`:

```go
package endpoint

import "errors"

// errBadScope is returned when a subscription scope is invalid. (The eventlog
// layer validates the scope string; this sentinel is for test clarity.)
var errBadScope = errors.New("invalid scope")

// rpcProcessContractRegister registers a contract + ABI. params:
// {name, symbol, address, abi (canonical JSON ABI string)}.
func (r *BackRpc) rpcProcessContractRegister(ctx RequestContext, request RpcRequest, response RpcResponse) {
	type params struct {
		Name    string `json:"name"`
		Symbol  string `json:"symbol"`
		Address string `json:"address"`
		ABI     string `json:"abi"`
	}
	p := &params{}
	if err := request.ParseParams(p); err != nil {
		response.SetError(ERROR_CODE_PARSE_ERROR, ERROR_MESSAGE_PARSE_ERROR)
		return
	}
	if p.Address == "" || p.ABI == "" {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "address and abi are required")
		return
	}
	if r.contractAdder == nil {
		response.SetError(ERROR_CODE_SERVER_ERROR, "contract registry not configured")
		return
	}
	if err := r.contractAdder.AddContractFromABI(p.Name, p.Symbol, p.Address, []byte(p.ABI)); err != nil {
		response.SetError(ERROR_CODE_INVALID_REQUEST, err.Error())
		return
	}
	response.SetResult(map[string]string{"name": p.Name, "address": p.Address, "status": "registered"})
}

// rpcProcessContractList returns the registered contracts as name→address.
func (r *BackRpc) rpcProcessContractList(ctx RequestContext, request RpcRequest, response RpcResponse) {
	if r.abiManager == nil {
		response.SetError(ERROR_CODE_SERVER_ERROR, "contract registry not configured")
		return
	}
	response.SetResult(r.abiManager.GetSmartContractList())
}

// rpcProcessContractSubscribe subscribes a service to a contract's events.
// params: {serviceId, address, scope (whole_contract|managed_only)}.
func (r *BackRpc) rpcProcessContractSubscribe(ctx RequestContext, request RpcRequest, response RpcResponse) {
	type params struct {
		ServiceID string `json:"serviceId"`
		Address   string `json:"address"`
		Scope     string `json:"scope"`
	}
	p := &params{}
	if err := request.ParseParams(p); err != nil {
		response.SetError(ERROR_CODE_PARSE_ERROR, ERROR_MESSAGE_PARSE_ERROR)
		return
	}
	if p.ServiceID == "" || p.Address == "" {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "serviceId and address are required")
		return
	}
	if r.eventLog == nil {
		response.SetError(ERROR_CODE_SERVER_ERROR, "event subscriber not configured")
		return
	}
	if err := r.eventLog.SubscribeAndSave(p.ServiceID, p.Address, p.Scope); err != nil {
		response.SetError(ERROR_CODE_INVALID_REQUEST, err.Error())
		return
	}
	response.SetResult(map[string]string{"serviceId": p.ServiceID, "address": p.Address, "scope": p.Scope, "status": "subscribed"})
}

// rpcProcessContractListSubscriptions returns the active event subscriptions.
func (r *BackRpc) rpcProcessContractListSubscriptions(ctx RequestContext, request RpcRequest, response RpcResponse) {
	if r.eventLog == nil {
		response.SetError(ERROR_CODE_SERVER_ERROR, "event subscriber not configured")
		return
	}
	response.SetResult(r.eventLog.ListSubscriptions())
}
```

- [ ] **Step 5: Register the methods** — in `endpoint/rpc_init.go`, add to `InitProcessors()` (after the existing registrations):

```go
	r.RegisterProcessor("contract.register", r.rpcProcessContractRegister)
	r.RegisterProcessor("contractRegister", r.rpcProcessContractRegister)
	r.RegisterProcessor("contract.list", r.rpcProcessContractList)
	r.RegisterProcessor("contractList", r.rpcProcessContractList)
	r.RegisterProcessor("contract.subscribe", r.rpcProcessContractSubscribe)
	r.RegisterProcessor("contractSubscribe", r.rpcProcessContractSubscribe)
	r.RegisterProcessor("contract.subscriptions", r.rpcProcessContractListSubscriptions)
	r.RegisterProcessor("contractSubscriptions", r.rpcProcessContractListSubscriptions)
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./endpoint/ -run 'TestContractRegister|TestContractList|TestContractSubscribe' -v`
Expected: PASS (all four). Then `go build ./endpoint/` → exit 0.

- [ ] **Step 7: Commit**

```bash
git add endpoint/methods_contracts.go endpoint/rpc_handler.go endpoint/rpc_init.go endpoint/methods_contracts_test.go
git commit -m "feat(endpoint): contract register/list/subscribe RPC methods"
```

---

### Task 5: contractCall read-only RPC

**Files:**
- Modify: `endpoint/rpc_handler.go` (add `ContractCaller` interface + field + option)
- Create: `endpoint/methods_contract_call.go`
- Modify: `endpoint/rpc_init.go` (register)
- Test: `endpoint/methods_contract_call_test.go`

- [ ] **Step 1: Add the ContractCaller interface + wiring** — in `endpoint/rpc_handler.go`, add the field to the struct (next to the other M6 fields):

```go
	contractCaller ContractCaller
```

and define the interface + option at the end of the file:

```go
// ContractCaller invokes a contract view method by name and returns decoded
// outputs. Satisfied by *ethclient.Client (M4 CallMethod), wired in main.go —
// kept narrow so endpoint does not import clients/ethclient.
type ContractCaller interface {
	CallMethod(contractAddress, methodName string, args ...any) ([]abi.DecodedValue, error)
}

// WithContractCaller wires the view-call surface used by contractCall.
func WithContractCaller(c ContractCaller) BackRpcOption {
	return func(r *BackRpc) { r.contractCaller = c }
}
```

Add the `abi` import to `rpc_handler.go` if not present: `"github.com/ITProLabDev/ethbacknode/abi"`.

- [ ] **Step 2: Write the failing test** — create `endpoint/methods_contract_call_test.go`:

```go
package endpoint

import (
	"math/big"
	"testing"

	"github.com/ITProLabDev/ethbacknode/abi"
)

type fakeCaller struct {
	gotContract, gotMethod string
	gotArgs                []any
	out                    []abi.DecodedValue
	err                    error
}

func (f *fakeCaller) CallMethod(contract, method string, args ...any) ([]abi.DecodedValue, error) {
	f.gotContract, f.gotMethod, f.gotArgs = contract, method, args
	return f.out, f.err
}

func TestContractCall_ReturnsDecodedOutputs(t *testing.T) {
	caller := &fakeCaller{out: []abi.DecodedValue{{Name: "", Type: "uint256", Value: big.NewInt(42)}}}
	r := &BackRpc{contractCaller: caller}
	req := &fakeReq{params: map[string]interface{}{
		"address": "0xabc", "method": "totalSupply",
	}}
	resp := &fakeResp{}
	r.rpcProcessContractCall(nil, req, resp)
	if resp.hasError {
		t.Fatalf("unexpected error: %s", resp.errMsg)
	}
	if caller.gotContract != "0xabc" || caller.gotMethod != "totalSupply" {
		t.Fatalf("caller got %q.%q", caller.gotContract, caller.gotMethod)
	}
	out, ok := resp.result.([]abi.DecodedValue)
	if !ok || len(out) != 1 || out[0].Value.(*big.Int).Int64() != 42 {
		t.Fatalf("result=%v", resp.result)
	}
}

func TestContractCall_PropagatesError(t *testing.T) {
	caller := &fakeCaller{err: errBadScope} // any non-nil error
	r := &BackRpc{contractCaller: caller}
	req := &fakeReq{params: map[string]interface{}{"address": "0xabc", "method": "x"}}
	resp := &fakeResp{}
	r.rpcProcessContractCall(nil, req, resp)
	if !resp.hasError {
		t.Fatal("call error must surface as RPC error")
	}
}

func TestContractCall_RequiresAddressAndMethod(t *testing.T) {
	r := &BackRpc{contractCaller: &fakeCaller{}}
	resp := &fakeResp{}
	r.rpcProcessContractCall(nil, &fakeReq{params: map[string]interface{}{"address": "0xabc"}}, resp)
	if !resp.hasError {
		t.Fatal("missing method must error")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./endpoint/ -run TestContractCall -v`
Expected: FAIL — `rpcProcessContractCall` undefined.

- [ ] **Step 4: Write minimal implementation** — create `endpoint/methods_contract_call.go`:

```go
package endpoint

// rpcProcessContractCall invokes a contract view method and returns its decoded
// outputs. params: {address, method, args (optional, positional)}.
//
// NOTE: args support in this first version is limited to no-arg view methods
// (e.g. totalSupply, decimals). Typed-argument encoding from JSON is a
// follow-up; passing args returns an error to avoid silently mis-encoding.
func (r *BackRpc) rpcProcessContractCall(ctx RequestContext, request RpcRequest, response RpcResponse) {
	type params struct {
		Address string `json:"address"`
		Method  string `json:"method"`
		Args    []any  `json:"args"`
	}
	p := &params{}
	if err := request.ParseParams(p); err != nil {
		response.SetError(ERROR_CODE_PARSE_ERROR, ERROR_MESSAGE_PARSE_ERROR)
		return
	}
	if p.Address == "" || p.Method == "" {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "address and method are required")
		return
	}
	if len(p.Args) > 0 {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "contractCall with arguments is not yet supported")
		return
	}
	if r.contractCaller == nil {
		response.SetError(ERROR_CODE_SERVER_ERROR, "contract caller not configured")
		return
	}
	out, err := r.contractCaller.CallMethod(p.Address, p.Method)
	if err != nil {
		response.SetError(ERROR_CODE_INVALID_REQUEST, err.Error())
		return
	}
	response.SetResult(out)
}
```

- [ ] **Step 5: Register the method** — in `endpoint/rpc_init.go`, add to `InitProcessors()`:

```go
	r.RegisterProcessor("contract.call", r.rpcProcessContractCall)
	r.RegisterProcessor("contractCall", r.rpcProcessContractCall)
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./endpoint/ -run TestContractCall -v` then `go test ./endpoint/ -count=1`
Expected: PASS (all three); full endpoint package `ok`.

- [ ] **Step 7: Commit**

```bash
git add endpoint/methods_contract_call.go endpoint/rpc_handler.go endpoint/rpc_init.go endpoint/methods_contract_call_test.go
git commit -m "feat(endpoint): contractCall read-only view-method RPC"
```

---

### Task 6: Manager-side adapters + wire everything in main.go

**Files:**
- Create: `eventlog/rpc_surface.go` (the `SubscribeAndSave(string,string,string)` + `ListSubscriptions` shape the endpoint expects)
- Modify: `main.go`
- Test: `eventlog/rpc_surface_test.go`

The endpoint's `EventSubscriber` interface expects `SubscribeAndSave(serviceID, contractAddress, scope string) error` and `ListSubscriptions() []map[string]string`. M5's `Service.SubscribeAndSave` takes a `*Subscription`. Add string-based wrappers so the endpoint stays decoupled from eventlog's types.

- [ ] **Step 1: Write the failing test** — create `eventlog/rpc_surface_test.go`:

```go
package eventlog

import "testing"

func TestSubscribeAndSaveStrings(t *testing.T) {
	st := &memStore{}
	svc := New(WithSubscriptionStorage(st))
	if err := svc.SubscribeAndSaveStrings("7", "0xABC", "managed_only"); err != nil {
		t.Fatal(err)
	}
	subs := svc.subs.forAddress("0xabc")
	if len(subs) != 1 || subs[0].ServiceID != "7" || subs[0].Scope != ScopeManagedOnly {
		t.Fatalf("subs=%+v", subs)
	}
	if !st.exists {
		t.Fatal("must persist")
	}
}

func TestSubscribeAndSaveStrings_BadScope(t *testing.T) {
	svc := New()
	if err := svc.SubscribeAndSaveStrings("7", "0xABC", "bogus"); err == nil {
		t.Fatal("bad scope must error")
	}
}

func TestListSubscriptions(t *testing.T) {
	svc := New()
	_ = svc.SubscribeAndSaveStrings("1", "0xAA", "whole_contract")
	_ = svc.SubscribeAndSaveStrings("2", "0xBB", "managed_only")
	list := svc.ListSubscriptions()
	if len(list) != 2 {
		t.Fatalf("list=%v", list)
	}
	// Each entry exposes serviceId, address, scope as strings.
	for _, m := range list {
		if m["serviceId"] == "" || m["address"] == "" || m["scope"] == "" {
			t.Fatalf("incomplete entry: %v", m)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./eventlog/ -run 'TestSubscribeAndSaveStrings|TestListSubscriptions' -v`
Expected: FAIL — methods undefined.

- [ ] **Step 3: Write minimal implementation** — create `eventlog/rpc_surface.go`:

```go
package eventlog

// SubscribeAndSaveStrings registers a subscription from string parameters
// (as received over RPC) and persists it. The scope string is validated via
// ParseScope. This is the surface the endpoint's EventSubscriber wires to.
func (s *Service) SubscribeAndSaveStrings(serviceID, contractAddress, scope string) error {
	sc, err := ParseScope(scope)
	if err != nil {
		return err
	}
	return s.SubscribeAndSave(&Subscription{
		ServiceID:       serviceID,
		ContractAddress: contractAddress,
		Scope:           sc,
	})
}

// ListSubscriptions returns all subscriptions as string maps (serviceId,
// address, scope) for RPC responses.
func (s *Service) ListSubscriptions() []map[string]string {
	all := s.subs.all()
	out := make([]map[string]string, len(all))
	for i, sub := range all {
		out[i] = map[string]string{
			"serviceId": sub.ServiceID,
			"address":   sub.ContractAddress,
			"scope":     scopeName(sub.Scope),
		}
	}
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./eventlog/ -run 'TestSubscribeAndSaveStrings|TestListSubscriptions' -v` then `go test ./eventlog/ -count=1`
Expected: PASS; full package `ok`.

- [ ] **Step 5: Add an abi-manager adapter for ContractAdder** — the endpoint's `ContractAdder` expects `AddContractFromABI(name, symbol, address string, rawABI []byte) error`. `abi.SmartContractsManager` has `Add(*SmartContractInfo)` (no error) and `abi.NewContractFromABI(...) (*SmartContractInfo, error)`. Add a thin method to the abi manager. In a NEW file `abi/register.go`:

```go
package abi

// AddContractFromABI imports a canonical Ethereum JSON ABI and registers the
// contract in one step. Returns an error if the ABI is invalid.
func (m *SmartContractsManager) AddContractFromABI(name, symbol, address string, rawABI []byte) error {
	info, err := NewContractFromABI(name, symbol, address, rawABI)
	if err != nil {
		return err
	}
	m.Add(info)
	return nil
}
```

Create `abi/register_test.go`:

```go
package abi

import "testing"

func TestAddContractFromABI(t *testing.T) {
	m := newTestManager(t)
	raw := `[{"type":"event","name":"Transfer","inputs":[
	  {"name":"from","type":"address","indexed":true},
	  {"name":"to","type":"address","indexed":true},
	  {"name":"value","type":"uint256"}]}]`
	if err := m.AddContractFromABI("Tok", "TOK", "0xabc0000000000000000000000000000000000001", []byte(raw)); err != nil {
		t.Fatal(err)
	}
	if _, err := m.GetSmartContractByAddress("0xabc0000000000000000000000000000000000001"); err != nil {
		t.Fatalf("contract not registered: %v", err)
	}
	if err := m.AddContractFromABI("Bad", "BAD", "0xdef", []byte(`not json`)); err == nil {
		t.Fatal("invalid ABI must error")
	}
}
```

Run: `go test ./abi/ -run TestAddContractFromABI -v` → PASS.

- [ ] **Step 6: Wire everything in main.go.** Two changes:

(a) Replace the M5 logging sink with the delivery sink. The eventlog service is constructed in main.go (M5). Change the `eventlog.WithSink(...)` option to use the delivery sink, AND add subscription storage + load. Replace the existing logging sink block:

```go
		eventlog.WithSink(func(ce *eventlog.ContractEvent) {
			log.Info("contractEvent:", ce.Event.Name, "contract:", ce.Event.Contract, "block:", ce.BlockNumber, "tx:", ce.TransactionHash)
		}),
```

with a forward-declared sink wired after the subscriptions manager exists. Since the eventlog service is built before the endpoint, build the sink from the subscriptions manager directly. Add an eventlog subscription storage module and load it. Concretely, where the eventlog service is constructed, add:

```go
		eventlog.WithSubscriptionStorage(storageManager.GetModuleStorage("EventLog", "eventlog").GetBinFileStorage("subscriptions.json")),
		eventlog.WithSink(endpoint.NewContractEventSink(
			endpoint.NotifierFunc(func(serviceID int64, subject string, payload interface{}) {
				subscriptionsManager.NotifySubscriberRaw(subscriptions.ServiceId(serviceID), subject, payload)
			}),
		)),
```

and AFTER constructing `eventLogService`, load persisted subscriptions:

```go
	if err := eventLogService.LoadSubscriptions(); err != nil {
		log.Error("Can not load eventlog subscriptions:", err)
	}
```

(b) Wire the contract RPC deps into `NewBackRpc` via the new options:

```go
		endpoint.WithAbiManager(abiManager, abiManager),
		endpoint.WithEventSubscriber(eventLogService),
		endpoint.WithContractCaller(chainClient),
```

- [ ] **Step 7: Add the two main.go support pieces.**

(i) `endpoint.NotifierFunc` — a func adapter for `ContractEventNotifier`. Append to `endpoint/contract_delivery.go`:

```go
// NotifierFunc adapts a function to ContractEventNotifier.
type NotifierFunc func(serviceID int64, subject string, payload interface{})

func (f NotifierFunc) NotifyContractEvent(serviceID int64, subject string, payload interface{}) {
	f(serviceID, subject, payload)
}
```

(ii) `subscriptions.Manager.NotifySubscriberRaw` — deliver an arbitrary payload (not requiring `Signer`). `NotifySubscriber` requires a `Signer`; contract events do not sign, so add a sibling. Append to `subscriptions/notifysubscriber.go`:

```go
// NotifySubscriberRaw sends a notification with an arbitrary JSON payload to a
// subscriber (no Signer required). Used for contractEvent delivery.
func (s *Manager) NotifySubscriberRaw(serviceId ServiceId, subject string, payload interface{}) {
	s.subscribersMux.RLock()
	subscriber, found := s.subscribers[serviceId]
	s.subscribersMux.RUnlock()
	if !found {
		log.Error("NotifySubscriberRaw: unknown serviceId:", serviceId)
		return
	}
	subscriber.sendNotification(subject, payload, s.config.Debug)
}
```

(`sendNotification(method string, message interface{}, debug bool)` already exists and accepts any message.)

- [ ] **Step 8: Build and test everything**

Run: `go build ./...` (must exit 0 — confirms all wiring type-checks), then `go test ./eventlog/ ./endpoint/ ./abi/ -count=1` (ignore `[CRIT] Duplicated Contract` log lines).
Expected: build 0; all three packages `ok`.

- [ ] **Step 9: Commit**

```bash
git add eventlog/rpc_surface.go eventlog/rpc_surface_test.go abi/register.go abi/register_test.go endpoint/contract_delivery.go subscriptions/notifysubscriber.go main.go
git commit -m "feat(m6): wire contractEvent delivery + contract RPC into main"
```

---

### Task 7: Final verification & docs

**Files:**
- Modify: `todo/TASKS.md`

- [ ] **Step 1: Run affected packages with the race detector**

Run: `go test ./eventlog/ ./endpoint/ ./abi/ -race -count=1`
Expected: `ok` for all three (ignore `[CRIT] Duplicated Contract` log lines).

- [ ] **Step 2: Confirm nothing else broke**

Run: `go build ./... && go vet ./eventlog/ ./endpoint/ ./abi/ .`
Expected: both exit 0.

- [ ] **Step 3: Mark M6 tasks done in `todo/TASKS.md`**

Change each of `M6.1`–`M6.4` from `- [ ]` to `- [x]` in the "Milestone 6" section, append " ✅ DONE" to the header line, and add a status block (mirror M1–M5). Mapping:
- M6.1 `contractEvent` notification + extensible dispatch → Tasks 1, 3 (delivery sink via the existing per-subject sendNotification path; no dispatch-switch rewrite)
- M6.2 register/list/subscribe/unsubscribe RPC with scope → Task 4 (+ Task 6 wiring)
- M6.3 `contractCall` read-only RPC → Task 5
- M6.4 persist contracts (abi manager already persists on Add) + subscriptions (eventlog persist) → Tasks 2, 6

- [ ] **Step 4: Commit**

```bash
git add todo/TASKS.md
git commit -m "docs(m6): mark M6 (delivery & API) complete"
```

---

## Notes for the implementer

- **Extensibility (the user's requirement) is satisfied structurally:** contractEvent delivery reuses the existing `sendNotification(subject, payload)` path — a new event type is a new subject string + payload struct, not a dispatch-switch edit. New RPC methods are new `RegisterProcessor` lines into the existing map. No core rewrite.
- **The endpoint stays decoupled** from `abi`/`eventlog`/`ethclient` concrete types via narrow interfaces (`ContractRegistry`, `ContractAdder`, `EventSubscriber`, `ContractCaller`, `ContractEventNotifier`) wired in main.go — mirroring M5's LogSource pattern. The only concrete import the endpoint needs is `abi` for `abi.DecodedValue` in the call result + payload (a pure data type).
- **ServiceID is numeric end-to-end** but carried as a string in eventlog (M5's type) and parsed back to `int64` at the delivery boundary (Task 3). A non-numeric ServiceID is logged and skipped, never panics.
- **contractCall is no-arg in this first cut** (totalSupply/decimals/etc.). Passing args returns a clear "not yet supported" error rather than mis-encoding JSON values into ABI — typed-arg encoding from JSON is a deliberate follow-up. Note this in the API docs (M7).
- **Concurrency:** the delivery sink may be called from multiple goroutines (M5 contract); `NotifySubscriberRaw` is RLock-guarded and `sendNotification` does its own HTTP call per invocation — safe.
- **Persistence:** registered contracts are already durable (the abi manager Saves on `Add`). Eventlog subscriptions get `data/eventlog/subscriptions.json` and are reloaded on startup (`LoadSubscriptions`).
- **Unsubscribe** is listed in M6.2 but is YAGNI for the first cut unless trivially addable — the plan implements register/list/subscribe/list-subscriptions. If you add unsubscribe, mirror `SubscribeAndSaveStrings` with a `RemoveSubscription(serviceID, address)` on the set + a `contractUnsubscribe` processor; otherwise note it deferred when marking M6.2.
- Run only the affected packages' tests; the repo has a pre-existing unrelated compile error in `crypto/secp256k1`.
