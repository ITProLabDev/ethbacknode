# M5 — Event-Log Service Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A new `eventlog/` service that, on each new block, collects the registered contracts' event logs from the chain, decodes them with the ABI engine, filters them per subscription scope (`whole_contract` vs `managed_only`), and hands decoded events to a delivery callback — plugging into the existing watchdog as a block listener without changing watchdog internals.

**Architecture:** Approach A — `abi/` stays a pure codec library; `eventlog/` is a thin service that depends on a narrow `LogSource` interface (satisfied by `*ethclient.Client`'s M4 `GetLogs`/`GetTransactionReceipt`), the `abi` registry (`DecodeLog`), and an address-membership check. It registers via `watchdog.RegisterBlockEventListen`, so M5 adds zero lines to watchdog. Two collection modes (Mode A: `eth_getLogs` per block; Mode B: per-receipt) feed one decoded-log stream; subscription scope filtering happens after decode.

**Tech Stack:** Go 1.24, the M4 `clients/ethclient` (`GetLogs`/`LogFilter`/`Receipt`/`Log`/`Topics32`), the M1–M3 `abi` engine (`SmartContractsManager.DecodeLog`, `DecodedEvent`), `tools/flow.FanOut` (Mode B parallel receipts), the existing `address.Manager` and `watchdog` listener API, standard `testing`.

---

## Background & Constraints

**Hard rule:** self-built stack only — NO go-ethereum / external ABI/RPC library.

**How M5 plugs in (verified against the code):**
- `watchdog.Service` exposes `RegisterBlockEventListen(func(blockNum int64, blockId string))` (watchdog/event.go). On each new block the watchdog calls every registered block handler. `eventlog.Service.OnBlock(blockNum, blockId)` is registered there in `main.go` — that is the entire watchdog integration (M5.5). No change to `watchdog/processblock.go` or the `Service` struct.
- The watchdog's chain client is `types.ChainClient` (an interface that does NOT include `GetLogs`/`GetTransactionReceipt`). Those M4 methods live on the concrete `*ethclient.Client`. So `eventlog` defines its OWN narrow interface `LogSource` (just the methods it needs), which `*ethclient.Client` satisfies structurally. This keeps `eventlog` decoupled and unit-testable with a fake `LogSource`.
- Decoding: `abi.(*SmartContractsManager).DecodeLog(contractAddress string, topics [][32]byte, data []byte) (*abi.DecodedEvent, error)` (M2/M3). A `clients/ethclient.Log` provides `Topics32() ([][32]byte, error)`, `Data []byte`, and `Address string` (M4) — exactly the inputs `DecodeLog` needs.
- `managed_only` scope: scan a `DecodedEvent.Inputs` for address-typed values. The decoder emits an indexed/non-indexed address as `DecodedValue.Value` of type `[]byte` (20 bytes) with `DecodedValue.Type == "address"`. Encode via the address codec and test with `address.Manager.IsAddressKnown(string) bool`.

**Key types already available:**
- `clients/ethclient`: `(*Client).GetLogs(LogFilter) ([]*Log, error)`, `(*Client).GetTransactionReceipt(txHash string) (*Receipt, error)`; `LogFilter{ FromBlock, ToBlock int64; Addresses, Topics []string }`; `Log{ Address string; Topics []string; Data []byte; BlockNumber, TransactionIndex, LogIndex int64; TransactionHash string; Removed bool }` with `Topics32() ([][32]byte, error)`; `Receipt{ ... Logs []*Log }`; `GetAddressCodec() address.AddressCodec` with `EncodeBytesToAddress([]byte) (string, error)`.
- `abi`: `(*SmartContractsManager).DecodeLog(...)`, `(*SmartContractsManager).GetSmartContractList() map[string]string` (name→address), `(*SmartContractsManager).Walk(func(*SmartContractInfo))`, `DecodedEvent{ Name, Contract string; Inputs []DecodedValue }`, `DecodedValue{ Name, Type string; Value any }`.
- `address`: `(*Manager).IsAddressKnown(addr string) bool`.
- `tools/flow`: `FanOut[T any](ctx context.Context, items []T, concurrency int, step func(context.Context, T) (T, error)) ([]T, error)`.

**Decisions locked for M5:**
- M5 produces decoded events and delivers them to a `Sink` callback (`func(*ContractEvent)`). It does NOT itself do JSON-RPC subscriber notification — that's M6. M5's job ends at "decoded + scope-filtered event handed to a sink."
- A `ContractEvent` wraps the `abi.DecodedEvent` plus block/tx context (block number, tx hash, log index) so M6 has everything to deliver.
- Subscriptions in M5 are an in-memory registry keyed by contract address, each carrying a `Scope`. M6 will persist/expose them via RPC; M5 only needs the in-memory model + matching logic. (No storage in M5.)
- Mode is chosen by config (`getLogs` default | `receipts`); subscription SCOPE (`whole_contract`|`managed_only`) is independent of mode and applied after decode.
- The carry-over perf note (cached topic0 map) is NOT implemented here; `DecodeLog` already works. Revisit only if profiling shows it hot.

---

## File Structure

- **Create `eventlog/eventlog.go`** — `Service`, `LogSource` interface, `Sink`, `ContractEvent`, options (`New(...)`), `OnBlock(blockNum int64, blockId string)`.
- **Create `eventlog/subscriptions.go`** — `Scope` type + consts, `Subscription`, in-memory `subscriptionSet` (register/list/match-by-address), and `scopeMatch(ev, scope, isManaged)` logic.
- **Create `eventlog/collect.go`** — `collectModeGetLogs` (Mode A) and `collectModeReceipts` (Mode B) producing `[]*ethclient.Log` for a block; `decodeAndDeliver` shared tail.
- **Create `eventlog/config.go`** — `Mode` type/consts, `Config` (mode + receipt concurrency), default.
- **Create test files** alongside each.
- **Modify `main.go`** — construct `eventlog.Service`, register `OnBlock` as a watchdog block listener (M5.5 wiring).

No changes to `abi/`, `watchdog/` internals, or `clients/ethclient` beyond what M4 added.

---

### Task 1: Subscription model + scope matching (`subscriptions.go`)

**Files:**
- Create: `eventlog/subscriptions.go`
- Test: `eventlog/subscriptions_test.go`

- [ ] **Step 1: Write the failing test** — create `eventlog/subscriptions_test.go`:

```go
package eventlog

import "testing"

func TestScope_Parse(t *testing.T) {
	cases := map[string]Scope{
		"whole_contract": ScopeWholeContract,
		"WHOLE_CONTRACT": ScopeWholeContract,
		"managed_only":   ScopeManagedOnly,
		"Managed_Only":   ScopeManagedOnly,
	}
	for in, want := range cases {
		got, err := ParseScope(in)
		if err != nil || got != want {
			t.Fatalf("ParseScope(%q)=%v,%v want %v", in, got, err, want)
		}
	}
	if _, err := ParseScope("nonsense"); err == nil {
		t.Fatal("unknown scope must error")
	}
}

func TestSubscriptionSet_RegisterAndMatch(t *testing.T) {
	set := newSubscriptionSet()
	addr := "0xAbC0000000000000000000000000000000000001"
	set.add(&Subscription{ServiceID: "svc1", ContractAddress: addr, Scope: ScopeWholeContract})
	set.add(&Subscription{ServiceID: "svc2", ContractAddress: addr, Scope: ScopeManagedOnly})

	// Lookup is case-insensitive on address.
	subs := set.forAddress("0xabc0000000000000000000000000000000000001")
	if len(subs) != 2 {
		t.Fatalf("forAddress=%d want 2", len(subs))
	}
	if len(set.forAddress("0x9999999999999999999999999999999999999999")) != 0 {
		t.Fatal("unknown address must match nothing")
	}
}

func TestSubscriptionSet_Addresses(t *testing.T) {
	set := newSubscriptionSet()
	set.add(&Subscription{ServiceID: "a", ContractAddress: "0xAA", Scope: ScopeWholeContract})
	set.add(&Subscription{ServiceID: "b", ContractAddress: "0xaa", Scope: ScopeManagedOnly}) // same addr, diff case
	set.add(&Subscription{ServiceID: "c", ContractAddress: "0xBB", Scope: ScopeWholeContract})
	addrs := set.addresses()
	if len(addrs) != 2 {
		t.Fatalf("addresses=%v want 2 unique (0xaa,0xbb lowercased)", addrs)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./eventlog/ -run 'TestScope|TestSubscriptionSet' -v`
Expected: FAIL — `undefined: Scope` / package not found.

- [ ] **Step 3: Write minimal implementation** — create `eventlog/subscriptions.go`:

```go
// Package eventlog collects, decodes, and scope-filters smart-contract event
// logs per block, handing decoded events to a delivery sink. It plugs into the
// watchdog as a block listener and keeps the abi/ engine a pure library.
package eventlog

import (
	"fmt"
	"strings"
	"sync"
)

// Scope selects which of a contract's events a subscription receives.
type Scope int

const (
	// ScopeWholeContract delivers every event of the contract, regardless of
	// which addresses are involved.
	ScopeWholeContract Scope = iota
	// ScopeManagedOnly delivers only events that involve a managed address
	// (matched inside the event's address-typed parameters).
	ScopeManagedOnly
)

// ParseScope parses a scope name (case-insensitive): "whole_contract" or
// "managed_only".
func ParseScope(s string) (Scope, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "whole_contract":
		return ScopeWholeContract, nil
	case "managed_only":
		return ScopeManagedOnly, nil
	default:
		return 0, fmt.Errorf("unknown event scope %q (want whole_contract | managed_only)", s)
	}
}

// Subscription is an in-memory contract-event subscription. ServiceID
// identifies the subscriber (for M6 delivery); ContractAddress is the contract
// whose events are wanted; Scope selects the filtering mode.
type Subscription struct {
	ServiceID       string
	ContractAddress string
	Scope           Scope
}

// subscriptionSet is a concurrency-safe in-memory set of subscriptions keyed by
// lowercased contract address.
type subscriptionSet struct {
	mu    sync.RWMutex
	byAddr map[string][]*Subscription
}

func newSubscriptionSet() *subscriptionSet {
	return &subscriptionSet{byAddr: make(map[string][]*Subscription)}
}

func (s *subscriptionSet) add(sub *Subscription) {
	key := strings.ToLower(sub.ContractAddress)
	s.mu.Lock()
	s.byAddr[key] = append(s.byAddr[key], sub)
	s.mu.Unlock()
}

// forAddress returns the subscriptions registered for a contract address
// (case-insensitive). The returned slice is a copy safe to read concurrently.
func (s *subscriptionSet) forAddress(addr string) []*Subscription {
	key := strings.ToLower(addr)
	s.mu.RLock()
	defer s.mu.RUnlock()
	src := s.byAddr[key]
	out := make([]*Subscription, len(src))
	copy(out, src)
	return out
}

// addresses returns the unique (lowercased) contract addresses with at least
// one subscription — used to build the eth_getLogs address filter.
func (s *subscriptionSet) addresses() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.byAddr))
	for addr := range s.byAddr {
		out = append(out, addr)
	}
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./eventlog/ -run 'TestScope|TestSubscriptionSet' -v`
Expected: PASS (all three).

- [ ] **Step 5: Commit**

```bash
git add eventlog/subscriptions.go eventlog/subscriptions_test.go
git commit -m "feat(eventlog): subscription model with scope and address index"
```

---

### Task 2: ContractEvent + managed-address scope filtering

**Files:**
- Create: `eventlog/event.go`
- Test: `eventlog/event_test.go`

- [ ] **Step 1: Write the failing test** — create `eventlog/event_test.go`:

```go
package eventlog

import (
	"testing"

	"github.com/ITProLabDev/ethbacknode/abi"
)

// stubManaged implements the managedAddresses interface for tests.
type stubManaged struct{ known map[string]bool }

func (s stubManaged) IsAddressKnown(addr string) bool { return s.known[addr] }

// stubCodec encodes a 20-byte slice to a lowercase 0x-hex string.
type stubCodec struct{}

func (stubCodec) EncodeBytesToAddress(b []byte) (string, error) {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, 0, 2+len(b)*2)
	out = append(out, '0', 'x')
	for _, by := range b {
		out = append(out, hexdigits[by>>4], hexdigits[by&0xf])
	}
	return string(out), nil
}

func addr20(last byte) []byte {
	b := make([]byte, 20)
	b[19] = last
	return b
}

func TestEventMatchesScope_WholeContract(t *testing.T) {
	ev := &abi.DecodedEvent{Name: "X", Inputs: []abi.DecodedValue{
		{Name: "to", Type: "address", Value: addr20(0x11)},
	}}
	managed := stubManaged{known: map[string]bool{}} // nothing managed
	// whole_contract always matches, even with no managed address involved.
	if !eventMatchesScope(ev, ScopeWholeContract, managed, stubCodec{}) {
		t.Fatal("whole_contract must always match")
	}
}

func TestEventMatchesScope_ManagedOnly(t *testing.T) {
	codec := stubCodec{}
	managedAddr, _ := codec.EncodeBytesToAddress(addr20(0x22))
	managed := stubManaged{known: map[string]bool{managedAddr: true}}

	hit := &abi.DecodedEvent{Name: "Transfer", Inputs: []abi.DecodedValue{
		{Name: "from", Type: "address", Value: addr20(0x11)},
		{Name: "to", Type: "address", Value: addr20(0x22)}, // managed
		{Name: "value", Type: "uint256", Value: nil},
	}}
	if !eventMatchesScope(hit, ScopeManagedOnly, managed, codec) {
		t.Fatal("managed_only must match when a managed address is involved")
	}

	miss := &abi.DecodedEvent{Name: "Transfer", Inputs: []abi.DecodedValue{
		{Name: "from", Type: "address", Value: addr20(0x11)},
		{Name: "to", Type: "address", Value: addr20(0x33)}, // not managed
	}}
	if eventMatchesScope(miss, ScopeManagedOnly, managed, codec) {
		t.Fatal("managed_only must NOT match when no managed address is involved")
	}
}

func TestEventMatchesScope_ManagedOnly_IgnoresNonAddress(t *testing.T) {
	managed := stubManaged{known: map[string]bool{}}
	ev := &abi.DecodedEvent{Name: "X", Inputs: []abi.DecodedValue{
		{Name: "n", Type: "uint256", Value: nil},
		{Name: "s", Type: "string", Value: "hi"},
	}}
	if eventMatchesScope(ev, ScopeManagedOnly, managed, stubCodec{}) {
		t.Fatal("managed_only with no address params must not match")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./eventlog/ -run TestEventMatchesScope -v`
Expected: FAIL — `undefined: eventMatchesScope` / `ContractEvent`.

- [ ] **Step 3: Write minimal implementation** — create `eventlog/event.go`:

```go
package eventlog

import "github.com/ITProLabDev/ethbacknode/abi"

// ContractEvent is a decoded contract event plus the block/tx context needed
// for delivery. It is the unit handed to a Sink.
type ContractEvent struct {
	Event            *abi.DecodedEvent // decoded event (name, contract, inputs)
	BlockNumber      int64
	TransactionHash  string
	LogIndex         int64
}

// ManagedAddresses is the subset of address.Manager that eventlog needs to
// decide managed_only matches. Satisfied by *address.Manager. Exported so
// main.go (another package) can wire it via WithManaged without friction.
type ManagedAddresses interface {
	IsAddressKnown(addr string) bool
}

// AddressEncoder turns a 20-byte address into its string form. Satisfied by
// the ethclient address codec. Exported for the same cross-package reason.
type AddressEncoder interface {
	EncodeBytesToAddress(b []byte) (string, error)
}

// eventMatchesScope reports whether a decoded event should be delivered to a
// subscription with the given scope. whole_contract always matches.
// managed_only matches iff at least one address-typed parameter of the event
// is a managed address.
func eventMatchesScope(ev *abi.DecodedEvent, scope Scope, managed ManagedAddresses, codec AddressEncoder) bool {
	if scope == ScopeWholeContract {
		return true
	}
	// ScopeManagedOnly: look for any address-typed input that is managed.
	for _, in := range ev.Inputs {
		if in.Type != "address" {
			continue
		}
		b, ok := in.Value.([]byte)
		if !ok {
			continue
		}
		addr, err := codec.EncodeBytesToAddress(b)
		if err != nil {
			continue
		}
		if managed.IsAddressKnown(addr) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./eventlog/ -run TestEventMatchesScope -v`
Expected: PASS (all three).

- [ ] **Step 5: Commit**

```bash
git add eventlog/event.go eventlog/event_test.go
git commit -m "feat(eventlog): ContractEvent + managed-address scope matching"
```

---

### Task 3: Config (mode selection)

**Files:**
- Create: `eventlog/config.go`
- Test: `eventlog/config_test.go`

- [ ] **Step 1: Write the failing test** — create `eventlog/config_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./eventlog/ -run 'TestParseMode|TestDefaultConfig' -v`
Expected: FAIL — `undefined: Mode`.

- [ ] **Step 3: Write minimal implementation** — create `eventlog/config.go`:

```go
package eventlog

import (
	"fmt"
	"strings"
)

// Mode selects how the service collects a block's logs.
type Mode int

const (
	// ModeGetLogs fetches a block's logs with a single eth_getLogs call,
	// filtered by the subscribed contract addresses. Independent of tx count.
	ModeGetLogs Mode = iota
	// ModeReceipts fetches each relevant transaction's receipt and reads its
	// logs. Precise per-tx attribution at the cost of N calls per block.
	ModeReceipts
)

// ParseMode parses a mode name (case-insensitive). Empty defaults to getLogs.
func ParseMode(s string) (Mode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "getlogs":
		return ModeGetLogs, nil
	case "receipts":
		return ModeReceipts, nil
	default:
		return 0, fmt.Errorf("unknown eventlog mode %q (want getLogs | receipts)", s)
	}
}

// Config configures the eventlog service.
type Config struct {
	// Mode selects the per-block log collection strategy.
	Mode Mode
	// ReceiptConcurrency bounds parallel receipt fetches in ModeReceipts.
	ReceiptConcurrency int
}

// DefaultConfig returns the default configuration: getLogs mode, 8-way receipt
// concurrency.
func DefaultConfig() Config {
	return Config{Mode: ModeGetLogs, ReceiptConcurrency: 8}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./eventlog/ -run 'TestParseMode|TestDefaultConfig' -v`
Expected: PASS (both).

- [ ] **Step 5: Commit**

```bash
git add eventlog/config.go eventlog/config_test.go
git commit -m "feat(eventlog): config with collection mode selection"
```

---

### Task 4: Service skeleton + LogSource + decode/deliver tail

**Files:**
- Create: `eventlog/eventlog.go`
- Test: `eventlog/eventlog_test.go`

- [ ] **Step 1: Write the failing test** — create `eventlog/eventlog_test.go`:

```go
package eventlog

import (
	"sync"
	"testing"

	"github.com/ITProLabDev/ethbacknode/abi"
)

// --- fakes ---------------------------------------------------------------

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

// fakeDecoder decodes a log into a fixed event keyed by the log's first topic.
type fakeDecoder struct{ events map[string]*abi.DecodedEvent }

func (d *fakeDecoder) DecodeLog(contract string, topics [][32]byte, data []byte) (*abi.DecodedEvent, error) {
	ev := d.events[contract]
	if ev == nil {
		return nil, errNoEvent
	}
	return ev, nil
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
	// The getLogs filter must be scoped to the subscribed address and block.
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
```

NOTE: this test references aliases (`ethLog`, `ethReceipt`, `logFilter`, `topicHex`, `errNoEvent`) that you DEFINE in Step 3 as part of the service's own interface so the package does not import `clients/ethclient` directly (avoiding an import cycle risk and keeping eventlog testable). See Step 3.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./eventlog/ -run TestService_OnBlock -v`
Expected: FAIL — `undefined: New` etc.

- [ ] **Step 3: Write minimal implementation** — create `eventlog/eventlog.go`:

```go
package eventlog

import (
	"errors"

	"github.com/ITProLabDev/ethbacknode/abi"
	"github.com/ITProLabDev/ethbacknode/tools/log"
)

// errNoEvent is returned by a decoder when a log does not match a known event.
var errNoEvent = errors.New("no matching event")

// ethLog is the minimal log shape eventlog needs. It mirrors
// clients/ethclient.Log; the adapter in main.go converts between them.
type ethLog struct {
	Address          string
	Topics           []string
	Data             []byte
	BlockNumber      int64
	TransactionHash  string
	LogIndex         int64
}

// topics32 converts the 0x-hex topics to [32]byte for the decoder.
func (l *ethLog) topics32() ([][32]byte, error) {
	out := make([][32]byte, len(l.Topics))
	for i, t := range l.Topics {
		b, err := parseTopic(t)
		if err != nil {
			return nil, err
		}
		out[i] = b
	}
	return out, nil
}

// ethReceipt is the minimal receipt shape eventlog needs.
type ethReceipt struct {
	Logs []*ethLog
}

// logFilter mirrors clients/ethclient.LogFilter for the LogSource interface.
type logFilter struct {
	FromBlock int64
	ToBlock   int64
	Addresses []string
	Topics    []string
}

// LogSource is the narrow chain-access surface eventlog needs. *ethclient.Client
// satisfies an adapter of this (wired in main.go); tests use a fake.
type LogSource interface {
	GetLogs(filter logFilter) ([]*ethLog, error)
	GetTransactionReceipt(txHash string) (*ethReceipt, error)
}

// Decoder decodes a single log into a DecodedEvent. Satisfied by an adapter
// over *abi.SmartContractsManager.DecodeLog (and by a fake in tests).
type Decoder interface {
	DecodeLog(contractAddress string, topics [][32]byte, data []byte) (*abi.DecodedEvent, error)
}

// Sink receives decoded, scope-filtered contract events for delivery (M6).
type Sink func(*ContractEvent)

// Service collects, decodes, and scope-filters contract event logs per block.
type Service struct {
	source  LogSource
	decoder Decoder
	managed ManagedAddresses
	codec   AddressEncoder
	sink    Sink
	cfg     Config
	subs    *subscriptionSet
}

// Option configures a Service.
type Option func(*Service)

func WithLogSource(s LogSource) Option         { return func(svc *Service) { svc.source = s } }
func WithDecoder(d Decoder) Option             { return func(svc *Service) { svc.decoder = d } }
func WithManaged(m ManagedAddresses) Option    { return func(svc *Service) { svc.managed = m } }
func WithAddressCodec(c AddressEncoder) Option { return func(svc *Service) { svc.codec = c } }
func WithSink(s Sink) Option                   { return func(svc *Service) { svc.sink = s } }
func WithConfig(c Config) Option               { return func(svc *Service) { svc.cfg = c } }

// New builds a Service from options.
func New(opts ...Option) *Service {
	svc := &Service{cfg: DefaultConfig(), subs: newSubscriptionSet()}
	for _, o := range opts {
		o(svc)
	}
	return svc
}

// Subscribe registers a contract-event subscription.
func (s *Service) Subscribe(sub *Subscription) { s.subs.add(sub) }

// OnBlock is the watchdog block-listener entry point: collect the block's logs,
// decode them, and deliver scope-matched events to the sink.
func (s *Service) OnBlock(blockNum int64, blockID string) {
	addrs := s.subs.addresses()
	if len(addrs) == 0 {
		return // nothing subscribed; do no chain work
	}
	logs, err := s.collect(blockNum, addrs)
	if err != nil {
		log.Error("eventlog: collect block", blockNum, "error:", err)
		return
	}
	for _, lg := range logs {
		s.decodeAndDeliver(lg)
	}
}

// decodeAndDeliver decodes one log and delivers it to every matching
// subscription for its contract address.
func (s *Service) decodeAndDeliver(lg *ethLog) {
	subs := s.subs.forAddress(lg.Address)
	if len(subs) == 0 {
		return
	}
	topics, err := lg.topics32()
	if err != nil {
		return // malformed topic; skip
	}
	ev, err := s.decoder.DecodeLog(lg.Address, topics, lg.Data)
	if err != nil {
		return // unknown event / undecodable; skip
	}
	delivered := false
	for _, sub := range subs {
		if delivered && sub.Scope == ScopeWholeContract {
			// already delivered an identical event for whole_contract; still
			// deliver per-subscription so each subscriber gets its own copy.
		}
		if eventMatchesScope(ev, sub.Scope, s.managed, s.codec) {
			s.sink(&ContractEvent{
				Event:           ev,
				BlockNumber:     lg.BlockNumber,
				TransactionHash: lg.TransactionHash,
				LogIndex:        lg.LogIndex,
			})
			delivered = true
		}
	}
}
```

Also create `eventlog/collect.go` with the Mode A collector and the `parseTopic` helper (Mode B is added in Task 5):

```go
package eventlog

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// collect gathers the block's logs according to the configured mode.
func (s *Service) collect(blockNum int64, addrs []string) ([]*ethLog, error) {
	switch s.cfg.Mode {
	case ModeReceipts:
		return s.collectModeReceipts(blockNum, addrs)
	default:
		return s.collectModeGetLogs(blockNum, addrs)
	}
}

// collectModeGetLogs fetches the block's logs with one eth_getLogs call,
// filtered to the subscribed contract addresses.
func (s *Service) collectModeGetLogs(blockNum int64, addrs []string) ([]*ethLog, error) {
	return s.source.GetLogs(logFilter{
		FromBlock: blockNum,
		ToBlock:   blockNum,
		Addresses: addrs,
	})
}

// parseTopic decodes a 0x-hex 32-byte topic string.
func parseTopic(s string) ([32]byte, error) {
	var out [32]byte
	h := strings.TrimPrefix(s, "0x")
	b, err := hex.DecodeString(h)
	if err != nil {
		return out, err
	}
	if len(b) != 32 {
		return out, fmt.Errorf("topic is %d bytes, want 32", len(b))
	}
	copy(out[:], b)
	return out, nil
}
```

`topicHex` test helper — add it to `eventlog_test.go`:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./eventlog/ -run TestService_OnBlock -v`
Expected: PASS (both).

- [ ] **Step 5: Commit**

```bash
git add eventlog/eventlog.go eventlog/collect.go eventlog/eventlog_test.go
git commit -m "feat(eventlog): service skeleton, LogSource, getLogs collection + decode/deliver"
```

---

### Task 5: Mode B — per-receipt collection with bounded parallel fetch

**Files:**
- Modify: `eventlog/collect.go`
- Test: `eventlog/collect_test.go`

- [ ] **Step 1: Write the failing test** — create `eventlog/collect_test.go`:

```go
package eventlog

import (
	"sort"
	"testing"
)

// recProvider serves receipts by tx hash and a block's tx-hash list.
type recProvider struct {
	blockTxs map[int64][]string
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
	// Inject the block's tx hashes via the test seam.
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
	// Order-independent: collect the log indices and check the set.
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./eventlog/ -run TestCollectModeReceipts -v`
Expected: FAIL — `svc.blockTxHashes` undefined / `collectModeReceipts` undefined.

- [ ] **Step 3: Add the `blockTxHashes` seam to the Service** — in `eventlog/eventlog.go`, add the field to the `Service` struct (after `subs`):

```go
	// blockTxHashes returns the tx hashes in a block (Mode B). Wired in main.go
	// from the chain client; a test seam in unit tests.
	blockTxHashes func(blockNum int64) ([]string, error)
```

- [ ] **Step 4: Implement `collectModeReceipts`** — append to `eventlog/collect.go` (and update its import block to add `context` and `tools/flow` — the full block is shown after the function):

```go
// collectModeReceipts fetches each of the block's transaction receipts (bounded
// parallel) and returns the logs emitted by the subscribed contract addresses.
func (s *Service) collectModeReceipts(blockNum int64, addrs []string) ([]*ethLog, error) {
	if s.blockTxHashes == nil {
		return nil, fmt.Errorf("eventlog: receipts mode requires blockTxHashes")
	}
	hashes, err := s.blockTxHashes(blockNum)
	if err != nil {
		return nil, err
	}
	if len(hashes) == 0 {
		return nil, nil
	}

	wanted := make(map[string]bool, len(addrs))
	for _, a := range addrs {
		wanted[strings.ToLower(a)] = true
	}

	receipts, err := flow.FanOut(context.Background(), hashes, s.cfg.ReceiptConcurrency,
		func(_ context.Context, txHash string) (*ethReceipt, error) {
			return s.source.GetTransactionReceipt(txHash)
		})
	if err != nil {
		return nil, err
	}

	var out []*ethLog
	for _, r := range receipts {
		if r == nil {
			continue
		}
		for _, lg := range r.Logs {
			if wanted[strings.ToLower(lg.Address)] {
				out = append(out, lg)
			}
		}
	}
	return out, nil
}
```

The `collect.go` import block becomes:

```go
import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ITProLabDev/ethbacknode/tools/flow"
)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./eventlog/ -run TestCollectModeReceipts -v`
Expected: PASS (both). Note: `flow.FanOut` preserves order, but the address filter and the test assert order-independently, so either way is fine.

- [ ] **Step 6: Run the whole eventlog package**

Run: `go test ./eventlog/ -count=1`
Expected: `ok`.

- [ ] **Step 7: Commit**

```bash
git add eventlog/collect.go eventlog/eventlog.go eventlog/collect_test.go
git commit -m "feat(eventlog): receipts mode with bounded parallel receipt fetch"
```

---

### Task 6: Wire eventlog into main.go (watchdog block listener + adapters)

**Files:**
- Modify: `main.go`
- Create: `eventlog/adapter.go`
- Test: `eventlog/adapter_test.go`

The service uses its own `ethLog`/`ethReceipt`/`logFilter`/`LogSource`/`Decoder` types so it does not import `clients/ethclient` (avoiding a heavy dependency and keeping it unit-testable). `main.go` wires real adapters that convert between `clients/ethclient` types and eventlog's. This task provides the adapter constructors and tests them, then registers `OnBlock` as a watchdog block listener.

- [ ] **Step 1: Write the failing test** — create `eventlog/adapter_test.go`:

```go
package eventlog

import (
	"testing"

	"github.com/ITProLabDev/ethbacknode/abi"
)

// chainStub stands in for *ethclient.Client at the adapter boundary, using the
// real ethclient-shaped types via small local mirrors. We assert the adapter
// converts filter/logs faithfully.

type ethclientLogMirror struct {
	Address         string
	Topics          []string
	Data            []byte
	BlockNumber     int64
	TransactionHash string
	LogIndex        int64
}

func TestDecoderAdapter_DelegatesToManager(t *testing.T) {
	// managerDecodeFunc mirrors abi.(*SmartContractsManager).DecodeLog.
	called := false
	adapter := decoderFunc(func(contract string, topics [][32]byte, data []byte) (*abi.DecodedEvent, error) {
		called = true
		return &abi.DecodedEvent{Name: "E", Contract: contract}, nil
	})
	ev, err := adapter.DecodeLog("0xabc", [][32]byte{{0x01}}, nil)
	if err != nil || !called || ev.Name != "E" {
		t.Fatalf("decoder adapter failed: ev=%v err=%v called=%v", ev, err, called)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./eventlog/ -run TestDecoderAdapter -v`
Expected: FAIL — `undefined: decoderFunc`.

- [ ] **Step 3: Add a `decoderFunc` adapter** — create `eventlog/adapter.go`:

```go
package eventlog

import "github.com/ITProLabDev/ethbacknode/abi"

// decoderFunc adapts a plain function to the Decoder interface, so main.go can
// wire abi.(*SmartContractsManager).DecodeLog without a named wrapper type.
type decoderFunc func(contractAddress string, topics [][32]byte, data []byte) (*abi.DecodedEvent, error)

func (f decoderFunc) DecodeLog(contractAddress string, topics [][32]byte, data []byte) (*abi.DecodedEvent, error) {
	return f(contractAddress, topics, data)
}

// NewDecoder wraps any DecodeLog-shaped function (e.g. the abi manager's
// DecodeLog method) as a Decoder.
func NewDecoder(fn func(contractAddress string, topics [][32]byte, data []byte) (*abi.DecodedEvent, error)) Decoder {
	return decoderFunc(fn)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./eventlog/ -run TestDecoderAdapter -v`
Expected: PASS.

- [ ] **Step 5: Add a LogSource adapter constructor** — append to `eventlog/adapter.go`:

```go
// RawLogSource is the ethclient-shaped surface the LogSource adapter needs.
// *ethclient.Client satisfies it (GetLogs/GetTransactionReceipt from M4). Using
// 'any'-free function fields keeps eventlog free of an ethclient import.
type RawLogSource struct {
	GetLogsFn    func(fromBlock, toBlock int64, addresses, topics []string) ([]RawLog, error)
	GetReceiptFn func(txHash string) (logs []RawLog, err error)
}

// RawLog is the ethclient-shaped log the adapter receives.
type RawLog struct {
	Address         string
	Topics          []string
	Data            []byte
	BlockNumber     int64
	TransactionHash string
	LogIndex        int64
}

func (r RawLogSource) GetLogs(filter logFilter) ([]*ethLog, error) {
	raw, err := r.GetLogsFn(filter.FromBlock, filter.ToBlock, filter.Addresses, filter.Topics)
	if err != nil {
		return nil, err
	}
	return rawToEthLogs(raw), nil
}

func (r RawLogSource) GetTransactionReceipt(txHash string) (*ethReceipt, error) {
	raw, err := r.GetReceiptFn(txHash)
	if err != nil {
		return nil, err
	}
	return &ethReceipt{Logs: rawToEthLogs(raw)}, nil
}

func rawToEthLogs(raw []RawLog) []*ethLog {
	out := make([]*ethLog, len(raw))
	for i, l := range raw {
		out[i] = &ethLog{
			Address:         l.Address,
			Topics:          l.Topics,
			Data:            l.Data,
			BlockNumber:     l.BlockNumber,
			TransactionHash: l.TransactionHash,
			LogIndex:        l.LogIndex,
		}
	}
	return out
}
```

- [ ] **Step 6: Wire it in `main.go`** — after the subscriptions manager is created and before `watchdogService.Run()`, add the eventlog construction and registration. Insert near the other `RegisterBlockEventListen` calls (the existing lines registering `subscriptionsManager.BlockEvent` / `txCacheManager.BlockEvent`):

```go
	// Event-log service: decode registered contracts' events per block and
	// deliver them. Wired as a watchdog block listener (Approach A).
	eventLogService := eventlog.New(
		eventlog.WithLogSource(eventlog.RawLogSource{
			GetLogsFn: func(fromBlock, toBlock int64, addresses, topics []string) ([]eventlog.RawLog, error) {
				logs, err := chainClient.GetLogs(ethclient.LogFilter{
					FromBlock: fromBlock, ToBlock: toBlock, Addresses: addresses, Topics: topics,
				})
				if err != nil {
					return nil, err
				}
				out := make([]eventlog.RawLog, len(logs))
				for i, l := range logs {
					out[i] = eventlog.RawLog{
						Address: l.Address, Topics: l.Topics, Data: l.Data,
						BlockNumber: l.BlockNumber, TransactionHash: l.TransactionHash, LogIndex: l.LogIndex,
					}
				}
				return out, nil
			},
			GetReceiptFn: func(txHash string) ([]eventlog.RawLog, error) {
				r, err := chainClient.GetTransactionReceipt(txHash)
				if err != nil {
					return nil, err
				}
				out := make([]eventlog.RawLog, len(r.Logs))
				for i, l := range r.Logs {
					out[i] = eventlog.RawLog{
						Address: l.Address, Topics: l.Topics, Data: l.Data,
						BlockNumber: l.BlockNumber, TransactionHash: l.TransactionHash, LogIndex: l.LogIndex,
					}
				}
				return out, nil
			},
		}),
		eventlog.WithDecoder(eventlog.NewDecoder(abiManager.DecodeLog)),
		eventlog.WithManaged(addressManager),
		eventlog.WithAddressCodec(addressCodec),
		eventlog.WithSink(func(ce *eventlog.ContractEvent) {
			log.Info("contractEvent:", ce.Event.Name, "contract:", ce.Event.Contract, "block:", ce.BlockNumber, "tx:", ce.TransactionHash)
		}),
		eventlog.WithConfig(eventlog.DefaultConfig()),
	)
	watchdogService.RegisterBlockEventListen(func(blockNum int64, blockId string) {
		eventLogService.OnBlock(blockNum, blockId)
	})
	_ = eventLogService // referenced by the listener closure
```

Add `"github.com/ITProLabDev/ethbacknode/eventlog"` to the `main.go` import block. `addressCodec` is already defined in `main.go` (`addressCodec := ethclient.GetAddressCodec()`); `addressManager`, `abiManager`, `chainClient`, `watchdogService` are all already in scope at that point.

- [ ] **Step 7: Verify the whole thing builds and the eventlog tests pass**

Run: `go build ./... && go test ./eventlog/ -count=1`
Expected: build exits 0; eventlog tests `ok`.

NOTE: `WithManaged`/`WithAddressCodec` take EXPORTED interfaces (`ManagedAddresses`/`AddressEncoder`), so `main.go` wires `addressManager` (has `IsAddressKnown(string) bool`) and `addressCodec` (has `EncodeBytesToAddress([]byte) (string, error)`) across packages without friction.

- [ ] **Step 8: Commit**

```bash
git add main.go eventlog/adapter.go eventlog/adapter_test.go
git commit -m "feat(eventlog): wire into watchdog as block listener via adapters"
```

---

### Task 7: Final verification & docs

**Files:**
- Modify: `todo/TASKS.md`

- [ ] **Step 1: Run the eventlog package with the race detector**

Run: `go test ./eventlog/ -race -count=1`
Expected: `ok` — no failures, no data races.

- [ ] **Step 2: Confirm nothing else broke**

Run: `go build ./... && go vet ./eventlog/ .`
Expected: both exit 0.

- [ ] **Step 3: Mark M5 tasks done in `todo/TASKS.md`**

Change each of `M5.1`–`M5.7` from `- [ ]` to `- [x]` in the "Milestone 5" section, append " ✅ DONE" to the header line, and add a status block (mirror M1–M4 style). Mapping:
- M5.1 `eventlog/` package (Approach A, abi pure) → Tasks 1–4
- M5.2 Mode A getLogs per block → Task 4
- M5.3 Mode B per-receipt via `flow.FanOut` → Task 5
- M5.4 config mode switch (default getLogs) → Task 3
- M5.5 watchdog integration as block listener → Task 6
- M5.6 match managed addresses inside indexed params → Task 2
- M5.7 per-subscription scope (whole_contract | managed_only) → Tasks 1, 2, 4

- [ ] **Step 4: Commit**

```bash
git add todo/TASKS.md
git commit -m "docs(eventlog): mark M5 (event-log service) complete"
```

---

## Notes for the implementer

- **Do NOT change `watchdog/` internals.** M5 plugs in solely via the existing `RegisterBlockEventListen`. The block listener calls `eventLogService.OnBlock`.
- **`eventlog` must NOT import `clients/ethclient`.** It defines its own minimal `ethLog`/`ethReceipt`/`logFilter` and `LogSource`/`Decoder` interfaces; `main.go` wires adapters (`RawLogSource`, `NewDecoder`) that translate from the real ethclient/abi types. This keeps eventlog decoupled and unit-testable with fakes.
- **`abi/` stays a pure library** — eventlog depends on `abi.SmartContractsManager.DecodeLog` and `abi.DecodedEvent` only, via the `Decoder` interface. No service code goes into `abi/`.
- **Scope vs mode are orthogonal:** Mode (getLogs|receipts) is how logs are collected; Scope (whole_contract|managed_only) is how decoded events are filtered per subscription. Both must work in any combination.
- **managed_only matching** treats an address parameter as `DecodedValue.Type == "address"` with `Value` a 20-byte `[]byte`; encode via the codec and check `IsAddressKnown`. Indexed addresses decode to the same shape (M2), so they are matched too.
- The `With...` option interfaces (`ManagedAddresses`, `AddressEncoder`) are EXPORTED so `main.go` wires `addressManager`/`addressCodec` frictionlessly across packages.
- M5 ends at the `Sink`. M6 replaces the logging sink with real JSON-RPC `contractEvent` delivery and adds the RPC to register contracts/subscriptions.
- Run only `./eventlog/` (and `go build ./...`); the repo has a pre-existing unrelated compile error in `crypto/secp256k1`.
