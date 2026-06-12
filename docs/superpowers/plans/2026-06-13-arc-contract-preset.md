# Arc Prediction-Market Contract Preset Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a versioned, `//go:embed`-ed contract preset (8 Arc prediction-market contracts, chainId 5042002) that is applied to the abi registry at startup via the §6 data-driven flow, so each contract automatically gains `contractEvent` delivery, `contractCall`, and subscriptions.

**Architecture:** A new isolated `presets/` package embeds `presets/arc.json` (canonical ABIs verbatim from the deployment artifact) and exposes `Apply(chainID int64, adder ContractAdder) (int, error)`. `main.go` calls it after `chainClient.Init()`, matching on the numeric chainId from `eth_chainId` (`GetNetId()`). Registration goes through `abi.NewContractFromABI` + `adder.Add`, which is exactly what `AddContractFromABI` does internally, but lets the loader carry `decimals`. Non-fatal, idempotent, chain-isolated.

**Tech Stack:** Go 1.24 (`//go:embed`), the project's own `abi` engine (`ImportEthereumABI` / `NewContractFromABI`), no external libraries.

---

## File Structure

| File | Responsibility |
|------|----------------|
| `presets/arc.json` (create) | The embedded preset bundle: `{chainId, source, generatedAt, contracts:[{name,symbol,decimals,address,abi:[<canonical ABI array>]}×8]}`. Generated from `_sources/contracts/deployments/arc.json`. |
| `presets/presets.go` (create) | `//go:embed arc.json`; types `bundle`/`presetContract`; narrow `ContractAdder` interface; `Apply(chainID int64, adder ContractAdder) (loaded int, err error)`. |
| `presets/presets_test.go` (create) | Unit tests (node-free): embed parses, apply-by-chainid, no-match, idempotency, bad-ABI error. |
| `main.go` (modify, after line 146) | Call `presets.Apply(netId, abiManager)` after the chain id is logged. |

**Note on `abi` coupling:** `presets/` imports `abi` directly. `abi` is a pure
library with no service dependencies, so this introduces no cycle and matches how
`eventlog`/`endpoint` use it.

---

## Task 1: Generate the embedded preset bundle `presets/arc.json`

**Files:**
- Create: `presets/arc.json`

This file is generated **once** from the (gitignored) deployment artifact and
committed. It is the version-controlled source of truth for the preset.

- [ ] **Step 1: Generate the file from the artifact**

Run this exact command from the repo root:

```bash
mkdir -p presets && python3 - <<'PY'
import json
d = json.load(open('_sources/contracts/deployments/arc.json'))
core = ['JustifyAccessControl','MockUSDC','OutcomeToken','FeeTreasury','OracleResolver','MarketFactory']
seeded = ['PredictionMarket','MarketAMM']
def mk(name, c):
    return {
        'name': name,
        'symbol': name,
        'decimals': 6 if name == 'MockUSDC' else 0,
        'address': c['address'],
        'abi': c['abi'],
    }
contracts = [mk(n, d['contracts'][n]) for n in core]
contracts += [mk(n, d['seededMarket']['contracts'][n]) for n in seeded]
bundle = {
    'chainId': d['chainId'],
    'source': '_sources/contracts/deployments/arc.json',
    'generatedAt': d['deployedAt'],
    'contracts': contracts,
}
open('presets/arc.json','w').write(json.dumps([bundle], indent=2) + '\n')
print('wrote presets/arc.json:', len(contracts), 'contracts, chainId', bundle['chainId'])
PY
```

Expected output: `wrote presets/arc.json: 8 contracts, chainId 5042002`

Note the top level is a **JSON array of bundles** (`[ {chainId,...} ]`) so more
chains can be added later without changing the embed/parse code.

- [ ] **Step 2: Verify the generated file is well-formed and complete**

Run:

```bash
python3 - <<'PY'
import json
b = json.load(open('presets/arc.json'))
assert isinstance(b, list) and len(b) == 1, 'top level must be a 1-element array'
bundle = b[0]
assert bundle['chainId'] == 5042002, bundle['chainId']
cs = bundle['contracts']
assert len(cs) == 8, len(cs)
names = [c['name'] for c in cs]
assert names == ['JustifyAccessControl','MockUSDC','OutcomeToken','FeeTreasury','OracleResolver','MarketFactory','PredictionMarket','MarketAMM'], names
assert [c for c in cs if c['name']=='MockUSDC'][0]['decimals'] == 6
addrs = [c['address'].lower() for c in cs]
assert len(set(addrs)) == 8, 'addresses must be unique'
for c in cs:
    assert isinstance(c['abi'], list) and len(c['abi']) > 0, c['name']
print('OK: 8 unique contracts, MockUSDC decimals=6, all ABIs non-empty')
PY
```

Expected output: `OK: 8 unique contracts, MockUSDC decimals=6, all ABIs non-empty`

- [ ] **Step 3: Commit**

```bash
git add -f presets/arc.json
git commit -m "feat(preset): add embedded Arc prediction-market contract bundle

Generated from _sources/contracts/deployments/arc.json (chainId 5042002):
8 contracts (6 core + 2 seeded-market), canonical ABIs verbatim, MockUSDC
decimals=6. Version-controlled source of truth for the startup preset.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: `presets` package — types, embed, and the failing test

**Files:**
- Create: `presets/presets.go`
- Test: `presets/presets_test.go`

- [ ] **Step 1: Write the failing test**

Create `presets/presets_test.go`:

```go
package presets

import (
	"errors"
	"testing"

	"github.com/ITProLabDev/ethbacknode/abi"
)

// fakeAdder records every contract handed to Add and can simulate a failure.
type fakeAdder struct {
	added   []*abi.SmartContractInfo
	failOn  string // contract name to fail validation for (unused here)
}

func (f *fakeAdder) Add(c *abi.SmartContractInfo) { f.added = append(f.added, c) }

func TestApply_ArcChainRegistersAllEight(t *testing.T) {
	adder := &fakeAdder{}
	n, err := Apply(5042002, adder)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if n != 8 {
		t.Fatalf("loaded=%d want 8", n)
	}
	if len(adder.added) != 8 {
		t.Fatalf("added=%d want 8", len(adder.added))
	}
	// Spot-check one contract: MockUSDC carries decimals=6 and a parsed ABI.
	var usdc *abi.SmartContractInfo
	for _, c := range adder.added {
		if c.Name == "MockUSDC" {
			usdc = c
		}
	}
	if usdc == nil {
		t.Fatal("MockUSDC not registered")
	}
	if usdc.Decimals != 6 {
		t.Fatalf("MockUSDC decimals=%d want 6", usdc.Decimals)
	}
	if usdc.Symbol != "MockUSDC" {
		t.Fatalf("MockUSDC symbol=%q", usdc.Symbol)
	}
	if usdc.Abi == nil || len(usdc.Abi.Entries) == 0 {
		t.Fatal("MockUSDC ABI not parsed")
	}
}

func TestApply_UnknownChainRegistersNothing(t *testing.T) {
	adder := &fakeAdder{}
	n, err := Apply(1, adder) // Ethereum mainnet — no bundle
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if n != 0 || len(adder.added) != 0 {
		t.Fatalf("loaded=%d added=%d want 0/0", n, len(adder.added))
	}
}

// errAdder is a fakeAdder that does not matter here; this guards the ABI is
// validated before Add — a malformed embedded ABI would surface as an error.
// (Covered indirectly: all embedded ABIs are valid, so Apply must NOT error.)
var _ = errors.New
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./presets/ -run TestApply -v`
Expected: FAIL — build error, `undefined: Apply` (and the package does not exist yet).

- [ ] **Step 3: Write the minimal implementation**

Create `presets/presets.go`:

```go
// Package presets ships built-in smart-contract bundles embedded in the binary
// and applies them to the abi registry at startup, matched by numeric chainId.
// This is the data/registration-driven preset path (PROJECT_STATUS §6): no
// contract-specific Go logic — only ABI data + registration.
package presets

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/ITProLabDev/ethbacknode/abi"
)

//go:embed arc.json
var arcBundleRaw []byte

// presetContract is one contract in a bundle: identity + its canonical ABI.
type presetContract struct {
	Name     string          `json:"name"`
	Symbol   string          `json:"symbol"`
	Decimals int             `json:"decimals"`
	Address  string          `json:"address"`
	ABI      json.RawMessage `json:"abi"` // canonical Ethereum ABI array, verbatim
}

// bundle is the set of contracts deployed to one chain.
type bundle struct {
	ChainID     int64            `json:"chainId"`
	Source      string           `json:"source"`
	GeneratedAt string           `json:"generatedAt"`
	Contracts   []presetContract `json:"contracts"`
}

// ContractAdder is the narrow registry surface Apply needs. Satisfied by
// *abi.SmartContractsManager.
type ContractAdder interface {
	Add(c *abi.SmartContractInfo)
}

// allBundles parses every embedded preset file into bundles.
func allBundles() ([]bundle, error) {
	var bundles []bundle
	if err := json.Unmarshal(arcBundleRaw, &bundles); err != nil {
		return nil, fmt.Errorf("presets: parse arc.json: %w", err)
	}
	return bundles, nil
}

// Apply registers every preset contract whose bundle chainId equals chainID.
// It builds each contract via abi.NewContractFromABI (validating the ABI) and
// hands it to adder.Add. Returns the number registered and the first error.
// A chainID with no matching bundle registers nothing and returns (0, nil).
func Apply(chainID int64, adder ContractAdder) (loaded int, err error) {
	bundles, err := allBundles()
	if err != nil {
		return 0, err
	}
	for _, b := range bundles {
		if b.ChainID != chainID {
			continue
		}
		for _, pc := range b.Contracts {
			info, ierr := abi.NewContractFromABI(pc.Name, pc.Symbol, pc.Address, []byte(pc.ABI))
			if ierr != nil {
				return loaded, fmt.Errorf("presets: contract %q (chain %d): %w", pc.Name, chainID, ierr)
			}
			info.Decimals = pc.Decimals
			adder.Add(info)
			loaded++
		}
	}
	return loaded, nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./presets/ -run TestApply -v`
Expected: PASS for `TestApply_ArcChainRegistersAllEight` and `TestApply_UnknownChainRegistersNothing`.

- [ ] **Step 5: Commit**

```bash
git add presets/presets.go presets/presets_test.go
git commit -m "feat(preset): presets package — embed bundle, Apply by chainId

Apply(chainID, adder) registers every preset contract whose bundle chainId
matches, via abi.NewContractFromABI (validates ABI) + Add, carrying decimals.
Non-matching chain registers nothing. TDD.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Idempotency + bad-ABI error coverage

**Files:**
- Modify: `presets/presets_test.go`

- [ ] **Step 1: Write the additional failing tests**

Append to `presets/presets_test.go`:

```go
func TestApply_IsIdempotentAcrossCalls(t *testing.T) {
	// Applying twice with the SAME adder must not error and must report 8 each
	// time. (The real registry dedups by address; the fake just records, so we
	// assert Apply itself is stable and side-effect-free beyond Add.)
	adder := &fakeAdder{}
	if n, err := Apply(5042002, adder); err != nil || n != 8 {
		t.Fatalf("first apply n=%d err=%v", n, err)
	}
	if n, err := Apply(5042002, adder); err != nil || n != 8 {
		t.Fatalf("second apply n=%d err=%v", n, err)
	}
	if len(adder.added) != 16 {
		t.Fatalf("added=%d want 16 (8+8; dedup is the registry's job)", len(adder.added))
	}
}

func TestApply_AllEmbeddedAbisAreValid(t *testing.T) {
	// Guards that every embedded ABI parses — a malformed bundle would make
	// Apply return an error mid-way.
	adder := &fakeAdder{}
	n, err := Apply(5042002, adder)
	if err != nil {
		t.Fatalf("an embedded ABI failed to import: %v", err)
	}
	if n != 8 {
		t.Fatalf("registered %d want 8", n)
	}
}

func TestApply_PreservesContractAddresses(t *testing.T) {
	adder := &fakeAdder{}
	if _, err := Apply(5042002, adder); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"JustifyAccessControl": "0xD45678caddB0301E8783D875a317C0cFEDa189Aa",
		"MockUSDC":             "0x8AAbb743F59dD725Cf16B06F53C1aDfB660969D8",
		"MarketFactory":        "0xd7f35035F6E6B42a1CB52e4AFc982896933cB20C",
		"MarketAMM":            "0xAbED8cB716ee30985df01b9641C1A01666b6c734",
	}
	got := map[string]string{}
	for _, c := range adder.added {
		got[c.Name] = c.ContractAddress
	}
	for name, addr := range want {
		if got[name] != addr {
			t.Fatalf("%s address=%q want %q", name, got[name], addr)
		}
	}
}
```

- [ ] **Step 2: Run to verify they pass (behavior already implemented in Task 2)**

Run: `go test ./presets/ -v`
Expected: all `TestApply_*` PASS. (These tests exercise already-correct behavior;
they lock it in. No production change needed — if any fails, fix `presets.go`.)

- [ ] **Step 3: Run the full package with race**

Run: `go test ./presets/ -race -count=1`
Expected: `ok  github.com/ITProLabDev/ethbacknode/presets`

- [ ] **Step 4: Commit**

```bash
git add presets/presets_test.go
git commit -m "test(preset): idempotency, address preservation, all-ABIs-valid

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Wire `presets.Apply` into main.go startup

**Files:**
- Modify: `main.go` (insert after line 146, the `log.Info("- Chain ID:", ...)` line)

The chain id is logged at `main.go:146`; insert the preset application right after
it, before the token loop / Address Manager init. Match on the numeric chainId
from `eth_chainId` via the existing `chainClient.GetNetId()`.

- [ ] **Step 1: Read the current context to confirm the insertion point**

Run: `sed -n '139,150p' main.go`
Expected to show:
```
	err = chainClient.Init()
	... (error check) ...
	log.Info("Blockchain Info:")
	log.Info("- Chain Name:", chainClient.GetChainName())
	log.Info("- Chain ID:", chainClient.GetChainId())
	for _, token := range chainClient.TokensList() {
		log.Info("- Token:", token.Name, "(", token.Symbol, ")")
	}
	// Init Address Manager
```

- [ ] **Step 2: Add the `presets` import**

Find the import block at the top of `main.go` and add this line alphabetically
among the `github.com/ITProLabDev/ethbacknode/...` imports:

```go
	"github.com/ITProLabDev/ethbacknode/presets"
```

- [ ] **Step 3: Insert the preset-application block**

Immediately after the line:
```go
	log.Info("- Chain ID:", chainClient.GetChainId())
```
insert:

```go
	// Apply any built-in contract preset for this chain (PROJECT_STATUS §6).
	// Matched on the numeric chainId from eth_chainId — the on-chain truth and
	// the key the deployment artifact uses. Non-fatal: a node without the preset
	// behaves as before. Idempotent: the registry dedups by address on restart.
	if netId, nerr := chainClient.GetNetId(); nerr != nil {
		log.Error("Can not read chain id for contract preset:", nerr)
	} else if loaded, perr := presets.Apply(netId, abiManager); perr != nil {
		log.Error("Can not apply contract preset:", perr)
	} else if loaded > 0 {
		log.Info("- Loaded contract preset:", loaded, "contracts")
	}
```

- [ ] **Step 4: Verify it builds**

Run: `go build ./...`
Expected: exit 0, no output.

- [ ] **Step 5: Verify `go vet` on main is clean (no new warnings)**

Run: `go vet . 2>&1 | grep -v "methods_info.go\|rpc_request.go" || echo "no new vet warnings"`
Expected: `no new vet warnings` (the pre-existing endpoint warnings are filtered out; the root package should be clean).

- [ ] **Step 6: Commit**

```bash
git add main.go
git commit -m "feat(preset): apply contract preset at startup by chainId

After chainClient.Init(), read the numeric chainId via eth_chainId and apply
the matching embedded preset through the abi registry. Non-fatal and idempotent.
Arc (5042002) gets its 8 prediction-market contracts; each automatically gains
contractEvent delivery, contractCall, and subscriptions.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Final verification

**Files:** none (verification only)

- [ ] **Step 1: Build the whole repo**

Run: `go build ./...`
Expected: exit 0.

- [ ] **Step 2: Race-test the new package and the packages it touches**

Run: `go test ./presets/ ./abi/ -race -count=1`
Expected: both `ok`.

- [ ] **Step 3: Confirm the three pre-existing failures are unchanged (not regressions)**

Run: `go test ./... 2>&1 | grep -E "^(FAIL|ok)" | grep FAIL`
Expected: only these three (pre-existing, unrelated, documented in PROJECT_STATUS):
```
FAIL	github.com/ITProLabDev/ethbacknode/common/rlp/rlpgen
FAIL	github.com/ITProLabDev/ethbacknode/crypto/secp256k1 [build failed]
FAIL	github.com/ITProLabDev/ethbacknode/uniclient
```
If any OTHER package fails, that is a regression — stop and fix.

- [ ] **Step 4: Update PROJECT_STATUS.md §6**

Modify `docs/PROJECT_STATUS.md` §6 to mark the preset delivered. Replace the
"Pending product request" body with a short RESOLVED note pointing at
`presets/arc.json` + `presets.Apply` wired in `main.go`, and noting the preset is
applied on the Arc chain (5042002) via the §6 `AddContractFromABI`/`Add` data path.

- [ ] **Step 5: Commit the docs update**

```bash
git add -f docs/PROJECT_STATUS.md
git commit -m "docs(preset): mark §6 built-in contract preset delivered (Arc)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-Review notes (for the implementer)

- **Spec coverage:** embed (Task 1+2), `Apply` by numeric chainId (Task 2), main wiring after Init (Task 4), non-fatal/idempotent/chain-isolated (Task 2+4), all 8 contracts with name=symbol and MockUSDC decimals=6 (Task 1+2), tests node-free (Task 2+3). All spec sections map to a task.
- **Decimals nuance:** `AddContractFromABI` does not carry decimals, so the loader uses `abi.NewContractFromABI` (same validation) + set `.Decimals` + `adder.Add` — still the §6 data-driven path, no hardcoded contract logic.
- **Type consistency:** `ContractAdder.Add(*abi.SmartContractInfo)` is satisfied by `*abi.SmartContractsManager` (its `Add` has that exact signature); the test `fakeAdder.Add` matches. `Apply(int64, ContractAdder) (int, error)` is used identically in tests and main.go (`abiManager` is `*abi.SmartContractsManager`).
- **`abiManager` availability in main.go:** declared at `main.go:102`, well before the insertion point at line 146.
