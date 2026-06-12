# EthBackNode — Universal Contract Layer: Project Status & Handoff

> **Purpose of this file:** a single, self-contained snapshot to resume work from
> any device. It captures the goal, every milestone's state, the exact branch &
> commits, carry-over notes, and the next concrete steps. Last updated after M6.

**Last updated:** 2026-06-13
**Branch:** `feature/ipc-pool-and-flow` (NOT yet merged to `main`)
**HEAD at last update:** `acfe91f` + uncommitted M7 work (docs + format fix + e2e test + subscribe guard)
**Build/test health:** `go build ./...` clean; `-race` green for ALL packages
except three with **pre-existing, unrelated** failures (proven on the clean base
via `git stash`): `crypto/secp256k1` (env build issue — stdlib `ecdsa.Sign`
arity), `common/rlp/rlpgen` (missing `testdata/*.in.txt` fixtures), and
`uniclient` `TestClient` (needs a live node). The 3 `go vet` warnings in
`endpoint/methods_info.go` + `endpoint/rpc_request.go` also pre-date this work.

---

## 1. The Goal (what we're building & why)

Extend EthBackNode (an Ethereum backend microservice exposing JSON-RPC 2.0) with
an **abstract, generic any-contract description layer**: describe arbitrary smart
contracts via their ABI, then interact with them and receive their info & events
from the blockchain. "Polymarket-like" only denotes the *complexity level*
(multi-token ERC-1155, tuple/struct orders, rich events) — this is **NOT** a
Polymarket-specific integration.

### Hard architectural constraints (do not violate)
- **Self-built ABI engine.** The `abi/` package MUST stay custom. It may be
  extended, but NEVER migrate to go-ethereum or any external ABI/RPC library.
  (Memory: `own-abi-no-eth-libs`.)
- **Decoupling.** The `endpoint` package must NOT import `clients/ethclient`; it
  talks to chain/abi/eventlog through narrow interfaces wired in `main.go`.
  `eventlog/` must NOT import `clients/ethclient` either (it has its own
  `LogSource`/`Decoder` seams + adapters).
- **Extensibility (user requirement, 2026-06-12).** Adding a new event/notification
  type or a new contract preset must be data/registration-driven, not a
  dispatch-core rewrite.

---

## 2. Milestone Status — at a glance

| Milestone | Title | Status |
|-----------|-------|--------|
| M1 | ABI dynamic types (bytes/string, head/tail codec) | ✅ DONE |
| M2 | Event-log decoding (topic0, indexed reference-type hashing) | ✅ DONE |
| M3 | Standard ABI import & generalized registry | ✅ DONE |
| M4 | Client: receipts / logs / `CallMethod` | ✅ DONE |
| M5 | Event-log service (`eventlog/` package) | ✅ DONE |
| M6 | Delivery & API (contractEvent + contract RPC) | ✅ DONE |
| **M7** | **Docs & end-to-end tests** | **✅ DONE** |
| Side | IPC connection pool | ✅ DONE |
| Side | `tools/flow` typed Envelope over go_pysyun_pipeline | ✅ DONE |
| Side | `--init Eth\|Arc` chain-profile bootstrap flag | ✅ DONE |

Full task detail with status blocks lives in **`todo/TASKS.md`** (the canonical
tracker). This file is the cross-device summary.

---

## 3. M6 — Delivery & API (just completed)

**What it delivers:** decoded contract events are pushed to subscribers as a
`contractEvent` JSON-RPC callback, plus RPC methods to register a contract+ABI,
list contracts, subscribe/unsubscribe to a contract's events with a `scope`, and
call a read-only view method.

### Delivery path (end-to-end, verified by final review)
```
chain log
  → eventlog.Service.OnBlock (watchdog block listener)
  → collect (eth_getLogs for subscribed addresses)
  → decodeAndDeliver → abiManager.DecodeLog + eventMatchesScope
  → sink(&ContractEvent{ServiceID: sub.ServiceID, ...})        // ServiceID threaded in (M6 Task 1)
  → endpoint.NewContractEventSink: strconv.ParseInt(ServiceID) // string → int64
  → endpoint.NotifierFunc closure (wired in main.go)
  → subscriptions.Manager.NotifySubscriberRaw(ServiceId(int64))
  → subscriber.sendNotification("contractEvent", *contractEventPayload)
```
ServiceID round-trip (`int → decimal string in eventlog → int64 at sink → ServiceId`)
is lossless for any positive id; a non-numeric id is logged and skipped, never panics.

### RPC methods added (in `endpoint/`, both `dot.case` + `camelCase` aliases)
| Method | Secured? | Purpose |
|--------|----------|---------|
| `contract.register` / `contractRegister` | 🔒 secured | register a contract + canonical JSON ABI |
| `contract.subscribe` / `contractSubscribe` | 🔒 secured | subscribe a serviceId to a contract's events with `scope` |
| `contract.unsubscribe` / `contractUnsubscribe` | 🔒 secured | remove a subscription |
| `contract.list` / `contractList` | open | list registered contracts (name→address) |
| `contract.subscriptions` / `contractSubscriptions` | open | list active event subscriptions |
| `contract.call` / `contractCall` | open | read-only view-method call, returns decoded outputs |

- **Secured = `RegisterSecuredProcessor`** (requires `serviceId` + API token; the
  subscriber must already exist — you subscribe an existing service to contract
  events). Decision made with the user on 2026-06-12.
- **`scope`** = `whole_contract` (all events) or `managed_only` (only events
  involving a managed address, matched inside the event's address-typed params).
  Chosen per-subscription at subscribe time.
- **`contractCall` is no-arg in this first cut** (e.g. `totalSupply`, `decimals`).
  Passing `args` returns a clear "not yet supported" error rather than
  mis-encoding JSON → ABI. **Typed-argument encoding from JSON is a deliberate
  follow-up** (good first task for a future milestone).

### Persistence
- **Registered contracts**: durable for free — the abi manager `Save()`s on `Add`
  (`data/abi/known_contracts.json`).
- **Event subscriptions**: persisted to `data/eventlog/subscriptions.json`,
  reloaded on startup via `LoadSubscriptions()`. Concurrent saves serialized by a
  `saveMu`. `LoadSubscriptions` is append-only (safe for one-time startup load).
  On restart, loaded subscriptions resume delivery (OnBlock reads `subs.addresses()`).

### Decoupling seams (narrow interfaces in `endpoint/`, concrete types wired in main.go)
- `ContractRegistry` (`GetSmartContractList`) ← `*abi.SmartContractsManager`
- `ContractAdder` (`AddContractFromABI`) ← `*abi.SmartContractsManager` (passed twice to `WithAbiManager(reg, adder)`)
- `EventSubscriber` (`SubscribeAndSaveStrings` / `UnsubscribeStrings` / `ListSubscriptions`) ← `*eventlog.Service`
- `ContractCaller` (`CallMethod`) ← `*ethclient.Client`
- `ContractEventNotifier` (`NotifyContractEvent`) ← `NotifierFunc` over `subscriptions.Manager.NotifySubscriberRaw`

### M6 commits (on top of base `8202ac9`)
```
9da89a7 feat(eventlog): tag ContractEvent with matched subscription ServiceID
8ef4137 feat(eventlog): persist subscriptions to storage (load/save JSON)
4b65ff8 fix(eventlog): serialize subscription saves + cover load error paths
34c7639 feat(endpoint): contractEvent delivery sink routing to subscribers
f0110d3 feat(endpoint): contract register/list/subscribe/unsubscribe RPC methods
5a37165 refactor(endpoint): move test-only errBadScope to test, clarify subscribe doc
07ffa4f feat(endpoint): contractCall read-only view-method RPC
e1c9187 test(endpoint): cover contractCall args-not-supported branch
67e19f2 feat(m6): wire contractEvent delivery + contract RPC into main
080e655 refactor(main): extract eventlog storage var for consistency
6cc69c5 docs(m6): mark M6 (delivery & API) complete
```

### M6 file inventory (new unless noted)
```
eventlog/event.go              (modified: +ServiceID field)
eventlog/eventlog.go           (modified: +ServiceID threading, +subStorage, +saveMu)
eventlog/subscriptions.go      (modified: +all(), +scopeName(), +remove())
eventlog/persist.go            (new: WithSubscriptionStorage, SubscribeAndSave, LoadSubscriptions)
eventlog/rpc_surface.go        (new: SubscribeAndSaveStrings, UnsubscribeStrings, ListSubscriptions)
eventlog/{persist,rpc_surface,eventlog}_test.go
abi/register.go                (new: AddContractFromABI) + register_test.go
endpoint/contract_delivery.go  (new: ContractEventNotifier, contractEventPayload, NewContractEventSink, NotifierFunc)
endpoint/methods_contracts.go  (new: register/list/subscribe/unsubscribe/listSubscriptions processors)
endpoint/methods_contract_call.go (new: contractCall processor)
endpoint/rpc_handler.go        (modified: +5 deps/interfaces/options)
endpoint/rpc_init.go           (modified: +12 registrations)
endpoint/{contract_delivery,methods_contracts,methods_contract_call}_test.go
subscriptions/notifysubscriber.go (modified: +NotifySubscriberRaw)
main.go                        (modified: delivery sink wiring, LoadSubscriptions, NewBackRpc options)
```

Plan: `docs/superpowers/plans/2026-06-12-m6-delivery-and-api.md`.

---

## 4. M7 — Docs & tests (just completed)

**What it delivers:** complete client-facing documentation of the contract layer,
a client-safe wire format for decoded values, and a real end-to-end test.

### Pre-doc format fix (client-safe decoded values)
Before documenting, the `contractEvent` / `contractCall` decoded-value wire form
was found to be hostile to clients and fixed via `abi.DecodedValue.MarshalJSON`
(`abi/value.go`, TDD in `abi/value_test.go`):
- **byte types** (`address`, `bytesN`, `bytes`) → `0x`-lowercase-hex strings
  (was base64 — a 20-byte address serialized as `"AAA…AAE="`).
- **big integers** (`uint*`/`int*`) → **decimal STRINGS** (was bare JSON numbers,
  which JS `JSON.parse` silently corrupts above 2^53 — i.e. most token amounts).
- bool/string native; arrays/tuples (`[]DecodedValue`) recurse. The in-memory
  `Value` type is unchanged; only JSON output is transformed. Decision made with
  the user 2026-06-13 ("починить формат, потом документировать").

### Subscribe-before-register guard (M6 carry-over RESOLVED)
`contract.subscribe` now rejects an unregistered contract address
(`unknown contract: register its ABI before subscribing`) via the new
`abi.IsContractKnown` (checksum-tolerant) exposed on the endpoint
`ContractRegistry` seam — mirrors the legacy ERC-20 `_isTokenKnown` guard but
keyed on contract address. Decision made with the user 2026-06-13.

### Docs written
- **`API.md`**: a full **Smart Contract Layer** section — quick-start, scope
  semantics, all 6 methods (params/results/errors, secured vs open, both name
  aliases), the `contractEvent` notification payload, and a **Decoded value
  format** subsection spelling out the JS-safe encoding + indexed-reference
  caveat. Overview lists + event-notifications list updated.
- **`DOC.md`**: contract method table, `contractEvent` flow under Event System,
  `eventlog` package row, and the `data/eventlog/subscriptions.json` dir entry.
- **`README.md`**: contract layer moved from "Planned" to "implemented"; API
  bullets + roadmap updated; stale "implementation not started" flow note removed.

### End-to-end test
`endpoint/e2e_contract_event_test.go` (package `endpoint_test`): registers a
generic multitoken-class ABI in the **real** `abi.SmartContractsManager`, replays
a block's `TransferSingle` log through the **real** `eventlog.Service` +
`NewDecoder(abiManager.DecodeLog)` + **real EIP-55 codec** + `NewContractEventSink`,
and asserts the delivered `contractEvent` **wire JSON** for both scopes —
`whole_contract` (asserts `to` is `0x`-hex and `value`/`id` are decimal strings,
locking in the format fix) and `managed_only` (asserts the **checksummed
managed-address invariant** matches, and does NOT match when no managed address
is involved). The managed set is a faithful exact-string stand-in for
`address.Manager` (whose `LookupString` is exact-byte) to avoid the pool's
async Badger init in a unit test; the codec — the actual source of the invariant
— is the real one.

---

## 5. Carry-over / known items

### From M6 final review (Minor — polish, not blockers)
- ~~**Subscribe-before-register is silent:**~~ **RESOLVED (2026-06-12).**
  `contract.subscribe` now rejects an address with no registered ABI
  (`"unknown contract: register its ABI before subscribing"`, INVALID_REQUEST)
  instead of silently accepting it. Implemented via `abi.IsContractKnown`
  (checksum-tolerant, reuses `GetSmartContractByAddress`) exposed on the
  endpoint `ContractRegistry` seam and guarded in `rpcProcessContractSubscribe`
  — mirrors the legacy ERC-20 `_isTokenKnown` guard, keyed on contract address.
- **`main.go` `eventlog.WithConfig(eventlog.DefaultConfig())` is redundant** —
  `New()` already defaults the config. Can drop for clarity.

### Pre-existing (NOT introduced by M6 — fix opportunistically, out of scope)
- **`subscriptions` `s.rpc` lazy-init race:** in `subscriptions/subscriptions.go`,
  the per-`Subscription` rpc client is lazily initialized under only an RLock, so
  concurrent deliveries to the SAME subscriber can race. Predates M6. Fix by
  guarding the lazy init or building `rpc` at subscribe time.
- **`abi.addUnsafe` duplicate check is case-sensitive** while `byAddress` is keyed
  lowercase — registering the same contract once checksummed and once lowercase
  bypasses the dup check (source of the harmless `[CRIT] Duplicated Contract` log
  lines in tests). Pre-existing abi behavior.

### From M5 final review (still relevant)
- **Receipts mode (`ModeReceipts`) is implemented + tested but NOT wired in
  main.go** — default is `ModeGetLogs`. To enable it, wire the `blockTxHashes`
  seam from the chain client and add a config knob. `TransactionIndex` is already
  on `ContractEvent` for ordering.
- **`managed_only` checksummed-address invariant:** matching works because both
  the managed pool and `eventMatchesScope` use the SAME codec
  (`EncodeBytesToAddress`, EIP-55). M6 didn't break this; the end-to-end test in
  M7 should assert it with a real checksummed managed address.
- **M2 perf carry-over:** `GetEventByTopic0` is O(entries) and recomputes keccak
  `Topic0()` per entry per call — fine for now; cache topic0 if log volume grows.

---

## 6. Pending product request (not yet started)

**Built-in contract preset (user, 2026-06-12):** the user will provide a specific
contract to integrate as a built-in preset. It must be integrated via import
path / ABI data (the existing `AddContractFromABI` registration flow), **NOT**
hardcoded Go. When it arrives: register its ABI as a preset loaded at startup,
and it automatically gains contractEvent delivery + contractCall + subscription
RPC. (Memory: `contract-preset-pending`.)

---

## 7. Working conventions (how this project is being built)

- **subagent-driven-development:** fresh implementer subagent per task →
  spec-compliance review → code-quality review → fix loop → final whole-impl
  review. TDD throughout (RED → GREEN → commit).
- **Plans** live in `docs/superpowers/plans/YYYY-MM-DD-<feature>.md`.
- **Commit trailer:** `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.
- **Chain profiles:** Ethereum, and Circle **Arc** testnet (chainId 5042002,
  native token USDC, 18 decimals) via the `--init Eth|Arc` flag (never overwrites
  existing config).
- **go_pysyun_pipeline** (LGPL-2.1) is used under `tools/flow` as a typed
  `Envelope` layer — credited in README per the author's request.

---

## 8. Resume checklist (from a fresh device)

1. `git checkout feature/ipc-pool-and-flow && git pull` (HEAD should be `6cc69c5` or later).
2. `go build ./...` and `go test ./eventlog/ ./endpoint/ ./abi/ -race` → expect green.
3. Read `todo/TASKS.md` (canonical tracker) + this file.
4. Start **M7** (§4), honoring the carry-overs in §5.
5. If the user has supplied the contract for §6, integrate it as a preset via
   `AddContractFromABI`.
