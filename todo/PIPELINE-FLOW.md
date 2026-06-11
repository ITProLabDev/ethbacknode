# Design Note: `tools/flow` — typed pipeline layer over go_pysyun_pipeline

**Status:** Design / planning. **Implementation NOT started.**
**Goal:** speed up and harden I/O-bound paths with bounded, order-preserving
fan-out, while keeping type safety, `context` cancellation, and explicit error
propagation that the underlying library does not provide.

---

## Decision

- Use **`github.com/pysyun/go_pysyun_pipeline`** (pkg `pipeline`, Go 1.21,
  **LGPL-2.1**) as the **outer execution layer** — its `Chainable`,
  `ChainableGroup` (bounded, order-preserving parallel fan-out), and `Pipe`.
- **Project convention:** stages pass **exactly one typed container — the
  `Envelope[T]`** — through the pipeline. The `any` boxing required by the
  library happens **only at its boundary**; inside stages everything is typed.
- The `Envelope` carries what the library lacks: a **`context.Context`**
  (cancellation/deadlines) and an **accumulated error** (short-circuit +
  the name of the failing stage).

### Why a wrapper instead of using the library directly
- Library API is `Process(any) any` — no generics, no `context`, no error
  channel. Raw use means pervasive type assertions and lost errors.
- The wrapper restores type safety (Go 1.24 generics), cancellation, and
  explicit errors, while still delegating concurrency to the library.

---

## Package `tools/flow`

```go
package flow

// Envelope is the ONLY value that travels through a pipeline.
// Convention: stages accept and return *Envelope[T], never a bare payload.
type Envelope[T any] struct {
    Ctx         context.Context // cancellation / deadlines (library has none)
    Payload     T               // typed payload
    Err         error           // first error in the chain (library has no error channel)
    FailedStage string          // stage where Err originated
}

// Step is what a stage author writes: a pure typed function.
type Step[T any] func(ctx context.Context, payload T) (T, error)

// NamedStep pairs a Step with a label for diagnostics / FailedStage.
type NamedStep[T any] struct {
    Name string
    Step Step[T]
}

// Stage adapts a typed Step into a pipeline.Processor operating on *Envelope[T].
// It handles: short-circuit when Err != nil, Ctx cancellation check,
// timing, and wrapping the returned error with the stage name.
func Stage[T any](name string, step Step[T]) pipeline.Processor

// Sequence runs a synchronous linear chain (built on pipeline.Pipe).
// Boxes the Envelope into `any` once at the boundary, unboxes at the end.
func Sequence[T any](ctx context.Context, in T, steps ...NamedStep[T]) (T, error)

// FanOut runs a bounded, order-preserving parallel fan-out over items
// (built on pipeline.ChainableGroup). Returns the first error encountered.
// concurrency <= 0 → library default (one goroutine per item).
func FanOut[T any](ctx context.Context, items []T, concurrency int, step Step[T]) ([]T, error)
```

### Boundary rule (the convention in one line)
> Only `*Envelope[T]` is ever boxed into `any` and passed to the library.
> Never pass a bare payload across a stage. Box once, unbox once.

### Stage author example
```go
fetchReceipt := func(ctx context.Context, tx *types.TransferInfo) (*types.TransferInfo, error) {
    r, err := client.GetTransactionReceipt(ctx, tx.TxID) // ctx from the Envelope
    if err != nil { return tx, err }                     // error → auto-captured into Envelope.Err
    tx.Logs = r.Logs
    return tx, nil
}
receipts, err := flow.FanOut(ctx, block.Transactions, cfg.ReceiptConcurrency, fetchReceipt)
```

---

## Prerequisite built: IPC connection pool

The recommended node transport is IPC, but `urpc.ipcClient` serializes every
call through one mutex-guarded connection — so fan-out over node RPC would not
speed anything up on IPC. To unlock real parallelism a bounded **IPC connection
pool** was added (TDD, race-clean):

- `urpc.newIPCPool` / `urpc.WithRpcIPCSocketPool(path, size, timeout)` — up to
  `size` live connections; LIFO idle reuse; per-connection exclusive ownership
  (no socket cross-talk); counting semaphore bounds concurrency.
- `ethclient.WithIPCClientPool(path, size, timeout)` exposes it to the client.
- `main.go` reads `ipcPoolSize` (config `paramsInt`, default 1 = old behavior).

`flow.FanOut` over node RPC only helps when paired with this pool (or HTTP
transport, which is already concurrency-safe).

## Where to apply (ranked by payoff)

1. **`eventlog` Mode B — receipt per tx** (see [`TASKS.md`](./TASKS.md) M5.3).
   `eth_getTransactionReceipt` per relevant tx is network I/O; today
   `watchdog/processblock.go` processes txs strictly sequentially. Use
   `FanOut[*types.TransferInfo]` with bounded concurrency. **Highest payoff.**
2. **Block catch-up** (`watchdog/runloop.go`): fan out `BlockByNum` over a
   missed range (order preserved); keep `state.UpdateState` sequential.
3. **Unbounded goroutine fan-out** in `watchdog/event.go`
   (`for _, h := range handlers { go h(...) }`) → replace with bounded `FanOut`.
   This is a **reliability** fix (goroutine-explosion guard), not just speed.
4. **Subscriber notifications** (`subscriptions/NotifySubscriber`): deliver one
   event to many subscribers over HTTP callbacks as a bounded fan-out.
5. **Multi-balance** (`TokensBalanceOf`, `endpoint/methods_balance.go`):
   N independent `eth_call`s → `FanOut`, order preserved.

---

## Constraints & caveats

- **Thread-safety:** `FanOut` stages run concurrently. Stages must be
  thread-safe; guard shared state. Stages writing into managed-address /
  registry maps must respect existing mutexes.
- **Library limits:** not streaming (`ChainableGroup.Process` takes the whole
  slice); `ChainableGroup` needs at least one `Pipe` before `Process`.
- **License:** library is LGPL-2.1; project is GPLv3 — compatible, but the
  dependency must be acknowledged.
- **Concurrency config:** expose per-use-site concurrency in config
  (e.g. `ReceiptConcurrency`, `NotifyConcurrency`) rather than hardcoding.

---

## Tasks (when implementation starts)

- [x] **F1** Add dependency `github.com/pysyun/go_pysyun_pipeline`; note LICENSE.
- [x] **F2** Implement `tools/flow`: `Envelope`, `Step`, `NamedStep`, `Stage`,
      `Sequence`, `FanOut`. Unit tests incl. cancellation, error short-circuit,
      order preservation, `-race`. *(18 tests, TDD, race-clean, vet-clean.)*
- [ ] **F3** Apply to `eventlog` Mode B receipt fetching (with M5.3).
- [ ] **F4** Apply to block catch-up in `watchdog/runloop.go`.
- [ ] **F5** Replace unbounded fan-out in `watchdog/event.go`.
- [ ] **F6** Apply to subscriber notifications (`subscriptions/`).
- [ ] **F7** Apply to multi-balance queries.
- [ ] **F8** Add concurrency settings to config + document in `DOC.md`.
