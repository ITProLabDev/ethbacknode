# Arc Prediction-Market Contract Preset — Design

**Date:** 2026-06-13
**Status:** Design (approved sections; pending written-spec review)
**Relates to:** PROJECT_STATUS §6 (pending built-in contract preset),
`todo/TASKS.md` Future Phases / M3 preset path.

## Goal

Generate a versioned, built-in **contract preset** from the Hardhat deployment
artifact `_sources/contracts/deployments/arc.json` (Circle Arc testnet,
chainId 5042002, a Polymarket-class prediction-market system of 8 contracts) and
load it into the abi registry **at startup**, so every preset contract
automatically gains `contractEvent` delivery, `contractCall`, and event
subscriptions — **without hardcoded Go contract logic**, exactly as PROJECT_STATUS
§6 requires.

## Why a preset (not the runtime registry file)

`data/abi/known_contracts.json` is **gitignored runtime state** (`.gitignore`
line 83 ignores `data`), is **not version-controlled**, is **regenerated from a
Go template by `ColdStart()`** when absent, and is also where RPC-registered
user contracts are written. Writing the Arc contracts directly there would not
ship with the binary, would be lost on a fresh deploy / cold start, and would mix
with user data. `_sources/` is also gitignored (line 93) — a working input, not a
repo artifact.

Therefore the preset is **embedded into the binary** with `//go:embed`. It is
version-controlled, survives cold start, ships with the binary, and is applied
through the §6 flow (`AddContractFromABI`). The `arcProfile()` comment already
anticipates this: *"No seed tokens — addresses are added via presets."*

## Architecture

```
presets/arc.json   (VERSION-CONTROLLED, //go:embed into the binary)
  { chainId: 5042002, source, generatedAt,
    contracts: [ {name, symbol, decimals, address, abi:[<canonical ABI array>]} × 8 ] }
        │  embedded — survives cold-start, ships with binary
        ▼
  presets.Apply(numericChainID, abiManager)      ← called from main.go AFTER chainClient.Init()
        │  loads ONLY the preset whose chainId == the live numeric chainId
        ▼
  for each contract: abiManager.AddContractFromABI(name, symbol, address, abiRaw)   ← §6 flow
        │  idempotent: registry dedups by address; re-runs are safe
        ▼
  each contract automatically gains contractEvent + contractCall + subscribe RPC
```

## Components

A new isolated package **`presets/`** (single responsibility: embed and apply
preset contracts; depends only on a narrow adder interface).

| File | Responsibility |
|------|----------------|
| `presets/arc.json` | Generated from `_sources/.../arc.json`. Top level: `chainId` (int, 5042002), `source` (provenance string), `generatedAt` (ISO date), `contracts` (array). Each contract: `name`, `symbol`, `decimals`, `address`, `abi` (the **canonical Ethereum ABI array**, verbatim from the artifact). |
| `presets/presets.go` | `//go:embed arc.json` + the bundle registry. `Apply(chainID int64, adder ContractAdder) (loaded int, err error)`: parse embedded presets, select the bundle whose `chainId == chainID`, and for each contract call `adder.AddContractFromABI(name, symbol, address, abiRaw)`. Declares its own narrow `ContractAdder interface { AddContractFromABI(name, symbol, address string, rawABI []byte) error }` (satisfied by `*abi.SmartContractsManager`; not imported from endpoint to avoid coupling). Returns the count applied and the first error. No match → `(0, nil)`. |
| `presets/presets_test.go` | TDD unit tests (no node needed). |

### Contract metadata mapping

| Field | Source |
|-------|--------|
| `name` | The contract name from the artifact (e.g. `MarketFactory`, `OutcomeToken`). |
| `symbol` | Same as `name` (no synthetic tickers). |
| `decimals` | `6` for `MockUSDC` (ERC-20); `0` for the rest (non-token; decimals unused for events/views). |
| `address` | The artifact address, verbatim (registry lowercases for lookup; checksum preserved for display). |
| `abi` | The artifact `abi` array, verbatim canonical Ethereum JSON ABI. |

The 8 contracts: `JustifyAccessControl`, `MockUSDC`, `OutcomeToken`,
`FeeTreasury`, `OracleResolver`, `MarketFactory` (the 6 core) +
`PredictionMarket`, `MarketAMM` (from `seededMarket`). All 8 addresses are unique.
None has tuple inputs or outputs, so `ImportEthereumABI` accepts every one.

## Integration point (main.go)

Inserted after `chainClient.Init()` (where the chain is connected and the numeric
chainId is fetchable). Matching is on the **numeric** chainId — the on-chain truth
and the exact key the artifact uses — obtained via the existing
`chainClient.GetNetId()` (`eth_chainId`). Note `GetChainId()` returns the logical
label (`"arc-testnet"`), not the number, so it is not used for matching.

```go
if netId, err := chainClient.GetNetId(); err == nil {
    if n, perr := presets.Apply(netId, abiManager); perr != nil {
        log.Error("Can not apply contract preset:", perr)
    } else if n > 0 {
        log.Info("- Loaded contract preset:", n, "contracts")
    }
}
```

## Error handling

- **Non-fatal.** A preset failure (malformed ABI, `eth_chainId` unavailable) is
  logged; the node continues to start. A node without the preset behaves exactly
  as before.
- **Idempotent.** On restart, `AddContractFromABI` → `Add` dedups by address, so
  re-applying the preset is a no-op (a harmless `[CRIT] Duplicated Contract` log
  line may appear — pre-existing registry behavior, see PROJECT_STATUS §5).
- **Chain isolation.** A preset is applied only when its `chainId` matches the
  live chain, so the Arc preset never loads on an Ethereum node.

## Preset generation (one-time, now)

`presets/arc.json` is generated **once** from `_sources/.../arc.json` (by a
throwaway transformation during this work) and committed. It is NOT a runtime
dependency on `_sources/` (which is gitignored). Re-generation for a future
deployment is a manual repeat of the same transform; a generator tool is out of
scope for this iteration (YAGNI — one deployment artifact today).

## Testing

TDD on `presets/` (unit, node-free):
1. embedded `arc.json` parses; bundle has chainId 5042002 and 8 contracts.
2. `Apply(5042002, fakeAdder)` registers all 8 (asserts names/addresses passed).
3. `Apply(1, fakeAdder)` registers 0 (no matching bundle).
4. idempotent: applying twice does not error.
5. a contract with malformed ABI surfaces as an error from `Apply` (via a fake
   adder that returns the abi import error), and the count reflects partial work.

The end-to-end delivery path (`AddContractFromABI` → decode → `contractEvent`) is
already covered by `endpoint/e2e_contract_event_test.go`; no new e2e needed.

Verification: `go build ./...`, `go test ./presets/ ./abi/ -race`, and a build of
`main` to confirm the wiring compiles.

## Out of scope

- A reusable Go generator/CLI (one artifact today; manual transform suffices).
- Loading presets from an external config-specified path (embed chosen).
- Per-contract `decimals`/`symbol` beyond the simple mapping above.
- Auto-discovering dynamically-created markets (factory-spawned
  `PredictionMarket`/`MarketAMM` instances beyond the seeded one) — a client
  registers those via `contractRegister` at runtime.
