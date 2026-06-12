# M2 — ABI Engine: Event-Log Decoding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Decode Ethereum event logs (topics + data) with the project's own ABI engine, splitting `indexed` parameters (from `topics[1:]`) from non-indexed parameters (from `data`), and provide a topic0→event registry lookup.

**Architecture:** Build on the typed M1 codec. Add a full 32-byte `Topic0()` to `SmartContractAbiEntry` (keccak of the canonical signature — events use the full hash, unlike the 4-byte method selector). Add `DecodeLog(topics, data)` on `SmartContractAbi` that finds the matching event by `topics[1:]` semantics and decodes each part. Add `GetEventByTopic0` to the registry. Indexed dynamic params are non-recoverable (only their keccak hash is in the topic) — represent them as a hash placeholder, never feed them to `decodeParams`.

**Tech Stack:** Go 1.24, `math/big`, existing `crypto.Keccak256`, the M1 typed codec (`abi/codec.go`, `abi/abitype.go`, `abi/value.go`), standard `testing`.

---

## Background & Constraints

**Hard rule (carried from M1):** the `abi/` package stays a self-built engine — NO go-ethereum or external ABI library.

**How Ethereum event logs are structured (the spec this implements):**
- A log has `topics [][32]byte` and `data []byte`.
- For a non-anonymous event, `topics[0]` = `keccak256("EventName(canonicalType1,...)")` — the **full 32 bytes**, not a 4-byte selector.
- Each `indexed` parameter consumes one topic slot, in order, from `topics[1:]`.
- Each non-indexed parameter is ABI-encoded together in `data` (as a head/tail sequence — exactly what `decodeParams` already does).
- **Indexed value types:** a static indexed param (uint, int, bool, address, bytesN) is stored directly, left/right-aligned in its 32-byte topic, and is fully decodable. A **dynamic** indexed param (string, bytes, array, tuple) is stored as `keccak256(value)` — the original value is NOT recoverable; only the hash is present.
- `anonymous` events have NO `topics[0]` (all indexed params start at `topics[0]`). EthBackNode's ABI model has no `anonymous` field today; we treat all registered events as non-anonymous and add an explicit guard.

**What already exists (M1, committed):**
- `SmartContractAbiEntry` with `Inputs []*SmartContractAbiEntryInput` (each has `Name`, `Type`, `Indexed bool`, `Components`), `Type string` ("Event" / "Function"), `Signature [4]byte`, `canonicalSignature()`, `typedInputs()`.
- `crypto.Keccak256(data ...[]byte) []byte` → 32 bytes.
- `decodeParams(types []abiType, block []byte) ([]DecodedValue, error)` and `decodeValue(t abiType, block []byte, at int) (DecodedValue, error)` in `abi/codec.go`.
- `parseType(typeStr, components) (abiType, error)`, `abiType.isDynamic()`, `abiType.canonical()` in `abi/abitype.go`.
- `DecodedValue{Name, Type string; Value any}` and `DecodedEvent{Name, Contract string; Inputs []DecodedValue}` in `abi/value.go`.
- Registry `SmartContractsManager` (`abi/manager.go`) with `byAddress` map and `GetSmartContractByAddress`.
- Errors in `abi/errors.go`: `ErrInvalidParamsData`, `ErrSmartContractUnknownMethod`, `ErrUnknownContract`, etc.

**Decoded-value convention (from M1, reused):** `Value` is one of `*big.Int` (uint/int), `bool`, `[]byte` (address 20 bytes; bytesN N bytes), `string`, `[]DecodedValue` (array/slice/tuple). For an indexed dynamic param we add the convention: `Value` is a `[]byte` of length 32 (the keccak hash) and the `DecodedValue.Type` is suffixed with " (indexed)" — see Task 3.

---

## File Structure

- **Create `abi/event.go`** — event decoding: `Topic0()`, `eventSignature` helper if needed, `DecodeLog` on `*SmartContractAbiEntry` and on `*SmartContractAbi`, indexed/non-indexed split.
- **Create `abi/event_test.go`** — topic0 hashing tests, ERC-20 `Transfer` and ERC-1155 `TransferSingle`/`TransferBatch` decode tests, indexed-dynamic placeholder test, anonymous/guard tests.
- **Modify `abi/manager.go`** — add `GetEventByTopic0(topic0 [32]byte)` registry lookup (searches a contract's entries; or across all contracts).
- **Modify `abi/errors.go`** — add `ErrNotAnEvent`, `ErrUnknownEvent`, `ErrTopicCountMismatch`.

No changes to the M1 codec, the legacy decoder, or `erc20.go`.

---

### Task 1: Event topic0 hashing (`Topic0`)

**Files:**
- Create: `abi/event.go`
- Modify: `abi/errors.go`
- Test: `abi/event_test.go`

- [ ] **Step 1: Add error values** to `abi/errors.go` — insert inside the existing `var ( ... )` block, after `ErrNotTransferMethod`:

```go
	// ErrNotAnEvent is returned when a non-event entry is used for log decoding.
	ErrNotAnEvent = errors.New("abi entry is not an event")
	// ErrUnknownEvent is returned when no event matches a log's topic0.
	ErrUnknownEvent = errors.New("unknown event")
	// ErrTopicCountMismatch is returned when a log's topic count does not match
	// the event's indexed-parameter count.
	ErrTopicCountMismatch = errors.New("log topic count mismatch")
```

- [ ] **Step 2: Write the failing test** — create `abi/event_test.go`:

```go
package abi

import (
	"encoding/hex"
	"strings"
	"testing"
)

func mustHex32(t *testing.T, s string) [32]byte {
	t.Helper()
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil || len(b) != 32 {
		t.Fatalf("bad 32-byte hex %q: %v", s, err)
	}
	var out [32]byte
	copy(out[:], b)
	return out
}

func TestTopic0_Erc20Transfer(t *testing.T) {
	// Transfer(address,address,uint256) — the canonical ERC-20 Transfer topic0.
	e := &SmartContractAbiEntry{
		Name: "Transfer",
		Type: "Event",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "from", Type: "address", Indexed: true},
			{Name: "to", Type: "address", Indexed: true},
			{Name: "value", Type: "uint256"},
		},
	}
	got := e.Topic0()
	want := mustHex32(t, "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	if got != want {
		t.Fatalf("topic0=%x want %x", got, want)
	}
}

func TestTopic0_Erc1155TransferSingle(t *testing.T) {
	// TransferSingle(address,address,address,uint256,uint256)
	e := &SmartContractAbiEntry{
		Name: "TransferSingle",
		Type: "Event",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "operator", Type: "address", Indexed: true},
			{Name: "from", Type: "address", Indexed: true},
			{Name: "to", Type: "address", Indexed: true},
			{Name: "id", Type: "uint256"},
			{Name: "value", Type: "uint256"},
		},
	}
	got := e.Topic0()
	want := mustHex32(t, "c3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62")
	if got != want {
		t.Fatalf("topic0=%x want %x", got, want)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./abi/ -run TestTopic0 -v`
Expected: FAIL — `e.Topic0 undefined`.

- [ ] **Step 4: Write minimal implementation** — create `abi/event.go`:

```go
package abi

import "github.com/ITProLabDev/ethbacknode/crypto"

// Topic0 returns the 32-byte event signature hash
// keccak256("EventName(type1,type2,...)") used as topics[0] for a
// non-anonymous event. Unlike a method selector (first 4 bytes), an event
// topic uses the full 32-byte hash.
func (e *SmartContractAbiEntry) Topic0() [32]byte {
	h := crypto.Keccak256([]byte(e.canonicalSignature()))
	var out [32]byte
	copy(out[:], h)
	return out
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./abi/ -run TestTopic0 -v`
Expected: PASS (both vectors match the real on-chain topic0 values).

- [ ] **Step 6: Commit**

```bash
git add abi/event.go abi/errors.go abi/event_test.go
git commit -m "feat(abi): event topic0 hashing"
```

---

### Task 2: Decode a log against a known event entry (`(*SmartContractAbiEntry).DecodeLog`)

**Files:**
- Modify: `abi/event.go`
- Test: `abi/event_test.go` (append)

- [ ] **Step 1a: Extend the import block** of `abi/event_test.go` to add `bytes` and `math/big` (the new tests use them). Replace:

```go
import (
	"encoding/hex"
	"strings"
	"testing"
)
```

with:

```go
import (
	"bytes"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"
)
```

- [ ] **Step 1b: Write the failing test** — append to `abi/event_test.go`:

```go
func TestEntryDecodeLog_Erc20Transfer(t *testing.T) {
	e := &SmartContractAbiEntry{
		Name: "Transfer",
		Type: "Event",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "from", Type: "address", Indexed: true},
			{Name: "to", Type: "address", Indexed: true},
			{Name: "value", Type: "uint256"},
		},
	}
	from := mustHex32(t, "000000000000000000000000"+"1111111111111111111111111111111111111111")
	to := mustHex32(t, "000000000000000000000000"+"2222222222222222222222222222222222222222")
	topics := [][32]byte{e.Topic0(), from, to}
	// data = the non-indexed uint256 value = 1000
	data := make([]byte, 32)
	big.NewInt(1000).FillBytes(data)

	ev, err := e.DecodeLog(topics, data)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Name != "Transfer" || len(ev.Inputs) != 3 {
		t.Fatalf("event=%+v", ev)
	}
	// indexed addresses decoded to 20-byte []byte
	wantFrom, _ := hex.DecodeString("1111111111111111111111111111111111111111")
	if !bytes.Equal(ev.Inputs[0].Value.([]byte), wantFrom) {
		t.Fatalf("from=%x", ev.Inputs[0].Value)
	}
	if ev.Inputs[0].Name != "from" {
		t.Fatalf("name=%q", ev.Inputs[0].Name)
	}
	// non-indexed value decoded from data
	if ev.Inputs[2].Value.(*big.Int).Int64() != 1000 {
		t.Fatalf("value=%v", ev.Inputs[2].Value)
	}
}

func TestEntryDecodeLog_RejectsNonEvent(t *testing.T) {
	e := &SmartContractAbiEntry{Name: "transfer", Type: "Function"}
	if _, err := e.DecodeLog(nil, nil); err == nil {
		t.Fatal("non-event must be rejected")
	}
}

func TestEntryDecodeLog_RejectsTopicCountMismatch(t *testing.T) {
	e := &SmartContractAbiEntry{
		Name: "Transfer",
		Type: "Event",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "from", Type: "address", Indexed: true},
			{Name: "to", Type: "address", Indexed: true},
			{Name: "value", Type: "uint256"},
		},
	}
	// only 1 topic past topic0, but 2 indexed params expected
	topics := [][32]byte{e.Topic0(), mustHex32(t, strings.Repeat("00", 32))}
	if _, err := e.DecodeLog(topics, nil); err == nil {
		t.Fatal("topic count mismatch must be rejected")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./abi/ -run TestEntryDecodeLog -v`
Expected: FAIL — `e.DecodeLog undefined`.

- [ ] **Step 3: Write minimal implementation** — append to `abi/event.go`:

```go
// DecodeLog decodes an event log (topics + data) into a DecodedEvent using the
// typed engine. topics[0] is the event signature; topics[1:] carry the indexed
// parameters in order; data carries the ABI-encoded non-indexed parameters.
//
// A static indexed parameter is decoded from its 32-byte topic slot. A dynamic
// indexed parameter (string, bytes, array, tuple) is NOT recoverable from the
// log — only keccak256(value) is stored in the topic — so it is returned as a
// 32-byte hash placeholder with its Type suffixed " (indexed)".
func (e *SmartContractAbiEntry) DecodeLog(topics [][32]byte, data []byte) (*DecodedEvent, error) {
	if !strings.EqualFold(e.Type, "Event") {
		return nil, ErrNotAnEvent
	}

	// Count indexed params and verify the topic count.
	indexedCount := 0
	for _, in := range e.Inputs {
		if in.Indexed {
			indexedCount++
		}
	}
	if len(topics) != indexedCount+1 { // +1 for topics[0] (the signature)
		return nil, ErrTopicCountMismatch
	}

	// Split inputs into indexed (decoded from topics) and non-indexed (decoded
	// from data, preserving order).
	nonIndexedTypes := make([]abiType, 0, len(e.Inputs))
	nonIndexedIdx := make([]int, 0, len(e.Inputs)) // maps data-decode position → input index
	for i, in := range e.Inputs {
		if in.Indexed {
			continue
		}
		typ, err := parseType(in.Type, in.Components)
		if err != nil {
			return nil, err
		}
		nonIndexedTypes = append(nonIndexedTypes, typ)
		nonIndexedIdx = append(nonIndexedIdx, i)
	}

	dataVals, err := decodeParams(nonIndexedTypes, data)
	if err != nil {
		return nil, err
	}

	out := &DecodedEvent{Name: e.Name, Inputs: make([]DecodedValue, len(e.Inputs))}
	topicPos := 1     // topics[0] is the signature
	dataPos := 0
	for i, in := range e.Inputs {
		if in.Indexed {
			dv, derr := decodeIndexedTopic(in, topics[topicPos])
			if derr != nil {
				return nil, derr
			}
			dv.Name = in.Name
			out.Inputs[i] = dv
			topicPos++
		} else {
			dv := dataVals[dataPos]
			dv.Name = in.Name
			out.Inputs[i] = dv
			dataPos++
		}
	}
	return out, nil
}

// decodeIndexedTopic decodes a single indexed parameter from its 32-byte topic.
// Value types are decoded normally; reference types (arrays — fixed or dynamic —
// tuples/structs, string, bytes) are stored as keccak256(value) in the topic per
// the Solidity ABI spec and are returned as a 32-byte hash placeholder.
func decodeIndexedTopic(in *SmartContractAbiEntryInput, topic [32]byte) (DecodedValue, error) {
	typ, err := parseType(in.Type, in.Components)
	if err != nil {
		return DecodedValue{}, err
	}
	// NOTE: gating on isDynamic() alone is WRONG — a static array (uint256[2])
	// or static tuple is still hashed when indexed. Gate on the kind instead.
	switch typ.kind {
	case kindArray, kindSlice, kindTuple, kindString, kindBytes:
		hash := make([]byte, 32)
		copy(hash, topic[:])
		return DecodedValue{Type: typ.canonical() + " (indexed)", Value: hash}, nil
	}
	return decodeValue(typ, topic[:], 0)
}
```

Add `"strings"` to the `abi/event.go` import block (it currently imports only `crypto`). The block becomes:

```go
import (
	"strings"

	"github.com/ITProLabDev/ethbacknode/crypto"
)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./abi/ -run TestEntryDecodeLog -v`
Expected: PASS (all three).

- [ ] **Step 5: Run the full abi suite (no regressions)**

Run: `go test ./abi/ -count=1`
Expected: `ok` (ignore `[CRIT] Duplicated Contract` log lines from a pre-existing concurrency test).

- [ ] **Step 6: Commit**

```bash
git add abi/event.go abi/event_test.go
git commit -m "feat(abi): decode event logs (indexed topics + data)"
```

---

### Task 3: Indexed-dynamic placeholder + ERC-1155 TransferBatch decode

**Files:**
- Test: `abi/event_test.go` (append)

This task adds no production code — it locks in two behaviors the engine must already satisfy after Task 2: (a) an indexed dynamic param yields a 32-byte hash placeholder; (b) an event with non-indexed dynamic arrays (ERC-1155 `TransferBatch`) decodes correctly from `data`. If either test fails, fix the Task 2 code.

- [ ] **Step 1: Write the tests** — append to `abi/event_test.go`:

```go
func TestEntryDecodeLog_IndexedDynamicPlaceholder(t *testing.T) {
	// An indexed string is stored as keccak256(value); decoding yields the
	// 32-byte hash placeholder, not the original string.
	e := &SmartContractAbiEntry{
		Name: "Named",
		Type: "Event",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "key", Type: "string", Indexed: true},
			{Name: "n", Type: "uint256"},
		},
	}
	keyHash := mustHex32(t, "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899")
	topics := [][32]byte{e.Topic0(), keyHash}
	data := make([]byte, 32)
	big.NewInt(7).FillBytes(data)

	ev, err := e.DecodeLog(topics, data)
	if err != nil {
		t.Fatal(err)
	}
	hash, ok := ev.Inputs[0].Value.([]byte)
	if !ok || len(hash) != 32 {
		t.Fatalf("indexed string must be a 32-byte hash placeholder, got %T", ev.Inputs[0].Value)
	}
	if !bytes.Equal(hash, keyHash[:]) {
		t.Fatalf("placeholder=%x want %x", hash, keyHash[:])
	}
	if !strings.Contains(ev.Inputs[0].Type, "indexed") {
		t.Fatalf("type should mark indexed: %q", ev.Inputs[0].Type)
	}
	if ev.Inputs[1].Value.(*big.Int).Int64() != 7 {
		t.Fatalf("n=%v", ev.Inputs[1].Value)
	}
}

func TestEntryDecodeLog_Erc1155TransferBatch(t *testing.T) {
	// TransferBatch(address operator, address from, address to,
	//               uint256[] ids, uint256[] values)
	// operator/from/to indexed; ids+values are non-indexed dynamic arrays in data.
	e := &SmartContractAbiEntry{
		Name: "TransferBatch",
		Type: "Event",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "operator", Type: "address", Indexed: true},
			{Name: "from", Type: "address", Indexed: true},
			{Name: "to", Type: "address", Indexed: true},
			{Name: "ids", Type: "uint256[]"},
			{Name: "values", Type: "uint256[]"},
		},
	}
	op := mustHex32(t, "000000000000000000000000"+"1111111111111111111111111111111111111111")
	from := mustHex32(t, "000000000000000000000000"+"2222222222222222222222222222222222222222")
	to := mustHex32(t, "000000000000000000000000"+"3333333333333333333333333333333333333333")
	topics := [][32]byte{e.Topic0(), op, from, to}

	// Build data = abi.encode(uint256[]{10,11}, uint256[]{20,21}) using the
	// engine itself (round-trip is the check).
	idsType, _ := parseType("uint256[]", nil)
	valsType, _ := parseType("uint256[]", nil)
	data, err := encodeParams([]abiType{idsType, valsType}, []any{
		[]any{big.NewInt(10), big.NewInt(11)},
		[]any{big.NewInt(20), big.NewInt(21)},
	})
	if err != nil {
		t.Fatal(err)
	}

	ev, err := e.DecodeLog(topics, data)
	if err != nil {
		t.Fatal(err)
	}
	if len(ev.Inputs) != 5 {
		t.Fatalf("inputs=%d", len(ev.Inputs))
	}
	ids := ev.Inputs[3].Value.([]DecodedValue)
	vals := ev.Inputs[4].Value.([]DecodedValue)
	if len(ids) != 2 || ids[0].Value.(*big.Int).Int64() != 10 || ids[1].Value.(*big.Int).Int64() != 11 {
		t.Fatalf("ids=%+v", ids)
	}
	if len(vals) != 2 || vals[0].Value.(*big.Int).Int64() != 20 || vals[1].Value.(*big.Int).Int64() != 21 {
		t.Fatalf("values=%+v", vals)
	}
	// indexed 'to' decoded correctly
	wantTo, _ := hex.DecodeString("3333333333333333333333333333333333333333")
	if !bytes.Equal(ev.Inputs[2].Value.([]byte), wantTo) {
		t.Fatalf("to=%x", ev.Inputs[2].Value)
	}
}
```

- [ ] **Step 2: Run the tests**

Run: `go test ./abi/ -run TestEntryDecodeLog_IndexedDynamicPlaceholder -v` then `go test ./abi/ -run TestEntryDecodeLog_Erc1155TransferBatch -v`
Expected: PASS. If either fails, the Task 2 `DecodeLog`/`decodeIndexedTopic` logic has a bug — fix it in `abi/event.go` (do not weaken the test).

- [ ] **Step 3: Commit**

```bash
git add abi/event_test.go
git commit -m "test(abi): indexed-dynamic placeholder and ERC-1155 batch log decode"
```

---

### Task 4: Registry topic0 lookup + manager-level DecodeLog

**Files:**
- Modify: `abi/event.go`
- Modify: `abi/manager.go`
- Test: `abi/event_test.go` (append)

- [ ] **Step 1: Write the failing test** — append to `abi/event_test.go`:

```go
func TestManagerDecodeLog_ByContractAddress(t *testing.T) {
	m := newTestManager(t) // helper from abi_test.go: builds a manager with mem storage + hex codec
	contractAddr := "0x1234567890123456789012345678901234567890"
	transfer := &SmartContractAbiEntry{
		Name: "Transfer",
		Type: "Event",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "from", Type: "address", Indexed: true},
			{Name: "to", Type: "address", Indexed: true},
			{Name: "value", Type: "uint256"},
		},
	}
	m.Add(&SmartContractInfo{
		Name:            "Tok",
		Symbol:          "TOK",
		ContractAddress: contractAddr,
		Abi:             &SmartContractAbi{Entries: []*SmartContractAbiEntry{transfer}},
	})

	from := mustHex32(t, "000000000000000000000000"+"1111111111111111111111111111111111111111")
	to := mustHex32(t, "000000000000000000000000"+"2222222222222222222222222222222222222222")
	topics := [][32]byte{transfer.Topic0(), from, to}
	data := make([]byte, 32)
	big.NewInt(500).FillBytes(data)

	ev, err := m.DecodeLog(contractAddr, topics, data)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Name != "Transfer" || ev.Contract != contractAddr {
		t.Fatalf("event=%+v", ev)
	}
	if ev.Inputs[2].Value.(*big.Int).Int64() != 500 {
		t.Fatalf("value=%v", ev.Inputs[2].Value)
	}
}

func TestAbiGetEventByTopic0(t *testing.T) {
	transfer := &SmartContractAbiEntry{
		Name: "Transfer",
		Type: "Event",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "from", Type: "address", Indexed: true},
			{Name: "to", Type: "address", Indexed: true},
			{Name: "value", Type: "uint256"},
		},
	}
	a := &SmartContractAbi{Entries: []*SmartContractAbiEntry{transfer}}
	got, err := a.GetEventByTopic0(transfer.Topic0())
	if err != nil || got.Name != "Transfer" {
		t.Fatalf("got %v err %v", got, err)
	}
	var zero [32]byte
	if _, err := a.GetEventByTopic0(zero); err == nil {
		t.Fatal("unknown topic0 must error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./abi/ -run 'TestManagerDecodeLog_ByContractAddress|TestAbiGetEventByTopic0' -v`
Expected: FAIL — `a.GetEventByTopic0 undefined`, `m.DecodeLog undefined`.

- [ ] **Step 3: Add `GetEventByTopic0` to `*SmartContractAbi`** — append to `abi/event.go`:

```go
// GetEventByTopic0 finds the event entry whose signature hash equals topic0.
func (a *SmartContractAbi) GetEventByTopic0(topic0 [32]byte) (*SmartContractAbiEntry, error) {
	a._prepare()
	for _, entry := range a.Entries {
		if !strings.EqualFold(entry.Type, "Event") {
			continue
		}
		if entry.Topic0() == topic0 {
			return entry, nil
		}
	}
	return nil, ErrUnknownEvent
}
```

- [ ] **Step 4: Add manager-level `DecodeLog`** — append to `abi/event.go`:

```go
// DecodeLog finds the contract registered at contractAddress, matches the event
// by topics[0], and decodes the log. The returned DecodedEvent has Contract set
// to contractAddress.
func (m *SmartContractsManager) DecodeLog(contractAddress string, topics [][32]byte, data []byte) (*DecodedEvent, error) {
	if len(topics) == 0 {
		return nil, ErrTopicCountMismatch
	}
	contract, err := m.GetSmartContractByAddress(contractAddress)
	if err != nil {
		return nil, err
	}
	if contract.Abi == nil {
		return nil, ErrUnknownEvent
	}
	entry, err := contract.Abi.GetEventByTopic0(topics[0])
	if err != nil {
		return nil, err
	}
	ev, err := entry.DecodeLog(topics, data)
	if err != nil {
		return nil, err
	}
	ev.Contract = contractAddress
	return ev, nil
}
```

- [ ] **Step 5: Verify `GetSmartContractByAddress` and `SmartContractInfo.Abi` exist as used**

Run: `grep -n "func (m \*SmartContractsManager) GetSmartContractByAddress\|Abi \*SmartContractAbi" abi/manager.go abi/smartcontractabi.go`
Expected: both present (manager has the lookup; `SmartContractInfo` has an `Abi *SmartContractAbi` field). No code change needed in `manager.go` if both exist — in that case this task only touches `abi/event.go` and the test. If `GetSmartContractByAddress` is not present, STOP and report.

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./abi/ -run 'TestManagerDecodeLog_ByContractAddress|TestAbiGetEventByTopic0' -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add abi/event.go abi/event_test.go
git commit -m "feat(abi): registry topic0 lookup and manager-level DecodeLog"
```

---

### Task 5: Final verification & docs

**Files:**
- Modify: `todo/TASKS.md`

- [ ] **Step 1: Run the entire abi package with the race detector**

Run: `go test ./abi/ -race -count=1`
Expected: `ok` — no failures, no data races (ignore `[CRIT] Duplicated Contract` log lines).

- [ ] **Step 2: Confirm nothing else broke**

Run: `go build ./... && go vet ./abi/`
Expected: both exit 0.

- [ ] **Step 3: Mark M2 tasks done in `todo/TASKS.md`**

Change each of `M2.1`–`M2.4` from `- [ ]` to `- [x]` in the "Milestone 2" section, and append " ✅ DONE" to the "## Milestone 2 — ABI engine: event-log decoding (`abi/`)" header line, matching how M1 was marked. Mapping:
- M2.1 topic0 hashing → `Topic0()` (Task 1)
- M2.2 `DecodeLog(topics, data)` with indexed/data split + anonymous guard → Tasks 2 & 3
- M2.3 `GetEventByTopic0` → Task 4
- M2.4 ERC-1155 `TransferSingle`/`TransferBatch` + Transfer fixtures → Tasks 1–3 tests

- [ ] **Step 4: Commit**

```bash
git add todo/TASKS.md
git commit -m "docs(abi): mark M2 (event-log decoding) complete"
```

---

## Notes for the implementer

- **Do not** feed an indexed dynamic param to `decodeParams` — only its keccak hash exists in the topic. `decodeIndexedTopic` returns the 32-byte hash placeholder for dynamic types; static indexed params go through `decodeValue` on the single 32-byte topic slot.
- Non-indexed params are decoded together as one head/tail block via `decodeParams` (they share the `data` blob), then interleaved back into ABI order with the indexed ones.
- `DecodedValue.Value` concrete types are unchanged from M1 (`*big.Int`, `bool`, `[]byte`, `string`, `[]DecodedValue`); the only addition is the 32-byte `[]byte` hash placeholder for indexed dynamics, distinguished by the " (indexed)" suffix in `.Type`.
- `topics` is typed `[][32]byte` throughout (not `[]string`); the M4/M5 client layer will convert hex topic strings from JSON-RPC into `[32]byte` when it lands — out of scope for M2.
- The topic0 test vectors (`ddf252ad…` for `Transfer`, `c3d58168…` for `TransferSingle`) are the real, well-known mainnet values — they double as a correctness check on `canonicalSignature()` + `Keccak256`.
- Keep `abi/event.go` focused on event logic only; do not move method-decoding code into it.
