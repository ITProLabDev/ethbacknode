# M4 — Client: Receipts, Logs & Generic Contract Calls Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give the Ethereum client the read primitives the universal contract layer needs — transaction receipts (with logs), `eth_getLogs` filtering, and a generic `CallMethod` that invokes any view method by name and returns decoded outputs — all on the project's own JSON-RPC + ABI stack.

**Architecture:** Add `Receipt`/`Log` Go types that decode geth's 0x-hex JSON via the project's proxy-map `UnmarshalJSON` idiom. Add `GetTransactionReceipt` and `GetLogs(filter)` RPC methods on `*ethclient.Client`. For generic reads, add an exported `DecodeOutputs` to the ABI entry (mirroring `DecodeInputsTyped` but for return data, no selector) and a client `CallMethod(contract, method, args...)` that encodes call-data via the ABI engine, runs `eth_call`, and decodes the result. No external libraries.

**Tech Stack:** Go 1.24, `encoding/json`, `math/big`, the project's `common/hexnum`, the M1–M3 ABI engine (`abi/`), the `urpc` client, standard `testing`.

---

## Background & Constraints

**Hard rule:** self-built stack only — NO go-ethereum / external ABI or RPC library.

**The proxy-map hex-decode idiom (MUST follow):** geth returns all numbers as 0x-prefixed hex strings, so every RPC struct in `clients/ethclient/` implements `UnmarshalJSON` by first decoding into `map[string]json.RawMessage`, then decoding each field by type and converting hex via `common/hexnum`. See `Block.UnmarshalJSON` / `Transaction.UnmarshalJSON` in `types_block.go` / `types_transaction.go`.

**`common/hexnum` helpers available:** `ParseHexInt64(string)(int64,error)`, `ParseHexUint64`, `ParseBigInt(string)(*big.Int,error)`, `ParseHexBytes(string)([]byte,error)`, `Int64ToHex`, `BigIntToHex`, `BytesToHex`.

**What already exists (verified):**
- `clients/ethclient/methods.go`: const `ethGetTransactionReceipt = "eth_getTransactionReceipt"` is declared but **no `GetTransactionReceipt` method**. `eth_getLogs` does not exist. `Call(contractAddress, data string) (string, error)` runs `eth_call` and returns the 0x-hex result string. `CallByBlockNumber` likewise. `GetTransactionByHash` shows the null-result guard pattern (`result.Result == nil || string == "null"`).
- `urpc`: `NewRequest(method, params...)`, `req.AddParams(...)` (accepts objects — marshaled to JSON), `c.rpcClient.Call(req) (*Response, error)`, `response.ParseResult(&target)`. Object params are supported (see `eth_call`'s struct param in `methods.go`).
- ABI engine: `(*SmartContractAbiEntry).EncodeInputsTyped(values ...any) ([]byte, error)` (selector + encoded args), `.DecodeInputsTyped`, `.canonicalSignature`, `.Topic0`, `.typedInputs`; `decodeParams(types []abiType, block []byte) ([]DecodedValue, error)` (UNEXPORTED, in `abi/codec.go`); `parseType`; `DecodedValue{Name,Type string; Value any}`; `(*SmartContractsManager).GetSmartContractByAddress`, `(*SmartContractAbi).GetMethodByName`.
- `Client.abi *abi.SmartContractsManager`, `Client.Call`, `Client.rpcClient *urpc.Client`.
- `c.abi.DecodeLog(contractAddress string, topics [][32]byte, data []byte) (*abi.DecodedEvent, error)` (M2) — for later M5 use; not directly used here.

**Key design decisions for M4:**
- `Log.Topics` is decoded as `[]string` (0x-hex) on the wire, plus a helper `Topics32() ([][32]byte, error)` that converts to the `[][32]byte` form `abi.DecodeLog` expects. This keeps the wire type faithful and bridges to the ABI layer without M4 depending on decode internals.
- Generic output decoding needs the method's `Outputs`. M3 deferred tuple OUTPUTS (the output struct has no `Components`), so `DecodeOutputs` supports the elementary/array/bytes/string output types (everything the importer accepts) and is the single new ABI method this milestone adds. Tuple-output support stays deferred.
- `CallMethod` returns `[]abi.DecodedValue` (the decoded outputs), mirroring the neutral model used everywhere else.

---

## File Structure

- **Create `clients/ethclient/types_receipt.go`** — `Log` and `Receipt` structs + their proxy-map `UnmarshalJSON`; `Log.Topics32()` helper.
- **Create `clients/ethclient/types_receipt_test.go`** — unmarshal tests against captured geth JSON, `Topics32` conversion.
- **Modify `clients/ethclient/methods.go`** — add `ethGetLogs` const, `GetTransactionReceipt`, `GetLogs`, and the `LogFilter` type.
- **Create `clients/ethclient/methods_logs_test.go`** — `LogFilter` → request param marshaling test (no live node).
- **Create `abi/output.go`** — `(*SmartContractAbiEntry).DecodeOutputs(data []byte) ([]DecodedValue, error)`.
- **Create `abi/output_test.go`** — output decode tests.
- **Create `clients/ethclient/call_method.go`** — `CallMethod(contractAddress, methodName string, args ...any) ([]abi.DecodedValue, error)`.

No changes to the codec internals, the legacy decoder, or `erc20.go`.

---

### Task 1: `Log` type + proxy-map decode + `Topics32`

**Files:**
- Create: `clients/ethclient/types_receipt.go`
- Test: `clients/ethclient/types_receipt_test.go`

- [ ] **Step 1: Write the failing test** — create `clients/ethclient/types_receipt_test.go`:

```go
package ethclient

import (
	"encoding/json"
	"testing"
)

const sampleLogJSON = `{
  "address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
  "topics": [
    "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
    "0x0000000000000000000000001111111111111111111111111111111111111111",
    "0x0000000000000000000000002222222222222222222222222222222222222222"
  ],
  "data": "0x00000000000000000000000000000000000000000000000000000000000003e8",
  "blockNumber": "0x10d4f",
  "transactionHash": "0xabc0000000000000000000000000000000000000000000000000000000000001",
  "transactionIndex": "0x2",
  "logIndex": "0x5",
  "removed": false
}`

func TestLog_UnmarshalJSON(t *testing.T) {
	var lg Log
	if err := json.Unmarshal([]byte(sampleLogJSON), &lg); err != nil {
		t.Fatal(err)
	}
	if lg.Address != "0xdac17f958d2ee523a2206206994597c13d831ec7" {
		t.Fatalf("address=%q", lg.Address)
	}
	if len(lg.Topics) != 3 {
		t.Fatalf("topics=%d want 3", len(lg.Topics))
	}
	if lg.BlockNumber != 0x10d4f {
		t.Fatalf("blockNumber=%d want %d", lg.BlockNumber, 0x10d4f)
	}
	if lg.TransactionIndex != 2 || lg.LogIndex != 5 {
		t.Fatalf("txIndex=%d logIndex=%d", lg.TransactionIndex, lg.LogIndex)
	}
	if lg.Removed {
		t.Fatal("removed should be false")
	}
	if len(lg.Data) != 32 {
		t.Fatalf("data len=%d want 32", len(lg.Data))
	}
}

func TestLog_Topics32(t *testing.T) {
	var lg Log
	if err := json.Unmarshal([]byte(sampleLogJSON), &lg); err != nil {
		t.Fatal(err)
	}
	t32, err := lg.Topics32()
	if err != nil {
		t.Fatal(err)
	}
	if len(t32) != 3 {
		t.Fatalf("t32 len=%d", len(t32))
	}
	// topic0 first byte 0xdd, last byte 0xef
	if t32[0][0] != 0xdd || t32[0][31] != 0xef {
		t.Fatalf("topic0=%x", t32[0])
	}
}

func TestLog_Topics32_RejectsBadLength(t *testing.T) {
	lg := Log{Topics: []string{"0x1234"}} // 2 bytes, not 32
	if _, err := lg.Topics32(); err == nil {
		t.Fatal("a non-32-byte topic must error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./clients/ethclient/ -run 'TestLog_' -v`
Expected: FAIL — `undefined: Log`.

- [ ] **Step 3: Write minimal implementation** — create `clients/ethclient/types_receipt.go`:

```go
package ethclient

import (
	"encoding/json"
	"fmt"

	"github.com/ITProLabDev/ethbacknode/common/hexnum"
)

// Log is a single event log entry as returned by eth_getLogs /
// eth_getTransactionReceipt. Numeric fields arrive as 0x-hex strings and are
// decoded via the proxy-map idiom used across this package.
type Log struct {
	Address          string   // 20-byte contract address (0x...)
	Topics           []string // 0x-hex 32-byte topics; Topics[0] is the event signature
	Data             []byte   // non-indexed event data (ABI-encoded)
	BlockNumber      int64    // block containing the log
	TransactionHash  string   // 0x-hex tx hash
	TransactionIndex int64    // tx position within the block
	LogIndex         int64    // log position within the block
	Removed          bool     // true if the log was reverted by a chain reorg
}

// Topics32 converts the 0x-hex topics into the [32]byte form the ABI decoder
// (abi.DecodeLog) expects. It errors if any topic is not exactly 32 bytes.
func (l *Log) Topics32() ([][32]byte, error) {
	out := make([][32]byte, len(l.Topics))
	for i, t := range l.Topics {
		b, err := hexnum.ParseHexBytes(t)
		if err != nil {
			return nil, err
		}
		if len(b) != 32 {
			return nil, fmt.Errorf("topic %d is %d bytes, want 32", i, len(b))
		}
		copy(out[i][:], b)
	}
	return out, nil
}

// UnmarshalJSON decodes a log from geth's 0x-hex JSON representation.
func (l *Log) UnmarshalJSON(data []byte) error {
	proxy := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &proxy); err != nil {
		return err
	}
	if v, ok := proxy["address"]; ok {
		if err := json.Unmarshal(v, &l.Address); err != nil {
			return err
		}
	}
	if v, ok := proxy["topics"]; ok {
		if err := json.Unmarshal(v, &l.Topics); err != nil {
			return err
		}
	}
	if v, ok := proxy["data"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return err
		}
		if s != "" && s != "0x" {
			b, err := hexnum.ParseHexBytes(s)
			if err != nil {
				return err
			}
			l.Data = b
		}
	}
	if v, ok := proxy["blockNumber"]; ok {
		if err := unmarshalHexInt64(v, &l.BlockNumber); err != nil {
			return err
		}
	}
	if v, ok := proxy["transactionHash"]; ok {
		if err := json.Unmarshal(v, &l.TransactionHash); err != nil {
			return err
		}
	}
	if v, ok := proxy["transactionIndex"]; ok {
		if err := unmarshalHexInt64(v, &l.TransactionIndex); err != nil {
			return err
		}
	}
	if v, ok := proxy["logIndex"]; ok {
		if err := unmarshalHexInt64(v, &l.LogIndex); err != nil {
			return err
		}
	}
	if v, ok := proxy["removed"]; ok {
		if err := json.Unmarshal(v, &l.Removed); err != nil {
			return err
		}
	}
	return nil
}

// unmarshalHexInt64 decodes a JSON 0x-hex string into an int64. Empty / "0x"
// values decode to 0.
func unmarshalHexInt64(raw json.RawMessage, dst *int64) error {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return err
	}
	if s == "" || s == "0x" {
		*dst = 0
		return nil
	}
	n, err := hexnum.ParseHexInt64(s)
	if err != nil {
		return err
	}
	*dst = n
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./clients/ethclient/ -run 'TestLog_' -v`
Expected: PASS (all three).

- [ ] **Step 5: Commit**

```bash
git add clients/ethclient/types_receipt.go clients/ethclient/types_receipt_test.go
git commit -m "feat(ethclient): Log type with hex-JSON decode and Topics32 bridge"
```

---

### Task 2: `Receipt` type + proxy-map decode (with nested logs)

**Files:**
- Modify: `clients/ethclient/types_receipt.go`
- Test: `clients/ethclient/types_receipt_test.go` (append)

- [ ] **Step 1: Write the failing test** — append to `clients/ethclient/types_receipt_test.go`:

```go
const sampleReceiptJSON = `{
  "transactionHash": "0xabc0000000000000000000000000000000000000000000000000000000000001",
  "transactionIndex": "0x2",
  "blockHash": "0xbbb0000000000000000000000000000000000000000000000000000000000002",
  "blockNumber": "0x10d4f",
  "from": "0x1111111111111111111111111111111111111111",
  "to": "0xdac17f958d2ee523a2206206994597c13d831ec7",
  "cumulativeGasUsed": "0x5208",
  "gasUsed": "0x5208",
  "contractAddress": null,
  "status": "0x1",
  "logs": [
    {
      "address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
      "topics": ["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"],
      "data": "0x00000000000000000000000000000000000000000000000000000000000003e8",
      "blockNumber": "0x10d4f",
      "transactionHash": "0xabc0000000000000000000000000000000000000000000000000000000000001",
      "transactionIndex": "0x2",
      "logIndex": "0x0",
      "removed": false
    }
  ]
}`

func TestReceipt_UnmarshalJSON(t *testing.T) {
	var r Receipt
	if err := json.Unmarshal([]byte(sampleReceiptJSON), &r); err != nil {
		t.Fatal(err)
	}
	if r.Status != 1 {
		t.Fatalf("status=%d want 1", r.Status)
	}
	if r.BlockNumber != 0x10d4f {
		t.Fatalf("blockNumber=%d", r.BlockNumber)
	}
	if r.GasUsed != 0x5208 {
		t.Fatalf("gasUsed=%d", r.GasUsed)
	}
	if r.From != "0x1111111111111111111111111111111111111111" {
		t.Fatalf("from=%q", r.From)
	}
	if len(r.Logs) != 1 {
		t.Fatalf("logs=%d want 1", len(r.Logs))
	}
	if len(r.Logs[0].Topics) != 1 {
		t.Fatalf("log topics=%d", len(r.Logs[0].Topics))
	}
}

func TestReceipt_Success(t *testing.T) {
	var r Receipt
	if err := json.Unmarshal([]byte(sampleReceiptJSON), &r); err != nil {
		t.Fatal(err)
	}
	if !r.Success() {
		t.Fatal("status 0x1 should be Success")
	}
	r.Status = 0
	if r.Success() {
		t.Fatal("status 0x0 should not be Success")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./clients/ethclient/ -run 'TestReceipt_' -v`
Expected: FAIL — `undefined: Receipt`.

- [ ] **Step 3: Write minimal implementation** — append to `clients/ethclient/types_receipt.go`:

```go
// Receipt is a transaction receipt as returned by eth_getTransactionReceipt.
type Receipt struct {
	TransactionHash   string // 0x-hex tx hash
	TransactionIndex  int64  // tx position within the block
	BlockHash         string // 0x-hex block hash
	BlockNumber       int64  // block number
	From              string // sender (0x...)
	To                string // recipient (0x...); empty for contract creation
	ContractAddress   string // created contract address, empty if none
	CumulativeGasUsed int64  // cumulative gas used in the block up to this tx
	GasUsed           int64  // gas used by this tx
	Status            int64  // 1 = success, 0 = failure
	Logs              []*Log // event logs emitted by this tx
}

// Success reports whether the transaction succeeded (status == 1).
func (r *Receipt) Success() bool { return r.Status == 1 }

// UnmarshalJSON decodes a receipt from geth's 0x-hex JSON representation.
func (r *Receipt) UnmarshalJSON(data []byte) error {
	proxy := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &proxy); err != nil {
		return err
	}
	strField := func(key string, dst *string) error {
		if v, ok := proxy[key]; ok && string(v) != "null" {
			return json.Unmarshal(v, dst)
		}
		return nil
	}
	intField := func(key string, dst *int64) error {
		if v, ok := proxy[key]; ok && string(v) != "null" {
			return unmarshalHexInt64(v, dst)
		}
		return nil
	}
	if err := strField("transactionHash", &r.TransactionHash); err != nil {
		return err
	}
	if err := intField("transactionIndex", &r.TransactionIndex); err != nil {
		return err
	}
	if err := strField("blockHash", &r.BlockHash); err != nil {
		return err
	}
	if err := intField("blockNumber", &r.BlockNumber); err != nil {
		return err
	}
	if err := strField("from", &r.From); err != nil {
		return err
	}
	if err := strField("to", &r.To); err != nil {
		return err
	}
	if err := strField("contractAddress", &r.ContractAddress); err != nil {
		return err
	}
	if err := intField("cumulativeGasUsed", &r.CumulativeGasUsed); err != nil {
		return err
	}
	if err := intField("gasUsed", &r.GasUsed); err != nil {
		return err
	}
	if err := intField("status", &r.Status); err != nil {
		return err
	}
	if v, ok := proxy["logs"]; ok && string(v) != "null" {
		if err := json.Unmarshal(v, &r.Logs); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./clients/ethclient/ -run 'TestReceipt_' -v`
Expected: PASS (both).

- [ ] **Step 5: Commit**

```bash
git add clients/ethclient/types_receipt.go clients/ethclient/types_receipt_test.go
git commit -m "feat(ethclient): Receipt type with nested logs and hex-JSON decode"
```

---

### Task 3: `GetTransactionReceipt` RPC method

**Files:**
- Modify: `clients/ethclient/methods.go`
- Test: `clients/ethclient/methods_logs_test.go` (create — also used by Task 4)

This task adds the method. Because it needs a live node, the unit test asserts the request is well-formed via a fake transport (no network). The fake transport is created here and reused in Task 4.

- [ ] **Step 1: Write the failing test** — create `clients/ethclient/methods_logs_test.go`:

```go
package ethclient

import (
	"encoding/json"
	"testing"

	"github.com/ITProLabDev/ethbacknode/clients/urpc"
)

// fakeTransport is a urpc transport that records the last request and replays a
// canned JSON result, so RPC methods can be unit-tested without a node.
type fakeTransport struct {
	lastMethod string
	lastParams []byte
	result     json.RawMessage
}

func (f *fakeTransport) Call(request interface{}, response interface{}) error {
	// request marshals to the JSON-RPC envelope; capture method + params.
	raw, _ := json.Marshal(request)
	var env struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	_ = json.Unmarshal(raw, &env)
	f.lastMethod = env.Method
	f.lastParams = env.Params
	// Write the canned result into the response (*urpc.Response).
	resp, _ := response.(*urpc.Response)
	if resp != nil {
		resp.Result = f.result
	}
	return nil
}

func newFakeClient(result string) (*Client, *fakeTransport) {
	ft := &fakeTransport{result: json.RawMessage(result)}
	c := &Client{rpcClient: urpc.NewClientWithTransport(ft)}
	return c, ft
}

func TestGetTransactionReceipt_RequestAndDecode(t *testing.T) {
	c, ft := newFakeClient(sampleReceiptJSON)
	r, err := c.GetTransactionReceipt("0xabc0000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	if ft.lastMethod != "eth_getTransactionReceipt" {
		t.Fatalf("method=%q", ft.lastMethod)
	}
	if r.Status != 1 || len(r.Logs) != 1 {
		t.Fatalf("receipt not decoded: %+v", r)
	}
}

func TestGetTransactionReceipt_NotFound(t *testing.T) {
	c, _ := newFakeClient("null")
	if _, err := c.GetTransactionReceipt("0xdead"); err == nil {
		t.Fatal("null result must error (receipt not found)")
	}
}
```

- [ ] **Step 2: Add the `urpc` test seam** — the fake transport needs a constructor that injects a custom `rpcTransport`. In `clients/urpc/client.go`, add (the `rpcTransport` interface and `Client.rpcClient` field already exist):

```go
// NewClientWithTransport builds a Client around a custom rpcTransport. Intended
// for tests that replace the network transport with a fake.
func NewClientWithTransport(t rpcTransport) *Client {
	return &Client{rpcClient: t}
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./clients/ethclient/ -run TestGetTransactionReceipt -v`
Expected: FAIL — `undefined: c.GetTransactionReceipt` (and `NewClientWithTransport` until Step 2 compiles).

- [ ] **Step 4: Implement `GetTransactionReceipt`** — add to `clients/ethclient/methods.go` (the const `ethGetTransactionReceipt` already exists at the top of the file):

```go
// GetTransactionReceipt returns the receipt for a mined transaction, including
// its event logs. Returns ErrTransactionNotFound if the node returns null
// (e.g. the tx is still pending or unknown).
func (c *Client) GetTransactionReceipt(txHash string) (*Receipt, error) {
	req := urpc.NewRequest(ethGetTransactionReceipt)
	req.AddParams(txHash)
	result, err := c.rpcClient.Call(req)
	if err != nil {
		return nil, err
	}
	if result.Result == nil || string(result.Result) == "null" {
		return nil, ErrTransactionNotFound
	}
	receipt := new(Receipt)
	if err := result.ParseResult(receipt); err != nil {
		return nil, err
	}
	return receipt, nil
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./clients/ethclient/ -run TestGetTransactionReceipt -v`
Expected: PASS (both).

- [ ] **Step 6: Commit**

```bash
git add clients/ethclient/methods.go clients/urpc/client.go clients/ethclient/methods_logs_test.go
git commit -m "feat(ethclient): GetTransactionReceipt + urpc test transport seam"
```

---

### Task 4: `eth_getLogs` — `LogFilter` + `GetLogs`

**Files:**
- Modify: `clients/ethclient/methods.go`
- Test: `clients/ethclient/methods_logs_test.go` (append)

- [ ] **Step 1: Write the failing test** — append to `clients/ethclient/methods_logs_test.go`:

```go
func TestGetLogs_RequestParamsAndDecode(t *testing.T) {
	c, ft := newFakeClient("[" + sampleLogJSON + "]")
	logs, err := c.GetLogs(LogFilter{
		FromBlock: 1000,
		ToBlock:   2000,
		Addresses: []string{"0xdac17f958d2ee523a2206206994597c13d831ec7"},
		Topics:    []string{"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if ft.lastMethod != "eth_getLogs" {
		t.Fatalf("method=%q", ft.lastMethod)
	}
	// The filter param must carry hex block bounds and the address/topics.
	params := string(ft.lastParams)
	for _, want := range []string{"0x3e8", "0x7d0", "0xdac17f958d2ee523a2206206994597c13d831ec7", "ddf252ad"} {
		if !contains(params, want) {
			t.Fatalf("params %s missing %q", params, want)
		}
	}
	if len(logs) != 1 || logs[0].BlockNumber != 0x10d4f {
		t.Fatalf("logs not decoded: %+v", logs)
	}
}

func TestGetLogs_Empty(t *testing.T) {
	c, _ := newFakeClient("[]")
	logs, err := c.GetLogs(LogFilter{FromBlock: 1, ToBlock: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 0 {
		t.Fatalf("want no logs, got %d", len(logs))
	}
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(h, n string) int {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./clients/ethclient/ -run TestGetLogs -v`
Expected: FAIL — `undefined: LogFilter` / `c.GetLogs`.

- [ ] **Step 3: Implement `LogFilter` + `GetLogs`** — add to `clients/ethclient/methods.go`, and add the const `ethGetLogs = "eth_getLogs"` to the const block at the top of the file (next to `ethCall`):

```go
// LogFilter describes an eth_getLogs query. Block bounds are inclusive; a zero
// ToBlock is treated as "latest". Addresses and Topics are optional filters.
type LogFilter struct {
	FromBlock int64
	ToBlock   int64
	Addresses []string
	Topics    []string
}

// GetLogs queries event logs matching the filter via eth_getLogs.
func (c *Client) GetLogs(filter LogFilter) ([]*Log, error) {
	param := map[string]interface{}{
		"fromBlock": hexnum.Int64ToHex(filter.FromBlock),
	}
	if filter.ToBlock > 0 {
		param["toBlock"] = hexnum.Int64ToHex(filter.ToBlock)
	} else {
		param["toBlock"] = tagBlockLatest
	}
	if len(filter.Addresses) > 0 {
		param["address"] = filter.Addresses
	}
	if len(filter.Topics) > 0 {
		// eth_getLogs topics is a positional array; each position may itself be
		// an array (OR). We pass topic[0] filtering as a flat list here.
		param["topics"] = filter.Topics
	}
	req := urpc.NewRequest(ethGetLogs)
	req.AddParams(param)
	result, err := c.rpcClient.Call(req)
	if err != nil {
		return nil, err
	}
	if result.Result == nil || string(result.Result) == "null" {
		return nil, nil
	}
	var logs []*Log
	if err := result.ParseResult(&logs); err != nil {
		return nil, err
	}
	return logs, nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./clients/ethclient/ -run TestGetLogs -v`
Expected: PASS (both).

- [ ] **Step 5: Commit**

```bash
git add clients/ethclient/methods.go clients/ethclient/methods_logs_test.go
git commit -m "feat(ethclient): eth_getLogs with LogFilter"
```

---

### Task 5: ABI `DecodeOutputs` — decode return data against a method's outputs

**Files:**
- Create: `abi/output.go`
- Test: `abi/output_test.go`

- [ ] **Step 1: Write the failing test** — create `abi/output_test.go`:

```go
package abi

import (
	"math/big"
	"testing"
)

func TestDecodeOutputs_SingleUint(t *testing.T) {
	e := &SmartContractAbiEntry{
		Name: "balanceOf",
		Type: "function",
		Outputs: []*SmartContractAbiEntryOutput{
			{Type: "uint256"},
		},
	}
	// abi.encode(uint256(1000))
	data, err := encodeParams([]abiType{{kind: kindUint, size: 256}}, []any{big.NewInt(1000)})
	if err != nil {
		t.Fatal(err)
	}
	vals, err := e.DecodeOutputs(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 1 || vals[0].Value.(*big.Int).Int64() != 1000 {
		t.Fatalf("outputs=%+v", vals)
	}
}

func TestDecodeOutputs_MultipleWithNames(t *testing.T) {
	e := &SmartContractAbiEntry{
		Name: "getReserves",
		Type: "function",
		Outputs: []*SmartContractAbiEntryOutput{
			{Name: "reserve0", Type: "uint112"},
			{Name: "reserve1", Type: "uint112"},
		},
	}
	data, err := encodeParams(
		[]abiType{{kind: kindUint, size: 112}, {kind: kindUint, size: 112}},
		[]any{big.NewInt(7), big.NewInt(8)},
	)
	if err != nil {
		t.Fatal(err)
	}
	vals, err := e.DecodeOutputs(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 2 {
		t.Fatalf("outputs=%d", len(vals))
	}
	if vals[0].Name != "reserve0" || vals[0].Value.(*big.Int).Int64() != 7 {
		t.Fatalf("out0=%+v", vals[0])
	}
	if vals[1].Name != "reserve1" || vals[1].Value.(*big.Int).Int64() != 8 {
		t.Fatalf("out1=%+v", vals[1])
	}
}

func TestDecodeOutputs_DynamicArray(t *testing.T) {
	e := &SmartContractAbiEntry{
		Name: "balanceOfBatch",
		Type: "function",
		Outputs: []*SmartContractAbiEntryOutput{
			{Type: "uint256[]"},
		},
	}
	arr, _ := parseType("uint256[]", nil)
	data, err := encodeParams([]abiType{arr}, []any{[]any{big.NewInt(1), big.NewInt(2)}})
	if err != nil {
		t.Fatal(err)
	}
	vals, err := e.DecodeOutputs(data)
	if err != nil {
		t.Fatal(err)
	}
	got := vals[0].Value.([]DecodedValue)
	if len(got) != 2 || got[1].Value.(*big.Int).Int64() != 2 {
		t.Fatalf("array out=%+v", got)
	}
}

func TestDecodeOutputs_NoOutputs(t *testing.T) {
	e := &SmartContractAbiEntry{Name: "doThing", Type: "function"}
	vals, err := e.DecodeOutputs(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 0 {
		t.Fatalf("want 0 outputs, got %d", len(vals))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./abi/ -run TestDecodeOutputs -v`
Expected: FAIL — `e.DecodeOutputs undefined`.

- [ ] **Step 3: Write minimal implementation** — create `abi/output.go`:

```go
package abi

// DecodeOutputs decodes ABI-encoded return data against this entry's declared
// outputs, returning the values in declaration order with their names. Unlike
// DecodeInputsTyped there is no 4-byte selector to skip — eth_call returns the
// bare encoded output block. Tuple outputs are not supported (the output model
// carries no components); such an ABI is rejected at import time (M3).
func (e *SmartContractAbiEntry) DecodeOutputs(data []byte) ([]DecodedValue, error) {
	if len(e.Outputs) == 0 {
		return nil, nil
	}
	types := make([]abiType, len(e.Outputs))
	for i, out := range e.Outputs {
		typ, err := parseType(out.Type, nil)
		if err != nil {
			return nil, err
		}
		types[i] = typ
	}
	vals, err := decodeParams(types, data)
	if err != nil {
		return nil, err
	}
	for i := range vals {
		if i < len(e.Outputs) {
			vals[i].Name = e.Outputs[i].Name
		}
	}
	return vals, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./abi/ -run TestDecodeOutputs -v`
Expected: PASS (all four).

- [ ] **Step 5: Commit**

```bash
git add abi/output.go abi/output_test.go
git commit -m "feat(abi): DecodeOutputs — decode eth_call return data by method outputs"
```

---

### Task 6: `CallMethod` — generic view-method call on the client

**Files:**
- Create: `clients/ethclient/call_method.go`
- Test: `clients/ethclient/call_method_test.go`

- [ ] **Step 1: Write the failing test** — create `clients/ethclient/call_method_test.go`:

```go
package ethclient

import (
	"math/big"
	"testing"

	"github.com/ITProLabDev/ethbacknode/abi"
)

// newCallClient wires a fake transport (canned eth_call hex result) and a real
// abi manager holding one contract, so CallMethod can be exercised offline.
func newCallClient(t *testing.T, contractAddr, rawABI, callResultHex string) *Client {
	t.Helper()
	mgr := abi.NewManager(
		abi.WithStorage(newMemABIStorage()),
		abi.WithAddressCodec(GetAddressCodec()),
	)
	if err := mgr.Init(); err != nil {
		t.Fatal(err)
	}
	c, err := abi.NewContractFromABI("C", "C", contractAddr, []byte(rawABI))
	if err != nil {
		t.Fatal(err)
	}
	mgr.Add(c)

	ft := &fakeTransport{result: jsonString(callResultHex)}
	return &Client{
		rpcClient:    urpcClientWith(ft),
		abi:          mgr,
		addressCodec: GetAddressCodec(),
	}
}

func TestCallMethod_DecodesUintResult(t *testing.T) {
	rawABI := `[{"type":"function","name":"balanceOf","stateMutability":"view",
	  "inputs":[{"name":"a","type":"address"}],"outputs":[{"name":"","type":"uint256"}]}]`
	// eth_call returns abi.encode(uint256(1000)) = 0x...03e8
	resultHex := "0x00000000000000000000000000000000000000000000000000000000000003e8"
	c := newCallClient(t, "0xabc0000000000000000000000000000000000000", rawABI, resultHex)

	addrBytes, _ := GetAddressCodec().DecodeAddressToBytes("0x1111111111111111111111111111111111111111")
	out, err := c.CallMethod("0xabc0000000000000000000000000000000000000", "balanceOf", addrBytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Value.(*big.Int).Int64() != 1000 {
		t.Fatalf("call output=%+v", out)
	}
}

func TestCallMethod_UnknownContract(t *testing.T) {
	rawABI := `[{"type":"function","name":"x","inputs":[],"outputs":[]}]`
	c := newCallClient(t, "0xabc0000000000000000000000000000000000000", rawABI, "0x")
	if _, err := c.CallMethod("0x9999999999999999999999999999999999999999", "x"); err == nil {
		t.Fatal("unknown contract must error")
	}
}

func TestCallMethod_UnknownMethod(t *testing.T) {
	rawABI := `[{"type":"function","name":"x","inputs":[],"outputs":[]}]`
	c := newCallClient(t, "0xabc0000000000000000000000000000000000000", rawABI, "0x")
	if _, err := c.CallMethod("0xabc0000000000000000000000000000000000000", "nope"); err == nil {
		t.Fatal("unknown method must error")
	}
}
```

This test references three small helpers that must be added to `clients/ethclient/methods_logs_test.go` (so they are shared across the package's tests). Add them there:

```go
// jsonString wraps a raw JSON string value as a json.RawMessage result.
func jsonString(hexResult string) json.RawMessage {
	b, _ := json.Marshal(hexResult)
	return b
}

// urpcClientWith builds a urpc.Client around a fake transport (test seam).
func urpcClientWith(ft *fakeTransport) *urpc.Client {
	return urpc.NewClientWithTransport(ft)
}

// newMemABIStorage returns an in-memory BinStorage for the abi manager.
func newMemABIStorage() *memABIStore { return &memABIStore{} }

type memABIStore struct {
	data   []byte
	exists bool
}

func (s *memABIStore) IsExists() bool          { return s.exists }
func (s *memABIStore) Save(b []byte) error     { s.data = append(s.data[:0], b...); s.exists = true; return nil }
func (s *memABIStore) Load() ([]byte, error)   { return s.data, nil }
```

(Confirm `abi.NewManager`, `abi.WithStorage`, `abi.WithAddressCodec`, `GetAddressCodec()`, and `DecodeAddressToBytes` exist — they do, used in `main.go` and the abi tests.)

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./clients/ethclient/ -run TestCallMethod -v`
Expected: FAIL — `c.CallMethod undefined`.

- [ ] **Step 3: Implement `CallMethod`** — create `clients/ethclient/call_method.go`:

```go
package ethclient

import (
	"github.com/ITProLabDev/ethbacknode/abi"
	"github.com/ITProLabDev/ethbacknode/common/hexnum"
)

// CallMethod invokes a contract view/pure method by name via eth_call and
// returns its decoded outputs. The contract must be registered in the ABI
// manager (so its ABI is known). args are the typed method inputs, matching
// the engine's per-kind Go types (address/bytesN/bytes -> []byte, bool -> bool,
// uint*/int* -> *big.Int|int|int64|uint64, string -> string, arrays/tuples ->
// []any). Returns the decoded return values in declaration order.
func (c *Client) CallMethod(contractAddress, methodName string, args ...any) ([]abi.DecodedValue, error) {
	contract, err := c.abi.GetSmartContractByAddress(contractAddress)
	if err != nil {
		return nil, err
	}
	if contract.Abi == nil {
		return nil, abi.ErrUnknownContract
	}
	method, err := contract.Abi.GetMethodByName(methodName)
	if err != nil {
		return nil, err
	}
	callData, err := method.EncodeInputsTyped(args...)
	if err != nil {
		return nil, err
	}
	resultHex, err := c.Call(contractAddress, hexnum.BytesToHex(callData))
	if err != nil {
		return nil, err
	}
	resultBytes, err := hexnum.ParseHexBytes(resultHex)
	if err != nil {
		return nil, err
	}
	return method.DecodeOutputs(resultBytes)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./clients/ethclient/ -run TestCallMethod -v`
Expected: PASS (all three).

- [ ] **Step 5: Commit**

```bash
git add clients/ethclient/call_method.go clients/ethclient/call_method_test.go clients/ethclient/methods_logs_test.go
git commit -m "feat(ethclient): generic CallMethod — encode call-data, eth_call, decode outputs"
```

---

### Task 7: Final verification & docs

**Files:**
- Modify: `todo/TASKS.md`

- [ ] **Step 1: Run the affected packages with the race detector**

Run: `go test ./clients/ethclient/ ./clients/urpc/ ./abi/ -race -count=1`
Expected: `ok` for all three (ignore `[CRIT] Duplicated Contract` log lines from the abi concurrency test; the pre-existing `clients/urpc` `TestIPCClient` prints a lot — that's fine).

- [ ] **Step 2: Confirm nothing else broke**

Run: `go build ./... && go vet ./clients/ethclient/ ./clients/urpc/ ./abi/`
Expected: both exit 0.

- [ ] **Step 3: Mark M4 tasks done in `todo/TASKS.md`**

Change each of `M4.1`–`M4.4` from `- [ ]` to `- [x]` in the "Milestone 4" section, and append " ✅ DONE" to the "## Milestone 4 — Client: receipts & logs (`clients/ethclient/`)" header line, matching M1/M2/M3. Mapping:
- M4.1 `GetTransactionReceipt` + Receipt/Log types → Tasks 1, 2, 3
- M4.2 `eth_getLogs` / `GetLogs(LogFilter)` → Task 4
- M4.3 generalized `eth_call` reads via `CallMethod` → Tasks 5, 6
- M4.4 tests against fake-transport fixtures (IPC/HTTP-agnostic) → Tasks 1–6 tests

Add a short status block under the header summarizing what landed (mirror the M3 style).

- [ ] **Step 4: Commit**

```bash
git add todo/TASKS.md
git commit -m "docs(ethclient): mark M4 (receipts, logs, CallMethod) complete"
```

---

## Notes for the implementer

- **Follow the proxy-map hex idiom** for every wire struct: decode into `map[string]json.RawMessage`, then per-field. Never decode 0x-hex directly into an int/`*big.Int` field.
- **Do NOT add go-ethereum** or any external RPC/ABI library. Use `urpc` + the project `abi/` engine + `common/hexnum`.
- The `urpc.NewClientWithTransport` seam (Task 3) is the only `urpc` change; it exists purely so RPC methods are testable without a node. Keep it minimal.
- `DecodeOutputs` (Task 5) deliberately does NOT support tuple outputs — that's deferred (the output struct has no `Components`; the importer already rejects tuple outputs in M3). Do not add tuple-output handling here.
- `Topics32()` is the bridge from the wire `[]string` topics to the `[][32]byte` that `abi.DecodeLog` consumes — M5 will use `receipt.Logs[i].Topics32()` + `log.Data` to call `manager.DecodeLog`.
- Run only the three affected packages' tests; the repo has pre-existing unrelated compile errors in `crypto/secp256k1`. The abi concurrency test prints `[CRIT] Duplicated Contract` (harmless); `clients/urpc`'s `TestIPCClient` is verbose (harmless).
- These methods are the read primitives M5's `eventlog` service consumes (GetLogs per block / GetTransactionReceipt per tx) and M6's `contractCall` RPC wraps (`CallMethod`).
