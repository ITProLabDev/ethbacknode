# Universal Smart-Contract Layer — TASKS

**Status:** Read-only phase **COMPLETE** — M1–M7 all done (2026-06-13). Write
support / typed `contractCall` args / EIP-712 remain in Future Phases.
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

## Milestone 3 — Standard ABI import & generalized registry (`abi/`)  ✅ DONE

**Status:** Complete. `abi/import.go` adds `ImportEthereumABI(raw)` (accepts the
canonical top-level JSON-array form AND the project's `{entries:[...]}` object
form; validates every input/output type via the typed engine; rejects null
entries and unrepresentable tuple outputs) and `NewContractFromABI(name,symbol,
address,rawABI)`. The registry was already generic over arbitrary contracts —
proven by a non-token round-trip test (register → lookup → decode event via
`manager.DecodeLog`). Generic role-based ABI fixtures under `abi/testdata/`
(multitoken ERC-1155, outcome-market, CLOB order tuple) prove the importer
handles real-world complexity (dynamic arrays, `bytes32`, 9-field tuples, mixed
indexed/non-indexed events) — these are GENERIC class representatives, NOT any
deployed contract. Backward-compat verified (cold start, legacy ERC-20
detect/decode, project template import). `go test ./abi/ -race` clean,
`go build ./...` and `go vet ./abi/` clean. See
`docs/superpowers/plans/2026-06-12-m3-standard-abi-import.md`.

- [x] **M3.1** `abi/import.go`: `ImportEthereumABI(raw []byte) (*SmartContractAbi, error)`
      converting canonical solc/Etherscan JSON (lowercase `type`,
      `stateMutability`, `components`, `anonymous`) into the project's own
      format. This removes the manual-conversion pain of the custom format.
- [x] **M3.2** Generalize the registry to register **any** contract (not just
      tokens) with its ABI; `NewContractFromABI` constructor; ERC-20 path kept
      working (regression). The registry was already generic — confirmed by test.
- [x] **M3.3** Generic contract-class ABI fixtures (abstract representatives of
      the class, NOT a deployed product): multitoken ERC-1155, outcome-market,
      CLOB order-book with a tuple order. Imported via M3.1; prove arrays,
      `bytes32`, tuples, and rich events all import.
- [x] **M3.4** Backward-compat: existing `known_contracts.json` and ERC-20
      template must still load. Tests.

## Milestone 4 — Client: receipts & logs (`clients/ethclient/`)  ✅ DONE

**Status:** Complete. `clients/ethclient/types_receipt.go` adds `Log` and
`Receipt` types (proxy-map 0x-hex `UnmarshalJSON`, nested logs, `Log.Topics32()`
bridge to `[][32]byte`, `Receipt.Success()`). `methods.go` adds
`GetTransactionReceipt` (null → `ErrTransactionNotFound`) and `eth_getLogs` via
`GetLogs(LogFilter)`. `abi/output.go` adds `(*SmartContractAbiEntry).DecodeOutputs`
(decode `eth_call` return data by a method's outputs). `call_method.go` adds the
generic `(*Client).CallMethod(contract, method, args...) ([]abi.DecodedValue, error)`
— look up registered contract → encode call-data → `eth_call` → decode outputs,
tying M1–M5 together. A `urpc.NewClientWithTransport` test seam enables
node-free RPC tests. `go test -race` (ethclient/urpc/abi) clean, `go build ./...`
and `go vet` clean. Tuple OUTPUT decoding remains deferred (output model has no
components). See `docs/superpowers/plans/2026-06-12-m4-client-receipts-logs.md`.

> **Carry-over from M3 final review (for M4/M5/M6):**
> - **Tuple OUTPUT decoding is deferred.** `ImportEthereumABI` currently rejects
>   tuple outputs because `SmartContractAbiEntryOutput` has no `Components`
>   field. Before M4.3 `CallMethod` can decode struct return values, extend the
>   output model with components (mirror the input struct) and route outputs
>   through the typed engine.
> - **Indexed reference-type placeholder** `DecodedValue.Type` carries a
>   `" (indexed)"` suffix that is NOT a valid ABI type string — M6 delivery/API
>   must not feed it back into `parseType`.
> - **Empty-name lookups:** `GetMethodByName("")` matches the first nameless
>   entry (constructor/fallback). The call/notification layer must guard against
>   name="" lookups.
> - **Preset path:** the future specific contract is added as preset DATA via
>   `ImportEthereumABI`/`NewContractFromABI` + registry cold-start — never as
>   hardcoded Go logic. The importer is ready for it.

- [x] **M4.1** Implement `GetTransactionReceipt(txHash) (*Receipt, error)`
      (the const already exists; the method does not). Add `Receipt`/`Log` Go
      types with the project's hex-decoding `UnmarshalJSON` style.
- [x] **M4.2** Implement `eth_getLogs`: `GetLogs(filter LogFilter) ([]*Log, error)`
      with fromBlock/toBlock, address list, topics.
- [x] **M4.3** Generalize `eth_call`-based reads: a `CallMethod(contract,
      method, args...) ([]DecodedValue, error)` helper using the ABI engine
      (e.g. ERC-1155 `balanceOf(addr,id)`, `balanceOfBatch`).
- [x] **M4.4** Tests against recorded JSON-RPC fixtures (IPC/HTTP).

## Milestone 5 — Event-log service (`eventlog/`, new package)  ✅ DONE

**Status:** Complete. New `eventlog/` package (Approach A — `abi/` stays pure):
a `Service` that registers as a watchdog block listener (`OnBlock`) and, per
block, collects the subscribed contracts' logs, decodes them via the abi
registry, scope-filters, and hands `ContractEvent`s to a `Sink`. Mode A
(`eth_getLogs` per block) and Mode B (per-receipt via `tools/flow.FanOut`
bounded parallel fetch) both feed one decoded stream; config selects the mode
(default getLogs). Per-subscription `Scope` (`whole_contract` | `managed_only`)
filters after decode — managed_only matches a managed address inside the
event's address-typed params. eventlog stays decoupled from `clients/ethclient`
via its own `LogSource`/`Decoder` interfaces; `main.go` wires real adapters
(`RawLogSource`, `NewDecoder(abiManager.DecodeLog)`) and registers `OnBlock` —
zero watchdog-internals change. M5's sink logs the event; M6 replaces it with
JSON-RPC `contractEvent` delivery + registration/subscription RPC.
`go test ./eventlog/ -race` clean (17 tests), `go build ./...` and `go vet` clean.
See `docs/superpowers/plans/2026-06-12-m5-eventlog-service.md`.

> **Carry-over from M2 review (perf, apply when log volume matters):**
> `(*SmartContractAbi).GetEventByTopic0` is O(entries) per log and recomputes
> `Topic0()` (keccak) for every entry on every call. For high-volume log
> decoding (Mode A/B), build a cached `map[[32]byte]*SmartContractAbiEntry` in
> `_prepare()` and look up by topic0 in O(1). Not needed for correctness; only
> if profiling shows it hot.

- [x] **M5.1** New package `eventlog/` (Approach A): consumes a chain client +
      the ABI registry, decodes logs into `DecodedEvent`, and exposes a
      listener handler. `abi/` stays a pure library with no service deps.
- [x] **M5.2** Mode A — `eth_getLogs` per block: one call per block filtered by
      registered contract addresses/topics; independent of tx count.
- [x] **M5.3** Mode B — per-receipt: fetch `eth_getTransactionReceipt` for
      relevant txs and decode their logs. Use `tools/flow.FanOut` for bounded,
      order-preserving parallel receipt fetching — see [`PIPELINE-FLOW.md`](./PIPELINE-FLOW.md).
- [x] **M5.4** Config flag to switch modes (default = `getLogs`). Document
      load/latency trade-offs.
- [x] **M5.5** Watchdog integration: register `eventlog` as an additional
      listener in the block-processing path (`watchdog/processblock.go`)
      without changing the existing From/To tx matching.
- [x] **M5.6** Match logs against managed addresses inside indexed params
      (e.g. ERC-1155 `from`/`to`/`operator`), not just `tx.To`.
- [x] **M5.7** **Per-subscription event-tracking scope** (user requirement
      2026-06-12). The subscribe call carries a `scope` parameter; BOTH modes
      are supported and may coexist on the same contract:
      - `whole_contract` — ALL events of the contract (by contract address +
        optional topic/event-name filter), regardless of participants. Decode
        every matching log via `abi.DecodeLog` and deliver.
      - `managed_only` — only events involving a managed address (matched inside
        indexed params: from/to/operator/etc.), extending M5.6.
      Scope is chosen at subscribe time (not a global flag). The eventlog
      collector must support both filtering paths off the same decoded-log
      stream.

## Milestone 6 — Delivery & API (`subscriptions/`, `endpoint/`)  ✅ DONE

**Status:** Complete. `contractEvent` is delivered to subscribers over the
existing JSON-RPC callback mechanism: `eventlog.ContractEvent` now carries the
matched subscription's `ServiceID`; `endpoint.NewContractEventSink` parses it
back to the numeric serviceId and routes the event via
`subscriptions.Manager.NotifySubscriberRaw` (a no-Signer sibling of
`NotifySubscriber`). Delivery is extensible by design — a new event type is a
new subject string + payload struct over the same `sendNotification` path, no
dispatch-switch edit. New RPC methods (in `endpoint/methods_contracts.go` +
`endpoint/methods_contract_call.go`, registered in `rpc_init.go`):
`contract.register`/`contract.subscribe`/`contract.unsubscribe` (SECURED — write
methods require serviceId + API token) and `contract.list`/
`contract.subscriptions`/`contract.call` (open, read-only); both `dot.case` and
`camelCase` aliases. The endpoint stays decoupled from the concrete
abi/eventlog/ethclient types via narrow seams (`ContractRegistry`,
`ContractAdder`, `EventSubscriber`, `ContractCaller`, `ContractEventNotifier`),
wired in main.go. Registered contracts persist for free (the abi manager Saves
on `Add`); eventlog subscriptions persist to `data/eventlog/subscriptions.json`
and reload on startup (`LoadSubscriptions`), with concurrent saves serialized by
a `saveMu`. Reorg `Removed` and `TransactionIndex` are threaded into the
delivered payload. `contractCall` is no-arg in this first cut (totalSupply/
decimals/etc.) — passing args returns a clear "not yet supported" error rather
than mis-encoding. `go test ./eventlog/ ./endpoint/ ./abi/ -race` clean,
`go build ./...` clean. See
`docs/superpowers/plans/2026-06-12-m6-delivery-and-api.md`.

> **Carry-over to M7 / future:**
> - **Pre-existing `s.rpc` lazy-init race (NOT introduced by M6):** in
>   `subscriptions/subscriptions.go`, `Subscription`'s `rpc` client is lazily
>   initialized under only an RLock, so concurrent deliveries to the SAME
>   subscriber can race. Predates M6 (the original `NotifySubscriber` already
>   held only an RLock). Fix by guarding the lazy init or building `rpc` at
>   subscribe time.
> - **Receipts mode still not wired** (see M5 carry-over below); `contractCall`
>   typed-argument encoding from JSON is a deliberate follow-up.

> **Carry-over from M5 final review (for M6):**
> - **Sink is concurrency-sensitive:** the watchdog dispatches block listeners
>   in goroutines, so `eventlog.Sink` may be called concurrently. M6's real
>   JSON-RPC delivery sink MUST be safe for concurrent use.
> - **Reorg flag:** `ContractEvent.Removed` is now carried through. M6 delivery
>   should either signal reverted events to subscribers or document that reorg
>   handling is out of scope.
> - **Checksummed-address invariant (managed_only):** matching works because
>   both the managed pool and `eventMatchesScope` use the SAME codec
>   (`EncodeBytesToAddress`, EIP-55 checksummed). If an M6 RPC registers
>   user-supplied addresses, normalize them through the same codec or
>   managed_only will silently under-match. Add an end-to-end test with a real
>   checksummed managed address.
> - **Receipts mode wiring:** `eventlog` Mode B is implemented + tested but the
>   `blockTxHashes` seam is NOT wired in main.go (default is getLogs). To enable
>   `ModeReceipts`, wire `blockTxHashes` from the chain client and add a config
>   knob. Also `TransactionIndex` is now on `ContractEvent` for ordering.

> **Extensibility requirement (user, 2026-06-12):** the notification system and
> the JSON-RPC endpoint must be EXTENSIBLE — adding a new event/notification
> type or a new contract preset must be data/registration-driven, not a core
> rewrite. Design `contractEvent` + the registration/subscription RPC surface
> so new event types and presets plug in without touching the dispatch core.

- [x] **M6.1** New notification type `contractEvent` delivered to subscribers via
      the existing JSON-RPC 2.0 callback mechanism (alongside `blockEvent` /
      `transactionEvent`). Dispatch is extensible: a new event type plugs in as a
      new subject+payload over `sendNotification`, no switch-arm edit.
- [x] **M6.2** RPC methods: register a contract + ABI, list registered
      contracts, subscribe to contract events with a `scope` param
      (`whole_contract` | `managed_only`) per M5.7, list AND unsubscribe.
      Registered via `RegisterProcessor`/`RegisterSecuredProcessor` (write methods
      secured) following existing `methods_*.go` patterns.
- [x] **M6.3** Read-only RPC: `contractCall` to invoke a view method by
      name and return decoded outputs (no-arg in this first cut; typed-arg
      encoding from JSON is a documented follow-up).
- [x] **M6.4** Persist registered contracts (abi manager Saves on `Add`) and
      subscriptions (`data/eventlog/subscriptions.json`, reloaded on startup).

## Milestone 7 — Docs & tests  ✅ DONE

> **Pre-doc fix (2026-06-13):** before documenting, the `contractEvent` /
> `contractCall` decoded-value wire format was made client-safe via
> `abi.DecodedValue.MarshalJSON`: byte types (`address`/`bytesN`/`bytes`) →
> `0x`-hex strings (not base64); big integers (`uint*`/`int*`) → decimal STRINGS
> (not bare JSON numbers, which JS `JSON.parse` corrupts above 2^53). The in-memory
> Value type is unchanged. Also: `contract.subscribe` now rejects unregistered
> contracts (see M6 carry-over RESOLVED). All docs describe the fixed format.

- [x] **M7.1** Update `DOC.md` (package table, interfaces, config, new RPC
      methods + `contractEvent` notification, contract event flow, data dir).
- [x] **M7.2** Update `API.md` with new methods and event payloads — full
      Smart Contract Layer section (6 methods + `contractEvent` + Decoded value
      format + integration quick-start). README updated (layer now implemented).
- [x] **M7.3** End-to-end test (`endpoint/e2e_contract_event_test.go`): registers
      a generic multitoken-class ABI via the REAL abi manager, replays a block's
      log through the REAL eventlog + decoder + EIP-55 codec + sink, and asserts
      the delivered `contractEvent` wire JSON for BOTH scopes — including the M5
      checksummed managed-address invariant and the JS-safe value encoding.
- [x] **M7.4** `-race` clean across all packages EXCEPT three with **pre-existing,
      unrelated** failures (proven by `git stash` on the clean base): build env
      issue in `crypto/secp256k1` (Go stdlib `ecdsa.Sign` signature), missing
      `testdata/*.in.txt` fixtures in `common/rlp/rlpgen`, and `uniclient`
      `TestClient` (needs a live node). None are in the M1–M7 diff.

---

## Milestone 8 — `contractTransaction` delivery (`eventlog/`, `endpoint/`) — ✅ DONE (2026-09-06)

**Goal:** notify a contract subscriber about the actual transactions sent to
the subscribed contract, not only the events it emitted — a transaction that
reverted, or that calls a method with no logs, is invisible to `contractEvent`
today. Agreed with the user (2026-09-06):

- New notification type `contractTransaction`, delivered through the **same**
  subscription as `contractEvent` (`{serviceId, contractAddress, scope}`) —
  subscribing to a contract yields both.
- **Payload is decoded, not raw calldata**: look up the method by its 4-byte
  selector (`tx.Input[:4]`) against the contract's registered ABI and decode
  the inputs (`abi.SmartContractAbiEntry.DecodeInputsTyped`) the same way
  `contractCall`/`contractEvent` already do — reuses existing decode machinery,
  nothing new to build there. Unlike `contractEvent` (which skips a log
  matching no known event), a transaction is **never silently skipped for
  being undecodable**: when the selector matches no known method, the
  `method` field carries the raw 4-byte selector as hex (e.g. `"0xa9059cbb"`)
  instead of a decoded name, with no decoded inputs. The raw calldata is
  **always** included too (`data`, 0x-hex), decoded or not — a subscriber can
  decode an unknown selector itself, or double-check a decoded one.
- **Scope for v1: `whole_contract` only.** `managed_only` for a raw transaction
  (`tx.From`/`tx.To` against the managed-address pool, analogous to what
  `watchdog` already does for plain transfers) is deferred — see Future Phases.

**Delivered as** a data path independent of `ModeGetLogs`/`ModeReceipts` (it
turned out not to need the latter): `eventlog.Service` gained
`TransactionSource`/`TransactionDecoder`/`TransactionSink` seams, all
optional — `OnBlock` only fetches block transactions when a `TransactionSink`
is wired, so a deployment that only wants `contractEvent` pays nothing extra.
`abi.SmartContractsManager.DecodeCall` (new, `abi/decodecall.go`) identifies
the method by its 4-byte selector and decodes via the existing
`DecodeInputsTyped`, returning the raw hex selector (never an error) when it
matches no known method. `endpoint.NewContractTransactionSink` (new) builds
the `contractTransaction` JSON-RPC notification, `Value` as a decimal string
like `contractEvent`'s `Inputs`. Wired in `main.go` via
`chainClient.GetBlockByNumber(num, true)` (one extra call per block, only
when some contract is subscribed) plus a receipt fetch per matched
transaction (the existing `GetTransactionReceipt` seam, extended with
`Status`/`GasUsed`). 20 new tests across `abi`/`eventlog`/`endpoint`, green
under `-race`; not additionally verified against a live node (unlike the
EIP-1559 signer above), since every collaborator here is internal Go already
covered by fakes matching the real interfaces.

**Docs and end-to-end coverage added 2026-09-07:** `API.md` documents it in
full (new `## contractTransaction` section; updated `contractSubscribe`/
`contractUnsubscribe`/scope docs — one subscription already yields both
notification types, no separate subscribe endpoint).
`endpoint/e2e_contract_transaction_test.go` (new, mirrors
`e2e_contract_event_test.go`) exercises the real RPC-facing stack end to end:
`SubscribeAndSaveStrings` → `OnBlock` → the actual delivered wire JSON, for a
decoded call, an unknown selector, and confirming `managed_only` does not
receive it.

### Milestone 8.1 — `contractTransaction` operation filter — ✅ DONE (2026-09-07)

Agreed with the user: of the two filtering gaps M8 deferred, **per-operation
filtering** (not `managed_only` for transactions) comes first — it solves the
sharper problem (noise from a busy contract) and needs no populated
managed-address pool. Decisions from the discussion:

- **Filter by 4-byte selector, never by decoded method name** — two
  differently-typed overloads of one name (`transfer(address,uint256)` vs
  `transfer(address,uint256,bytes)`) are two different selectors; a name-only
  filter would be ambiguous between them. New `abi.Selector(canonicalSignature
  string) [4]byte` computes one without needing an ABI entry — a pure
  keccak256 hash, the same one `SmartContractAbiEntry.updateSignature` uses
  internally.
- **A hybrid, both forms accepted at `contractSubscribe`:** `selectors`
  (raw `0x`-hex 4-byte values — works even for a method not in this node's
  registered ABI) and `methods` (canonical signatures, e.g.
  `"transfer(address,uint256)"`, resolved via `abi.Selector` — no ABI lookup
  needed either, just friendlier). Both may be given together and merge into
  one filter set (`endpoint.resolveSelectors`).
- **A list**, not one selector per subscription — subscribing separately per
  operation was rejected as impractical.
- No filter named (the default) means unfiltered, unchanged from before this
  existed. A plain value transfer (no calldata) never matches a *filtered*
  subscription — there is no selector to filter on — but still matches an
  unfiltered one.

`eventlog.Subscription` gained `Selectors [][4]byte`, checked in
`decodeAndDeliverTransaction` against each transaction's raw selector before
delivery (matching happens whether or not the call decodes — filtering is on
the raw bytes, decoding is a separate, later step). Persisted as `0x`-hex in
`persistedSub.Selectors` and round-trips through a restart.
`ListSubscriptions`'s per-entry map gained a `selectors` field (`[]string`,
always present, empty when unfiltered) — its return type changed from
`map[string]string` to `map[string]any` to carry it.

`API.md`: new `### Filtering contractTransaction by operation` section under
`contractSubscribe`, updated parameter/error tables and the
`contractSubscriptions` result shape. 20 new tests across
`abi`/`eventlog`/`endpoint` (including a full RPC-facing end-to-end case:
two different operations to one contract in one block, only the subscribed
one delivered), green under `-race`.

### Milestone 8.2 — `uniclient` gains Smart Contract Layer coverage — ✅ DONE (2026-09-07)

`uniclient` (`docs`: "unified JSON-RPC client for interacting with EthBackNode
services", embedded as a library by other services — not imported anywhere in
this repo) had **zero** coverage of the Smart Contract Layer despite it being
the project's main feature area: no `contractRegister`/`contractSubscribe`/
`contractUnsubscribe`/`contractList`/`contractSubscriptions`/`contractCall`.
Found while auditing `uniclient`'s state against `endpoint/rpc_init.go`'s
full registered-method list; also missing (deferred, see Future Phases):
`ping`, `infoGetTokenList`, `addressSubscribe`/`addressRecover`/
`addressGenerate`, and `serviceRegister`/`serviceConfig`
(`methods_service.go` is a 1-line stub — consistent with the server's own
`serviceRegister` being `panic("Not implemented")`).

New `uniclient/methods_contracts.go`: all 6 methods, matching API.md exactly,
including this session's `selectors`/`methods` operation filter on
`ContractSubscribe` and the `Selectors` field on `ContractSubscriptionInfo`.
`ContractCall` accepts `args` for forward-compatibility even though the
server currently rejects a non-empty list. New `uniclient/methods_contracts_test.go`
introduces a `fakeTransport` (the package's first mocked-transport tests —
every existing method is only exercised by `client_test.go`'s `TestClient`,
one integration test against a real server at `localhost:21280` with no
skip guard, which is `docs/PROJECT_STATUS.md`'s known "needs a live node"
failure and unaffected by this work). 9 new unit tests, green under `-race`.

---

## Future Phases (recorded, out of current scope)

- **Write support:** sign + broadcast arbitrary contract methods from a managed
  address (generalize `send_methods.go`), gas/nonce handling for contract calls.
  *(Nonce allocation and EIP-1559 signing landed 2026-09-06 —
  `clients/txmanager`, `crypto.EthDynamicFeeTxSigner` — but only for plain
  coin transfers; generalizing to arbitrary contract-write methods is still
  open.)*
- **EIP-712** typed-data signing for CLOB orders (`OrderFilled`/`OrdersMatched`).
- **Contract subscription filters, remaining:** `managed_only` scope for
  `contractTransaction` (M8 — matched on `from`/`to` against the managed pool,
  analogous to `watchdog`'s plain-transfer tracking); per-event-name and
  per-argument-value filtering for `contractEvent` (`contractTransaction` got
  its operation filter in M8.1 — this is the equivalent still missing on the
  event side).
- **CTF write methods:** `splitPosition`, `mergePositions`, `redeemPositions`.
- **Polymarket domain model:** markets, conditions, outcome tokens, position P&L.
- **`uniclient` remaining method coverage (M8.2 follow-up):** `ping`,
  `infoGetTokenList`, `addressSubscribe`/`addressRecover`/`addressGenerate`,
  and `serviceRegister`/`serviceConfig` (the last two only once the server
  side of `serviceRegister` is implemented — it is `panic("Not implemented")`
  today). Also: give `client_test.go`'s `TestClient` a mocked-transport
  counterpart (or a skip guard) so `go test ./uniclient/...` is not always
  red without a live server — deferred behind Smart Contract Layer coverage
  by the user's own priority call.

---

## Suggested order

M1 → M2 → M3 (pure `abi/`, no external deps, highest-leverage foundation) →
M4 (client receipts/logs) → M5 (eventlog service + watchdog) →
M6 (delivery + API) → M7 (docs + tests).