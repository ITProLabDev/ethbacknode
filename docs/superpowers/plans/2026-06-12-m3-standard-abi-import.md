# M3 — Standard ABI Import & Generalized Registry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Accept canonical Ethereum (solc/Etherscan/Polygonscan) JSON ABI directly — converting it into the project's own ABI model — and register arbitrary contracts (not just ERC-20 tokens) with their ABI, while keeping the existing ERC-20 path and `known_contracts.json` working unchanged.

**Architecture:** The project's `SmartContractAbiEntry`/`SmartContractAbiEntryInput` structs already carry json tags matching the canonical ABI shape (`name`, `type`, `inputs`, `outputs`, `components`, `indexed`, `stateMutability`), and the only `Type` comparisons in the codebase are case-insensitive (`strings.EqualFold`). So importing is a validated `json.Unmarshal` that (a) accepts a top-level JSON array (the canonical ABI form) and wraps it as `SmartContractAbi.Entries`, and (b) validates each entry parses with the M1/M2 typed engine. A convenience constructor builds a ready-to-use `SmartContractInfo` from an address + raw ABI. The registry is already generic over arbitrary contracts; M3 confirms this with regression tests rather than restructuring.

**Tech Stack:** Go 1.24, `encoding/json`, the M1/M2 typed engine (`abi/abitype.go`, `abi/codec.go`, `abi/event.go`), the existing `abi/manager.go` registry, standard `testing`.

---

## Background & Constraints

**Hard rule (carried from M1/M2):** the `abi/` package stays a self-built engine — NO go-ethereum or external ABI library. The importer must NOT pull in a third-party ABI parser; it only adapts JSON shape into our own structs and validates with our own engine.

**Canonical Ethereum JSON ABI (what we accept):** a JSON **array** of entry objects, e.g.:
```json
[
  {"type":"function","name":"transfer","stateMutability":"nonpayable",
   "inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],
   "outputs":[{"name":"","type":"bool"}]},
  {"type":"event","name":"Transfer","anonymous":false,
   "inputs":[{"name":"from","type":"address","indexed":true},
             {"name":"to","type":"address","indexed":true},
             {"name":"value","type":"uint256","indexed":false}]}
]
```
Field casing is **lowercase** (`"type":"function"`, `"event"`, `"constructor"`; `"stateMutability":"view"`). The project's own template (`abi/abi_tpls.go`) uses Capitalized values (`"Function"`, `"Event"`, `"View"`) — both must work, which they do because all `Type` comparisons use `strings.EqualFold`.

**What already exists (verified against the code):**
- `SmartContractAbi{ Entries []*SmartContractAbiEntry }` and `SmartContractAbiEntry{ Constant, Signature[4]byte (json:"-"), Name, StateMutability, Type, Inputs, Outputs }` with json tags already matching canonical ABI (`abi/smartcontractabi.go`).
- `SmartContractAbiEntryInput{ Name, Type, Indexed, Components, data }` — matches canonical (`components`, `indexed`).
- `SmartContractAbiEntryOutput{ Type, Name }`.
- `SmartContractInfo{ Name, Symbol, ContractAddress, OriginAddress, Decimals, OriginGasLimit, Abi *SmartContractAbi }` — already holds an arbitrary ABI.
- Typed engine: `parseType(typeStr, components) (abiType, error)` validates a type string; `(*SmartContractAbiEntry).canonicalSignature()`, `typedInputs()`, `Topic0()`, `DecodeLog`, `DecodeInputsTyped`.
- Registry: `(*SmartContractsManager).Add(*SmartContractInfo)`, `Load`, `GetSmartContractByAddress`, `GetSmartContractByToken`, `GetSmartContractList`, `Walk`; lookup maps `byName`/`bySymbol`/`byAddress` (address lowercased).
- `newTestManager(t)` test helper (`abi/abi_test.go`) builds a manager with in-memory storage + hex address codec.
- Errors in `abi/errors.go`: `ErrInvalidParamsData`, `ErrUnknownContract`, etc.

**Decisions locked for M3:**
- The importer does NOT mutate field casing (no normalization to "Function"). Canonical lowercase values are kept as-is; the engine is already case-insensitive where it matters. This keeps round-tripping faithful and avoids surprising the caller.
- Polymarket seed fixtures (M3.3) are stored as **test fixtures and a small embedded constant**, not auto-registered at runtime (no network calls, no hardcoded mainnet behavior in the service). They prove the importer handles real ABIs.

---

## File Structure

- **Create `abi/import.go`** — `ImportEthereumABI(raw []byte) (*SmartContractAbi, error)`, `NewContractFromABI(name, symbol, address string, rawABI []byte) (*SmartContractInfo, error)`.
- **Create `abi/import_test.go`** — importer unit tests (array form, `{entries:[...]}` form, validation errors, tuple/components, round-trip through Topic0/canonicalSignature), generalized-registry regression, and Polymarket-ABI import fixtures.
- **Create `abi/testdata/`** — small real-ABI JSON fixtures: `erc20_min.json`, `erc1155_ctf.json` (Conditional Tokens subset: `balanceOf`, `TransferSingle`, `TransferBatch`, `splitPosition`), `ctf_exchange.json` (a representative `OrderFilled` event + an order-struct method to exercise tuples). These are hand-written minimal subsets, NOT full mainnet ABIs.
- **Modify `abi/errors.go`** — add `ErrEmptyABI`, `ErrInvalidABIJSON`.

No changes to the codec, the legacy decoder, `erc20.go`, or `manager.go` internals (the registry is already generic). `manager.go` is touched only if M3.2 adds a convenience method — see Task 3, which keeps it in `import.go` instead to avoid touching the manager.

---

### Task 1: `ImportEthereumABI` — parse canonical JSON ABI (array or object form)

**Files:**
- Create: `abi/import.go`
- Modify: `abi/errors.go`
- Test: `abi/import_test.go`

- [ ] **Step 1: Add error values** to `abi/errors.go` — insert inside the existing `var ( ... )` block, after `ErrTopicCountMismatch`:

```go
	// ErrEmptyABI is returned when an imported ABI contains no entries.
	ErrEmptyABI = errors.New("abi is empty")
	// ErrInvalidABIJSON is returned when ABI JSON cannot be parsed.
	ErrInvalidABIJSON = errors.New("invalid abi json")
```

- [ ] **Step 2: Write the failing tests** — create `abi/import_test.go`:

```go
package abi

import (
	"strings"
	"testing"
)

const erc20ABIArray = `[
  {"type":"function","name":"transfer","stateMutability":"nonpayable",
   "inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],
   "outputs":[{"name":"","type":"bool"}]},
  {"type":"event","name":"Transfer","anonymous":false,
   "inputs":[{"name":"from","type":"address","indexed":true},
             {"name":"to","type":"address","indexed":true},
             {"name":"value","type":"uint256","indexed":false}]}
]`

func TestImportEthereumABI_ArrayForm(t *testing.T) {
	a, err := ImportEthereumABI([]byte(erc20ABIArray))
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Entries) != 2 {
		t.Fatalf("entries=%d want 2", len(a.Entries))
	}
	transfer, err := a.GetMethodByName("transfer")
	if err != nil {
		t.Fatal(err)
	}
	if len(transfer.Inputs) != 2 || transfer.Inputs[0].Type != "address" {
		t.Fatalf("transfer inputs=%+v", transfer.Inputs)
	}
	// The lowercase "function"/"event" types must be preserved as-is.
	if transfer.Type != "function" {
		t.Fatalf("type=%q want lowercase function", transfer.Type)
	}
	// Event lookup by the canonical mainnet topic0 proves the event imported
	// correctly (Transfer(address,address,uint256)).
	ev, err := a.GetEventByTopic0(mustHex32(t, "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"))
	if err != nil || ev.Name != "Transfer" {
		t.Fatalf("event by topic0: %v %v", ev, err)
	}
}

func TestImportEthereumABI_ObjectForm(t *testing.T) {
	// Also accept the project's own { "entries": [...] } wrapper form.
	obj := `{"entries":` + erc20ABIArray + `}`
	a, err := ImportEthereumABI([]byte(obj))
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Entries) != 2 {
		t.Fatalf("entries=%d want 2", len(a.Entries))
	}
}

func TestImportEthereumABI_Errors(t *testing.T) {
	cases := map[string]string{
		"empty array":   `[]`,
		"empty object":  `{"entries":[]}`,
		"not json":      `not json at all`,
		"bad type":      `[{"type":"function","name":"x","inputs":[{"name":"a","type":"uint999"}]}]`,
	}
	for name, raw := range cases {
		if _, err := ImportEthereumABI([]byte(raw)); err == nil {
			t.Fatalf("%s: expected error", name)
		}
	}
}

func TestImportEthereumABI_Tuple(t *testing.T) {
	// A function taking a struct (tuple with components) must import and its
	// canonical signature must render the tuple form.
	raw := `[
	  {"type":"function","name":"submit","stateMutability":"nonpayable",
	   "inputs":[{"name":"order","type":"tuple",
	     "components":[{"name":"amt","type":"uint256"},{"name":"maker","type":"address"}]}],
	   "outputs":[]}
	]`
	a, err := ImportEthereumABI([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	m, err := a.GetMethodByName("submit")
	if err != nil {
		t.Fatal(err)
	}
	if got := m.canonicalSignature(); got != "submit((uint256,address))" {
		t.Fatalf("canonicalSignature=%q", got)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./abi/ -run TestImportEthereumABI -v`
Expected: FAIL — `undefined: ImportEthereumABI`.

- [ ] **Step 4: Write minimal implementation** — create `abi/import.go`:

```go
package abi

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// ImportEthereumABI converts a canonical Ethereum JSON ABI into the project's
// own SmartContractAbi. It accepts either the standard top-level array form
// (as emitted by solc / Etherscan / Polygonscan) or the project's own
// {"entries": [...]} object form. Every entry's input types are validated with
// the typed engine, so a malformed type fails fast at import time.
//
// Field casing is preserved as-is: canonical ABIs use lowercase type values
// ("function", "event"); the engine compares them case-insensitively.
func ImportEthereumABI(raw []byte) (*SmartContractAbi, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, ErrInvalidABIJSON
	}

	abi := &SmartContractAbi{}
	switch trimmed[0] {
	case '[':
		var entries []*SmartContractAbiEntry
		if err := json.Unmarshal(trimmed, &entries); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidABIJSON, err)
		}
		abi.Entries = entries
	case '{':
		if err := json.Unmarshal(trimmed, abi); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidABIJSON, err)
		}
	default:
		return nil, ErrInvalidABIJSON
	}

	if len(abi.Entries) == 0 {
		return nil, ErrEmptyABI
	}

	// Validate every entry's input/output types parse with the typed engine.
	for _, e := range abi.Entries {
		for _, in := range e.Inputs {
			if _, err := parseType(in.Type, in.Components); err != nil {
				return nil, fmt.Errorf("%w: entry %q input %q: %v", ErrInvalidABIJSON, e.Name, in.Type, err)
			}
		}
		for _, out := range e.Outputs {
			if _, err := parseType(out.Type, nil); err != nil {
				return nil, fmt.Errorf("%w: entry %q output %q: %v", ErrInvalidABIJSON, e.Name, out.Type, err)
			}
		}
	}

	return abi, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./abi/ -run TestImportEthereumABI -v`
Expected: PASS (all four).

Note: `TestImportEthereumABI_Errors` "bad type" case relies on `parseType("uint999", nil)` returning an error — it does (width not in 8..256, step 8). The empty cases rely on the `ErrEmptyABI` guard.

- [ ] **Step 6: Commit**

```bash
git add abi/import.go abi/errors.go abi/import_test.go
git commit -m "feat(abi): import canonical Ethereum JSON ABI (array or object form)"
```

---

### Task 2: `NewContractFromABI` — build a registrable contract from an address + raw ABI

**Files:**
- Modify: `abi/import.go`
- Test: `abi/import_test.go` (append)

- [ ] **Step 1: Write the failing test** — append to `abi/import_test.go`:

```go
func TestNewContractFromABI(t *testing.T) {
	c, err := NewContractFromABI("MyToken", "MTK", "0xAbC1230000000000000000000000000000000001", []byte(erc20ABIArray))
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "MyToken" || c.Symbol != "MTK" {
		t.Fatalf("contract meta=%+v", c)
	}
	if c.ContractAddress != "0xAbC1230000000000000000000000000000000001" {
		t.Fatalf("address not preserved: %q", c.ContractAddress)
	}
	if c.Abi == nil || len(c.Abi.Entries) != 2 {
		t.Fatalf("abi not attached: %+v", c.Abi)
	}
}

func TestNewContractFromABI_RejectsBadABI(t *testing.T) {
	if _, err := NewContractFromABI("X", "X", "0x01", []byte(`[]`)); err == nil {
		t.Fatal("empty abi must be rejected")
	}
	if _, err := NewContractFromABI("X", "X", "0x01", []byte(`garbage`)); err == nil {
		t.Fatal("bad json must be rejected")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./abi/ -run TestNewContractFromABI -v`
Expected: FAIL — `undefined: NewContractFromABI`.

- [ ] **Step 3: Write minimal implementation** — append to `abi/import.go`:

```go
// NewContractFromABI builds a SmartContractInfo from a name, symbol, contract
// address, and a canonical Ethereum JSON ABI. The address is stored as-is
// (the registry lowercases it for lookups). Returns an error if the ABI is
// empty or invalid.
func NewContractFromABI(name, symbol, address string, rawABI []byte) (*SmartContractInfo, error) {
	abi, err := ImportEthereumABI(rawABI)
	if err != nil {
		return nil, err
	}
	return &SmartContractInfo{
		Name:            name,
		Symbol:          symbol,
		ContractAddress: address,
		Abi:             abi,
	}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./abi/ -run TestNewContractFromABI -v`
Expected: PASS (both).

- [ ] **Step 5: Commit**

```bash
git add abi/import.go abi/import_test.go
git commit -m "feat(abi): NewContractFromABI constructor from address + raw ABI"
```

---

### Task 3: Generalized-registry regression — register & use an arbitrary (non-token) contract

**Files:**
- Test: `abi/import_test.go` (append)

This task adds no production code. It proves the registry already supports arbitrary contracts end-to-end: import a non-ERC-20 ABI, register it, look it up by address, and decode one of its events through the manager. If anything fails, the registry needs a fix (fix in `abi/manager.go` only if genuinely required, and report it).

- [ ] **Step 1: Write the test** — append to `abi/import_test.go`:

```go
func TestRegistry_ArbitraryContractRoundTrip(t *testing.T) {
	m := newTestManager(t) // mem storage + hex codec, from abi_test.go

	raw := `[
	  {"type":"function","name":"balanceOf","stateMutability":"view",
	   "inputs":[{"name":"account","type":"address"},{"name":"id","type":"uint256"}],
	   "outputs":[{"name":"","type":"uint256"}]},
	  {"type":"event","name":"TransferSingle","anonymous":false,
	   "inputs":[{"name":"operator","type":"address","indexed":true},
	             {"name":"from","type":"address","indexed":true},
	             {"name":"to","type":"address","indexed":true},
	             {"name":"id","type":"uint256","indexed":false},
	             {"name":"value","type":"uint256","indexed":false}]}
	]`
	addr := "0xabc0000000000000000000000000000000000111"
	c, err := NewContractFromABI("CTF", "CTF", addr, []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	m.Add(c)

	got, err := m.GetSmartContractByAddress(addr)
	if err != nil || got.Name != "CTF" {
		t.Fatalf("lookup by address: %v %v", got, err)
	}

	ev, err := got.Abi.GetMethodByName("TransferSingle")
	if err != nil {
		t.Fatal(err)
	}
	op := mustHex32(t, "000000000000000000000000"+"1111111111111111111111111111111111111111")
	from := mustHex32(t, "000000000000000000000000"+"2222222222222222222222222222222222222222")
	to := mustHex32(t, "000000000000000000000000"+"3333333333333333333333333333333333333333")
	topics := [][32]byte{ev.Topic0(), op, from, to}
	idsType, _ := parseType("uint256", nil)
	valType, _ := parseType("uint256", nil)
	data, err := encodeParams([]abiType{idsType, valType}, []any{big.NewInt(5), big.NewInt(9)})
	if err != nil {
		t.Fatal(err)
	}

	decoded, err := m.DecodeLog(addr, topics, data)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Name != "TransferSingle" || len(decoded.Inputs) != 5 {
		t.Fatalf("decoded=%+v", decoded)
	}
	if decoded.Inputs[3].Value.(*big.Int).Int64() != 5 {
		t.Fatalf("id=%v", decoded.Inputs[3].Value)
	}
	if decoded.Inputs[4].Value.(*big.Int).Int64() != 9 {
		t.Fatalf("value=%v", decoded.Inputs[4].Value)
	}
}
```

Add `"math/big"` to the `abi/import_test.go` import block (it currently imports `strings`, `testing`). `GetMethodByName` works for events too (it matches by Name across all entries), so it is fine to fetch the event entry with it.

- [ ] **Step 2: Run the test**

Run: `go test ./abi/ -run TestRegistry_ArbitraryContractRoundTrip -v`
Expected: PASS. If it fails inside the registry (not the test), fix `abi/manager.go` and report what was needed.

- [ ] **Step 3: Commit**

```bash
git add abi/import_test.go
git commit -m "test(abi): arbitrary (non-token) contract registry round-trip"
```

---

### Task 4: Generic contract-class ABI fixtures import

These fixtures are **generic, role-based representatives** of complex contract
shapes — NOT the deployed Polymarket/CTF contracts. They exist to prove the
importer handles real-world ABI complexity (dynamic arrays, `bytes32`, tuple
structs, mixed indexed/non-indexed events), regardless of which contract it is.
No brand names; describe by role. Method/event names like `splitPosition`,
`fillOrder`, `OrderFilled` are generic prediction-market / order-book terms.

**Files:**
- Create: `abi/testdata/multitoken_erc1155.json`
- Create: `abi/testdata/outcome_market.json`
- Create: `abi/testdata/clob_exchange.json`
- Test: `abi/import_test.go` (append)

- [ ] **Step 1: Create `abi/testdata/multitoken_erc1155.json`** — a multi-token (ERC-1155) role: per-id balances + batch + transfer events with dynamic arrays:

```json
[
  {"type":"function","name":"balanceOf","stateMutability":"view",
   "inputs":[{"name":"account","type":"address"},{"name":"id","type":"uint256"}],
   "outputs":[{"name":"","type":"uint256"}]},
  {"type":"function","name":"balanceOfBatch","stateMutability":"view",
   "inputs":[{"name":"accounts","type":"address[]"},{"name":"ids","type":"uint256[]"}],
   "outputs":[{"name":"","type":"uint256[]"}]},
  {"type":"event","name":"TransferSingle","anonymous":false,
   "inputs":[{"name":"operator","type":"address","indexed":true},
             {"name":"from","type":"address","indexed":true},
             {"name":"to","type":"address","indexed":true},
             {"name":"id","type":"uint256","indexed":false},
             {"name":"value","type":"uint256","indexed":false}]},
  {"type":"event","name":"TransferBatch","anonymous":false,
   "inputs":[{"name":"operator","type":"address","indexed":true},
             {"name":"from","type":"address","indexed":true},
             {"name":"to","type":"address","indexed":true},
             {"name":"ids","type":"uint256[]","indexed":false},
             {"name":"values","type":"uint256[]","indexed":false}]}
]
```

- [ ] **Step 2: Create `abi/testdata/outcome_market.json`** — an outcome/condition-market role: a method and event using `bytes32` ids + dynamic arrays + indexed `bytes32`:

```json
[
  {"type":"function","name":"splitPosition","stateMutability":"nonpayable",
   "inputs":[{"name":"collateralToken","type":"address"},
             {"name":"parentCollectionId","type":"bytes32"},
             {"name":"conditionId","type":"bytes32"},
             {"name":"partition","type":"uint256[]"},
             {"name":"amount","type":"uint256"}],
   "outputs":[]},
  {"type":"event","name":"PositionSplit","anonymous":false,
   "inputs":[{"name":"stakeholder","type":"address","indexed":true},
             {"name":"collateralToken","type":"address","indexed":false},
             {"name":"parentCollectionId","type":"bytes32","indexed":true},
             {"name":"conditionId","type":"bytes32","indexed":true},
             {"name":"partition","type":"uint256[]","indexed":false},
             {"name":"amount","type":"uint256","indexed":false}]}
]
```

- [ ] **Step 3: Create `abi/testdata/clob_exchange.json`** — an order-book (CLOB) role exercising a tuple/struct order:

```json
[
  {"type":"event","name":"OrderFilled","anonymous":false,
   "inputs":[{"name":"orderHash","type":"bytes32","indexed":true},
             {"name":"maker","type":"address","indexed":true},
             {"name":"taker","type":"address","indexed":true},
             {"name":"makerAssetId","type":"uint256","indexed":false},
             {"name":"takerAssetId","type":"uint256","indexed":false},
             {"name":"makerAmountFilled","type":"uint256","indexed":false},
             {"name":"takerAmountFilled","type":"uint256","indexed":false},
             {"name":"fee","type":"uint256","indexed":false}]},
  {"type":"function","name":"fillOrder","stateMutability":"nonpayable",
   "inputs":[{"name":"order","type":"tuple",
     "components":[{"name":"salt","type":"uint256"},
                   {"name":"maker","type":"address"},
                   {"name":"signer","type":"address"},
                   {"name":"taker","type":"address"},
                   {"name":"tokenId","type":"uint256"},
                   {"name":"makerAmount","type":"uint256"},
                   {"name":"takerAmount","type":"uint256"},
                   {"name":"side","type":"uint8"},
                   {"name":"signature","type":"bytes"}]},
             {"name":"fillAmount","type":"uint256"}],
   "outputs":[]}
]
```

- [ ] **Step 4: Write the failing test** — append to `abi/import_test.go`:

```go
func TestImport_GenericContractClassFixtures(t *testing.T) {
	// Multi-token (ERC-1155) role: dynamic-array methods + transfer events.
	mt, err := os.ReadFile("testdata/multitoken_erc1155.json")
	if err != nil {
		t.Fatal(err)
	}
	a, err := ImportEthereumABI(mt)
	if err != nil {
		t.Fatalf("multitoken import: %v", err)
	}
	if _, err := a.GetMethodByName("balanceOfBatch"); err != nil {
		t.Fatal(err)
	}
	// TransferSingle event imported with the real mainnet ERC-1155 topic0.
	if _, err := a.GetEventByTopic0(mustHex32(t, "c3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62")); err != nil {
		t.Fatalf("TransferSingle topic0 not found: %v", err)
	}

	// Outcome-market role: bytes32 ids + dynamic arrays + indexed bytes32.
	om, err := os.ReadFile("testdata/outcome_market.json")
	if err != nil {
		t.Fatal(err)
	}
	oa, err := ImportEthereumABI(om)
	if err != nil {
		t.Fatalf("outcome_market import: %v", err)
	}
	if _, err := oa.GetMethodByName("splitPosition"); err != nil {
		t.Fatal(err)
	}
	if _, err := oa.GetMethodByName("PositionSplit"); err != nil { // GetMethodByName matches events by name too
		t.Fatal(err)
	}

	// Order-book (CLOB) role: a 9-field order tuple.
	ex, err := os.ReadFile("testdata/clob_exchange.json")
	if err != nil {
		t.Fatal(err)
	}
	ea, err := ImportEthereumABI(ex)
	if err != nil {
		t.Fatalf("clob_exchange import: %v", err)
	}
	fo, err := ea.GetMethodByName("fillOrder")
	if err != nil {
		t.Fatal(err)
	}
	want := "fillOrder((uint256,address,address,address,uint256,uint256,uint256,uint8,bytes),uint256)"
	if got := fo.canonicalSignature(); got != want {
		t.Fatalf("fillOrder canonicalSignature=\n%q\nwant\n%q", got, want)
	}
}
```

Add `"os"` to the `abi/import_test.go` import block.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./abi/ -run TestImport_GenericContractClassFixtures -v`
Expected: PASS. (If the TransferSingle topic0 assertion fails, the fixture's event signature doesn't match canonical — re-check the fixture's param types/order.)

- [ ] **Step 6: Commit**

```bash
git add abi/testdata/multitoken_erc1155.json abi/testdata/outcome_market.json abi/testdata/clob_exchange.json abi/import_test.go
git commit -m "test(abi): import generic contract-class ABI fixtures (multitoken, outcome-market, clob)"
```

---

### Task 5: Backward-compat regression — `known_contracts.json` + ERC-20 template still load

**Files:**
- Test: `abi/import_test.go` (append)

- [ ] **Step 1: Write the test** — append to `abi/import_test.go`:

```go
func TestBackwardCompat_ColdStartAndErc20(t *testing.T) {
	// newTestManager calls Init, which ColdStarts the built-in template and
	// loads the ERC-20 abi. The legacy path must still work after M3.
	m := newTestManager(t)
	list := m.GetSmartContractList()
	if len(list) == 0 {
		t.Fatal("cold start should load at least one template contract")
	}
	// ERC-20 transfer detection (legacy static codec path) must still work.
	addr := mustHex(t, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	data := transferCallData(t, addr, big.NewInt(123)) // helper from abi_test.go
	if !m.Erc20IsTransfer(data) {
		t.Fatal("ERC-20 transfer detection broke")
	}
	gotAddr, gotAmount, err := m.Erc20DecodeIfTransfer(data)
	if err != nil || gotAmount.Int64() != 123 {
		t.Fatalf("legacy decode broke: %s %v %v", gotAddr, gotAmount, err)
	}
}

func TestImportEthereumABI_ProjectTemplateRoundTrips(t *testing.T) {
	// The project's own erc20 template (capitalized "Function"/"Event" types,
	// object form) must import via ImportEthereumABI without error.
	a, err := ImportEthereumABI([]byte(erc20tpl))
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Entries) == 0 {
		t.Fatal("template imported empty")
	}
	if _, err := a.GetMethodByName("transfer"); err != nil {
		t.Fatalf("template transfer missing: %v", err)
	}
}
```

`transferCallData`, `mustHex`, and the `erc20tpl` const all already exist (in `abi_test.go` and `abi_tpls.go`). `big` is imported from Task 3.

- [ ] **Step 2: Run the tests**

Run: `go test ./abi/ -run 'TestBackwardCompat_ColdStartAndErc20|TestImportEthereumABI_ProjectTemplateRoundTrips' -v`
Expected: PASS. If `ProjectTemplateRoundTrips` fails because the template contains a type the engine rejects (e.g. `uint8` is fine; check `string`/`bool`/`address` all parse), fix `parseType` ONLY if it's a real gap — otherwise adjust nothing (the template is known-good ERC-20 types).

- [ ] **Step 3: Commit**

```bash
git add abi/import_test.go
git commit -m "test(abi): backward-compat — cold start, ERC-20 path, template import"
```

---

### Task 6: Final verification & docs

**Files:**
- Modify: `todo/TASKS.md`

- [ ] **Step 1: Run the entire abi package with the race detector**

Run: `go test ./abi/ -race -count=1`
Expected: `ok` — no failures, no data races (ignore `[CRIT] Duplicated Contract` log lines from a pre-existing concurrency test).

- [ ] **Step 2: Confirm nothing else broke**

Run: `go build ./... && go vet ./abi/`
Expected: both exit 0.

- [ ] **Step 3: Mark M3 tasks done in `todo/TASKS.md`**

Change each of `M3.1`–`M3.4` from `- [ ]` to `- [x]` in the "Milestone 3" section, and append " ✅ DONE" to the "## Milestone 3 — Standard ABI import & generalized registry (`abi/`)" header line, matching how M1/M2 were marked. Mapping:
- M3.1 `ImportEthereumABI` (array + object form, validation) → Tasks 1
- M3.2 generalized registry for arbitrary contracts → Tasks 2 & 3 (`NewContractFromABI` + round-trip; registry was already generic)
- M3.3 Polymarket seed fixtures (CTF ERC-1155, CTF Exchange) → Task 4
- M3.4 backward-compat (`known_contracts.json`, ERC-20 template) → Task 5

- [ ] **Step 4: Commit**

```bash
git add todo/TASKS.md
git commit -m "docs(abi): mark M3 (standard ABI import & generalized registry) complete"
```

---

## Notes for the implementer

- **Do not** add a go-ethereum or any third-party ABI dependency. The importer only adapts JSON shape into the project's own structs and validates with the project's own `parseType`.
- The importer preserves field casing verbatim — do NOT rewrite `"function"`→`"Function"`. The engine is already case-insensitive on `Type` (`strings.EqualFold` in `event.go`).
- `GetMethodByName` matches by `Name` across all entries regardless of `Type`, so it can fetch event entries too (used in tests to get an event's `Topic0()`).
- Fixtures in `abi/testdata/` are minimal, hand-written subsets of real Polymarket-on-Polygon contracts — enough to exercise arrays, `bytes32`, tuples, and indexed/non-indexed events. They are NOT auto-registered by the running service; they only prove the importer handles real-world ABI shapes.
- The two real topic0 vectors (`ddf252ad…` Transfer, `c3d58168…` TransferSingle) double as correctness checks that the imported event signatures are byte-correct.
- Run only `./abi/` tests; the repo has pre-existing unrelated compile errors in `crypto/secp256k1` and a noisy test in `clients/urpc`. The `[CRIT] Duplicated Contract` lines in the abi suite are from a pre-existing concurrency test and are harmless.
