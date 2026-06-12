# Universal Smart-Contract Layer — TASKS

**Status:** Design / planning. **Implementation NOT started.**
**Goal:** extend EthBackNode to work with arbitrary smart contracts, with
Polymarket-class contracts (Conditional Tokens Framework / ERC-1155, CLOB
exchange, USDC collateral) as the driving use case.

---

## Scope & Decisions

| Decision | Choice |
|----------|--------|
| Depth | **Universal contract layer.** Polymarket is a special case of the generic engine, not hardcoded. |
| ABI implementation | **Own implementation only.** Extend `abi/`; **never** migrate to go-ethereum or any external ABI library. Standard Polygonscan/solc JSON ABI is supported via a converter into the project's own format. |
| Read vs Write | **Read + decode + encode call-data only.** Decoding event logs and input data, encoding call-data, `eth_call` for view methods, reading receipt logs in watchdog. Signing/broadcasting arbitrary write calls is a **later phase, out of scope here.** |
| Log collection | **Both modes, switchable by config:** (1) `eth_getLogs` per block filtered by registered contract addresses/topics; (2) per-receipt via `eth_getTransactionReceipt` for relevant txs. |
| Architecture | **Approach A.** Keep `abi/` as a pure stateless codec library. Add a new service package (`eventlog/`) that reads + decodes logs and plugs into watchdog as an additional listener. |
| Concurrency | I/O-bound paths use a typed pipeline layer **`tools/flow`** (bounded, order-preserving fan-out) built over `go_pysyun_pipeline`. See [`PIPELINE-FLOW.md`](./PIPELINE-FLOW.md). |

### Out of scope (this phase)
- Signing / broadcasting arbitrary contract write methods (`splitPosition`,
  `mergePositions`, `redeemPositions`, order placement, etc.).
- EIP-712 typed-data signing for CLOB orders.
- EIP-1559 transaction type.
- Polymarket domain modeling beyond decoding (markets, outcomes, P&L).

These are recorded in the **Future Phases** section so they are not lost.

---

## Current State (gaps found in codebase)

- `abi/` engine decodes/encodes **only** static `address`, `uint256`/`int256`,
  `bool`. No head/tail dynamic ABI encoding, no `bytesN`/`bytes`/`string`,
  no arrays, no tuples, no event-log decoding.
- Method selectors are 4-byte Keccak (`updateSignature`); event topic0 hashing
  (full 32-byte Keccak of the event signature) is not implemented.
- `clients/ethclient/methods.go`: `eth_getTransactionReceipt` is declared as a
  const but **`GetTransactionReceipt` is not implemented**. `eth_getLogs` does
  not exist. No receipt/log Go types.
- `clients/ethclient/decode_transaction.go` recognizes only native ETH transfer
  and ERC-20 `transfer`; everything else returns `ErrUnsupportedTransactionType`.
- `watchdog/processtx.go` fires events **only** when `tx.From`/`tx.To` is a
  managed address; receipt logs are never read or decoded.
- `eth_call` (`Call`/`CallByBlockNumber`) already exists — the read foundation.
- ERC-20 contract registry (`SmartContractsManager`) exists and is the natural
  home for a generalized contract registry.

---

## Milestone 1 — ABI engine: dynamic types & tuples (`abi/`)  ✅ DONE

Pure codec work. No service dependencies. Fully unit-testable.

**Status:** Complete. Typed engine added alongside the legacy static codec
(`abi/abitype.go`, `abi/value.go`, `abi/codec.go`; wiring in
`abi/smartcontractabi.go`). The canonical Solidity `baz` and `sam` vectors pass
byte-for-byte; ERC-20 `transfer` selector `0xa9059cbb` preserved. Untrusted
32-byte counts/offsets are guarded with `big.Int.IsInt64()` + overflow-safe
bounds. `go test ./abi/ -race` clean (49 tests), `go build ./...` and
`go vet ./abi/` clean. See `docs/superpowers/plans/2026-06-11-m1-abi-dynamic-types.md`.

- [x] **M1.1** Introduce head/tail ABI encoding/decoding with offset handling.
      Today every param is read as a flat 32-byte slot; dynamic types require
      a head section (static slots + offsets) and a tail section (dynamic data).
- [x] **M1.2** Add static integer widths: `uintN` / `intN` for N in 8..256
      (step 8). Decode/encode right-aligned, 32-byte slots.
- [x] **M1.3** Add fixed bytes `bytes1`..`bytes32` (left-aligned) and `bytesN`
      validation.
- [x] **M1.4** Add dynamic `bytes` and `string` (length-prefixed, 32-byte padded).
- [x] **M1.5** Add fixed arrays `T[N]` and dynamic arrays `T[]` for supported
      element types.
- [x] **M1.6** Add `tuple` / struct support (nested `components`), including
      dynamic tuples. Required for CLOB order structs.
- [x] **M1.7** Extend the neutral decoded model:
      ```go
      type DecodedValue struct { Name, Type string; Value any }
      type DecodedCall  struct { Method string; Inputs []DecodedValue }
      type DecodedEvent struct { Name, Contract string; Inputs []DecodedValue }
      ```
      `Value` is one of `*big.Int | []byte | string | bool | []DecodedValue`.
- [x] **M1.8** Unit tests for every type against canonical ABI test vectors
      (encode↔decode round-trips, oversized/short input must error not panic —
      follow the existing `abi_test.go` style).

## Milestone 2 — ABI engine: event-log decoding (`abi/`)  ✅ DONE

**Status:** Complete. `abi/event.go` adds `Topic0()` (full 32-byte event
signature hash), `(*SmartContractAbiEntry).DecodeLog(topics [][32]byte, data
[]byte)` (splits indexed params from `topics[1:]` and non-indexed from `data`;
indexed reference types — arrays/tuples/string/bytes — return a 32-byte keccak
hash placeholder per the Solidity spec), `(*SmartContractAbi).GetEventByTopic0`,
and `(*SmartContractsManager).DecodeLog(contractAddress, topics, data)`.
Validated against real mainnet topic0 vectors (ERC-20 `Transfer`, ERC-1155
`TransferSingle`) and ERC-1155 `TransferBatch` decode. `go test ./abi/ -race`
clean, `go build ./...` and `go vet ./abi/` clean. Carry-over perf note (cached
topic0 map) recorded under Milestone 5. See
`docs/superpowers/plans/2026-06-12-m2-event-log-decoding.md`.

> **Carry-over notes from the M1 final review (applied during M2):**
> - **Indexed dynamic params** (e.g. `string`/`bytes`/arrays marked `indexed`)
>   appear in topics as their Keccak hash, NOT the original value — they are not
>   recoverable. Do NOT blindly reuse `decodeParams` for indexed args; decode
>   indexed (from `topics[1:]`) and non-indexed (from `data`) separately, and
>   represent un-recoverable indexed dynamics as a hash placeholder.
> - `DecodedEvent` (in `abi/value.go`) is already defined and ready to populate.
> - The typed engine accepts a per-kind set of Go input types for encode
>   (`bool`→bool, address/bytesN/bytes→[]byte, string→string, uint/int→
>   *big.Int|int|int64|uint64); decode emits *big.Int|bool|[]byte|string|
>   []DecodedValue. Code higher layers against these concrete types.
> - If you construct `abiType` values by hand (not via `parseType`), set `elem`
>   for array/slice kinds or add a nil guard — `isDynamic()`/`staticSize()`
>   deref `t.elem`.

- [x] **M2.1** Implement event topic0 hashing: 32-byte
      `keccak256("EventName(type1,type2,...)")` (vs the 4-byte method selector).
      Extend `updateSignature` with an event branch.
- [x] **M2.2** `DecodeLog(topics [][]byte, data []byte) (*DecodedEvent, error)`
      on `SmartContractAbi`: split `indexed` params from `topics[1:]` and
      non-indexed params from `data`. Handle `anonymous` events (no topic0).
- [x] **M2.3** Registry lookup `GetEventByTopic0([32]byte) (*SmartContractAbiEntry, error)`.
- [x] **M2.4** Unit tests with real ERC-1155 `TransferSingle`/`TransferBatch`
      and CTF `PositionSplit` / `ConditionResolution` log fixtures.

> **Carry-over from M2 final review (for M3/M4/M5):**
> - **Anonymous events** are unsupported (no `anonymous` field in the model);
>   they fail closed to `ErrUnknownEvent`. If a target contract uses anonymous
>   events, add an `anonymous` field + an alternate decode path (indexed params
>   start at `topics[0]`, no signature topic).
> - The indexed-reference placeholder `Type` string (`"… (indexed)"`) is
>   display-only and NOT re-parseable — do not feed it back into `parseType`.
> - Decode is concurrency-safe today; `_prepare`/`sync.Once` writes `Signature`
>   on first call — keep any future per-entry caching behind the same `Once`
>   (or read-only post-prepare) to stay race-free under watchdog fan-out.

## Milestone 3 — Standard ABI import & generalized registry (`abi/`)

- [ ] **M3.1** `abi/import.go`: `ImportEthereumABI(raw []byte) (*SmartContractAbi, error)`
      converting canonical solc/Polygonscan JSON (lowercase `type`,
      `stateMutability`, `components`, `anonymous`) into the project's own
      format. This removes the manual-conversion pain of the custom format.
- [ ] **M3.2** Generalize `SmartContractInfo` to register **any** contract
      (not just tokens) with its ABI; keep ERC-20 path working (regression).
- [ ] **M3.3** Seed registry entries / fixtures for Polymarket on Polygon:
      Conditional Tokens (ERC-1155), CTF Exchange, USDC. ABIs imported via M3.1.
- [ ] **M3.4** Backward-compat: existing `known_contracts.json` and ERC-20
      template must still load. Tests.

## Milestone 4 — Client: receipts & logs (`clients/ethclient/`)

- [ ] **M4.1** Implement `GetTransactionReceipt(txHash) (*Receipt, error)`
      (the const already exists; the method does not). Add `Receipt`/`Log` Go
      types with the project's hex-decoding `UnmarshalJSON` style.
- [ ] **M4.2** Implement `eth_getLogs`: `GetLogs(filter LogFilter) ([]*Log, error)`
      with fromBlock/toBlock, address list, topics.
- [ ] **M4.3** Generalize `eth_call`-based reads: a `CallMethod(contract,
      method, args...) ([]DecodedValue, error)` helper using the ABI engine
      (e.g. ERC-1155 `balanceOf(addr,id)`, `balanceOfBatch`).
- [ ] **M4.4** Tests against recorded JSON-RPC fixtures (IPC/HTTP).

## Milestone 5 — Event-log service (`eventlog/`, new package)

> **Carry-over from M2 review (perf, apply when log volume matters):**
> `(*SmartContractAbi).GetEventByTopic0` is O(entries) per log and recomputes
> `Topic0()` (keccak) for every entry on every call. For high-volume log
> decoding (Mode A/B), build a cached `map[[32]byte]*SmartContractAbiEntry` in
> `_prepare()` and look up by topic0 in O(1). Not needed for correctness; only
> if profiling shows it hot.

- [ ] **M5.1** New package `eventlog/` (Approach A): consumes a chain client +
      the ABI registry, decodes logs into `DecodedEvent`, and exposes a
      listener handler. `abi/` stays a pure library with no service deps.
- [ ] **M5.2** Mode A — `eth_getLogs` per block: one call per block filtered by
      registered contract addresses/topics; independent of tx count.
- [ ] **M5.3** Mode B — per-receipt: fetch `eth_getTransactionReceipt` for
      relevant txs and decode their logs. Use `tools/flow.FanOut` for bounded,
      order-preserving parallel receipt fetching — see [`PIPELINE-FLOW.md`](./PIPELINE-FLOW.md).
- [ ] **M5.4** Config flag to switch modes (default = `getLogs`). Document
      load/latency trade-offs.
- [ ] **M5.5** Watchdog integration: register `eventlog` as an additional
      listener in the block-processing path (`watchdog/processblock.go`)
      without changing the existing From/To tx matching.
- [ ] **M5.6** Match logs against managed addresses inside indexed params
      (e.g. ERC-1155 `from`/`to`/`operator`), not just `tx.To`.
- [ ] **M5.7** **Track ALL on-chain events of a registered contract** (user
      requirement 2026-06-12): subscribe to a contract's full event stream by
      contract address (+ optional topic filter), decode every matching log via
      `abi.DecodeLog`, and deliver it — independent of whether the tx touches a
      managed address. This is broader than M5.6's managed-address matching;
      both modes coexist (per-contract full stream AND per-address filtering).
      A registered preset contract (see contract-preset note) auto-enables full
      event tracking.

## Milestone 6 — Delivery & API (`subscriptions/`, `endpoint/`)

> **Extensibility requirement (user, 2026-06-12):** the notification system and
> the JSON-RPC endpoint must be EXTENSIBLE — adding a new event/notification
> type or a new contract preset must be data/registration-driven, not a core
> rewrite. Design `contractEvent` + the registration/subscription RPC surface
> so new event types and presets plug in without touching the dispatch core.

- [ ] **M6.1** New notification type `contractEvent` delivered to subscribers via
      the existing JSON-RPC 2.0 callback mechanism (alongside `blockEvent` /
      `transactionEvent`). Make the notification dispatch table extensible so
      future event types register rather than require new switch arms.
- [ ] **M6.2** RPC methods: register a contract + ABI, list registered
      contracts, subscribe an address/contract to contract events (incl.
      "all events of contract X"). Register via `AddRpcProcessor` following
      existing `methods_*.go` patterns.
- [ ] **M6.3** Read-only RPC: `contractCall` to invoke a view method by
      name+args and return decoded outputs.
- [ ] **M6.4** Persist registered contracts/subscriptions in existing storage
      (`data/abi/`, `data/subscriptions/`).

## Milestone 7 — Docs & tests

- [ ] **M7.1** Update `DOC.md` (package table, interfaces, config, new RPC
      methods + `contractEvent` notification).
- [ ] **M7.2** Update `API.md` with new methods and event payloads.
- [ ] **M7.3** End-to-end test: register a generic contract-class ABI
      (multitoken / outcome-market / CLOB fixtures), replay a block with known
      logs, assert decoded `contractEvent` output for ALL the contract's events.
- [ ] **M7.4** `go test ./...` and `-race` clean.

---

## Future Phases (recorded, out of current scope)

- **Write support:** sign + broadcast arbitrary contract methods from a managed
  address (generalize `send_methods.go`), gas/nonce handling for contract calls.
- **EIP-712** typed-data signing for CLOB orders (`OrderFilled`/`OrdersMatched`).
- **EIP-1559** transaction type support.
- **CTF write methods:** `splitPosition`, `mergePositions`, `redeemPositions`.
- **Polymarket domain model:** markets, conditions, outcome tokens, position P&L.

---

## Suggested order

M1 → M2 → M3 (pure `abi/`, no external deps, highest-leverage foundation) →
M4 (client receipts/logs) → M5 (eventlog service + watchdog) →
M6 (delivery + API) → M7 (docs + tests).