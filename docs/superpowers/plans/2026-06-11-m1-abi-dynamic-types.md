# M1 — ABI Engine: Dynamic Types & Tuples Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the project's own ABI engine (`abi/`) to encode and decode the full set of Solidity ABI types — sized integers, fixed/dynamic bytes, strings, arrays, and tuples — using correct head/tail offset encoding, exposing results through a neutral decoded-value model.

**Architecture:** Add a self-contained typed codec **alongside** the existing static-only path (`_parseParam`/`paramInput`), so the ERC-20 high-level methods (`Erc20IsTransfer`, `Erc20DecodeIfTransfer`, `Erc20CallGetBalance`, `Erc20DecodeAmount`) keep working unchanged. The new engine is three files: a type model/parser (`abitype.go`), a head/tail codec (`codec.go`), and a neutral decoded-value model (`value.go`). A `Components` field is added to the ABI input struct for tuples, and the method-selector hash is switched to a canonical type string (a no-op for elementary types, a fix for tuples).

**Tech Stack:** Go 1.24, `math/big`, the existing `crypto.Keccak256`, standard `testing`.

---

## Background & Constraints

**Must not break (public API, called from `clients/ethclient` and `main`):**
`SmartContractsManager`, `NewManager`, `Erc20CallGetBalance`, `Erc20DecodeAmount`, `Erc20IsTransfer`, `Erc20DecodeIfTransfer`, `abi.ErrUnknownContract`.

**Safe to build on (used only inside `abi/`):** the codec internals
`DecodeInputs`, `encodeInputs`, `encodeInputsBytes`, `_parseParam`, `paramInput`.
We leave them in place and add a parallel typed path.

**ABI encoding rules implemented here (Solidity ABI spec):**
- A type is *static* (fixed width, no length prefix) or *dynamic*.
- Dynamic: `bytes`, `string`, `T[]`, `T[N]` when `T` is dynamic, and any tuple
  containing a dynamic component. Everything else is static.
- A sequence of values is encoded as a **head** section followed by a **tail**.
  Each static value is placed inline in the head. Each dynamic value places a
  32-byte byte-offset in the head (measured from the start of the sequence) and
  appends its full encoding to the tail.

**Canonical test vectors (from the Solidity ABI docs) used below:**
- `baz(uint32,bool)` with `(69,true)` →
  `0000…0045` ‖ `0000…0001`.
- `sam(bytes,bool,uint256[])` with `("dave",true,[1,2,3])` →
  head `0000…0060` ‖ `0000…0001` ‖ `0000…00a0`,
  tail `0000…0004` ‖ `6461766500…00` ‖ `0000…0003` ‖ `0000…0001` ‖ `0000…0002` ‖ `0000…0003`.
  (`"dave"` = bytes `64 61 76 65`.)

---

## File Structure

- **Create `abi/abitype.go`** — `kind` enum, `abiType` struct, `parseType`,
  `(abiType).isDynamic`, `(abiType).staticSize`, `(abiType).canonical`.
- **Create `abi/value.go`** — `DecodedValue`, `DecodedCall`, `DecodedEvent`.
- **Create `abi/codec.go`** — `encodeParams`, `decodeParams`, and the per-type
  `encodeValue`/`decodeValue` helpers (the head/tail algorithm).
- **Create `abi/abitype_test.go`**, **`abi/codec_test.go`** — unit tests with the
  canonical vectors and round-trips.
- **Modify `abi/smartcontractabi.go`** — add `Components` to
  `SmartContractAbiEntryInput`; add `(*SmartContractAbiEntry).typedInputs()`
  helper, `DecodeInputsTyped`, `EncodeInputsTyped`; switch `updateSignature` to
  use `canonical`.

---

### Task 1: Type model and parser (`abitype.go`)

**Files:**
- Modify: `abi/smartcontractabi.go` (add `Components` to the input struct)
- Create: `abi/abitype.go`
- Test: `abi/abitype_test.go`

- [ ] **Step 0: Add the `Components` field to the input struct** (prerequisite — the tuple parser reads it)

In `abi/smartcontractabi.go`, replace:

```go
type SmartContractAbiEntryInput struct {
	Name    string `json:"name,omitempty"`
	Type    string `json:"type"`
	Indexed bool   `json:"indexed,omitempty"`
	data    []byte
}
```

with:

```go
type SmartContractAbiEntryInput struct {
	Name       string                        `json:"name,omitempty"`
	Type       string                        `json:"type"`
	Indexed    bool                          `json:"indexed,omitempty"`
	Components []*SmartContractAbiEntryInput `json:"components,omitempty"`
	data       []byte
}
```

Run `go build ./abi/` — expected: builds cleanly (field added, nothing uses it yet).

- [ ] **Step 1: Write the failing tests**

```go
package abi

import "testing"

func TestParseType_Elementary(t *testing.T) {
	cases := []struct {
		in       string
		wantKind kind
		wantSize int
		wantDyn  bool
		wantCanon string
	}{
		{"uint256", kindUint, 256, false, "uint256"},
		{"uint", kindUint, 256, false, "uint256"},
		{"uint8", kindUint, 8, false, "uint8"},
		{"int256", kindInt, 256, false, "int256"},
		{"int128", kindInt, 128, false, "int128"},
		{"bool", kindBool, 0, false, "bool"},
		{"address", kindAddress, 0, false, "address"},
		{"bytes32", kindFixedBytes, 32, false, "bytes32"},
		{"bytes1", kindFixedBytes, 1, false, "bytes1"},
		{"bytes", kindBytes, 0, true, "bytes"},
		{"string", kindString, 0, true, "string"},
	}
	for _, c := range cases {
		typ, err := parseType(c.in, nil)
		if err != nil {
			t.Fatalf("%s: %v", c.in, err)
		}
		if typ.kind != c.wantKind || typ.size != c.wantSize {
			t.Fatalf("%s: kind=%d size=%d", c.in, typ.kind, typ.size)
		}
		if typ.isDynamic() != c.wantDyn {
			t.Fatalf("%s: isDynamic=%v want %v", c.in, typ.isDynamic(), c.wantDyn)
		}
		if typ.canonical() != c.wantCanon {
			t.Fatalf("%s: canonical=%q want %q", c.in, typ.canonical(), c.wantCanon)
		}
	}
}

func TestParseType_Arrays(t *testing.T) {
	// uint256[] dynamic slice
	sl, err := parseType("uint256[]", nil)
	if err != nil {
		t.Fatal(err)
	}
	if sl.kind != kindSlice || !sl.isDynamic() || sl.elem.kind != kindUint {
		t.Fatalf("uint256[]: %+v", sl)
	}
	if sl.canonical() != "uint256[]" {
		t.Fatalf("canonical=%q", sl.canonical())
	}
	// address[3] fixed array of a static type → static
	fa, err := parseType("address[3]", nil)
	if err != nil {
		t.Fatal(err)
	}
	if fa.kind != kindArray || fa.size != 3 || fa.isDynamic() {
		t.Fatalf("address[3]: %+v dyn=%v", fa, fa.isDynamic())
	}
	if fa.staticSize() != 96 {
		t.Fatalf("address[3] staticSize=%d want 96", fa.staticSize())
	}
	// bytes[2] fixed array of a dynamic type → dynamic
	bd, err := parseType("bytes[2]", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bd.isDynamic() {
		t.Fatal("bytes[2] must be dynamic")
	}
	if bd.canonical() != "bytes[2]" {
		t.Fatalf("canonical=%q", bd.canonical())
	}
}

func TestParseType_Tuple(t *testing.T) {
	comps := []*SmartContractAbiEntryInput{
		{Name: "a", Type: "uint256"},
		{Name: "b", Type: "address"},
	}
	tup, err := parseType("tuple", comps)
	if err != nil {
		t.Fatal(err)
	}
	if tup.kind != kindTuple || len(tup.fields) != 2 || tup.isDynamic() {
		t.Fatalf("tuple: %+v dyn=%v", tup, tup.isDynamic())
	}
	if tup.canonical() != "(uint256,address)" {
		t.Fatalf("canonical=%q", tup.canonical())
	}
	// dynamic tuple
	comps2 := []*SmartContractAbiEntryInput{
		{Name: "a", Type: "uint256"},
		{Name: "s", Type: "string"},
	}
	tup2, err := parseType("tuple", comps2)
	if err != nil {
		t.Fatal(err)
	}
	if !tup2.isDynamic() || tup2.canonical() != "(uint256,string)" {
		t.Fatalf("tuple2: dyn=%v canon=%q", tup2.isDynamic(), tup2.canonical())
	}
	// tuple[] slice of tuples
	ts, err := parseType("tuple[]", comps)
	if err != nil {
		t.Fatal(err)
	}
	if ts.kind != kindSlice || ts.elem.kind != kindTuple || ts.canonical() != "(uint256,address)[]" {
		t.Fatalf("tuple[]: %+v canon=%q", ts, ts.canonical())
	}
}

func TestParseType_Errors(t *testing.T) {
	for _, bad := range []string{"", "uint9", "bytes33", "uint256[", "uint256[-1]", "nope"} {
		if _, err := parseType(bad, nil); err == nil {
			t.Fatalf("%q must error", bad)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./abi/ -run TestParseType -v`
Expected: FAIL — `undefined: parseType`, `kind`, `kindUint`, etc.

- [ ] **Step 3: Write minimal implementation**

```go
package abi

import (
	"fmt"
	"strconv"
	"strings"
)

// kind enumerates the ABI type families the typed codec understands.
type kind int

const (
	kindUint kind = iota
	kindInt
	kindBool
	kindAddress
	kindFixedBytes // bytesN, 1..32
	kindBytes      // dynamic bytes
	kindString
	kindArray // T[N]
	kindSlice // T[]
	kindTuple
)

// abiType is a parsed ABI type. For uint/int, size is the bit width; for
// fixedBytes, size is N; for array, size is the fixed length. elem is the
// element type for array/slice; fields/names describe tuple components.
type abiType struct {
	kind   kind
	size   int
	elem   *abiType
	fields []abiType
	names  []string
}

// parseType parses a Solidity ABI type string. For tuples (type "tuple",
// "tuple[]", "tuple[N]"), components supplies the component definitions.
func parseType(s string, components []*SmartContractAbiEntryInput) (abiType, error) {
	if s == "" {
		return abiType{}, fmt.Errorf("%w: empty type", ErrInvalidParamsData)
	}
	// Array/slice suffix at the outermost level.
	if strings.HasSuffix(s, "]") {
		open := strings.LastIndexByte(s, '[')
		if open < 0 {
			return abiType{}, fmt.Errorf("%w: malformed array %q", ErrInvalidParamsData, s)
		}
		inner := s[:open]
		between := s[open+1 : len(s)-1]
		elem, err := parseType(inner, components)
		if err != nil {
			return abiType{}, err
		}
		if between == "" {
			return abiType{kind: kindSlice, elem: &elem}, nil
		}
		n, err := strconv.Atoi(between)
		if err != nil || n <= 0 {
			return abiType{}, fmt.Errorf("%w: bad array size %q", ErrInvalidParamsData, between)
		}
		return abiType{kind: kindArray, size: n, elem: &elem}, nil
	}

	switch {
	case s == "bool":
		return abiType{kind: kindBool}, nil
	case s == "address":
		return abiType{kind: kindAddress}, nil
	case s == "string":
		return abiType{kind: kindString}, nil
	case s == "bytes":
		return abiType{kind: kindBytes}, nil
	case strings.HasPrefix(s, "bytes"):
		n, err := strconv.Atoi(s[len("bytes"):])
		if err != nil || n < 1 || n > 32 {
			return abiType{}, fmt.Errorf("%w: bad fixed bytes %q", ErrInvalidParamsData, s)
		}
		return abiType{kind: kindFixedBytes, size: n}, nil
	case strings.HasPrefix(s, "uint"):
		return parseIntType(s, "uint", kindUint)
	case strings.HasPrefix(s, "int"):
		return parseIntType(s, "int", kindInt)
	case s == "tuple":
		return parseTuple(components)
	}
	return abiType{}, fmt.Errorf("%w: unknown type %q", ErrInvalidParamsData, s)
}

func parseIntType(s, prefix string, k kind) (abiType, error) {
	rest := s[len(prefix):]
	if rest == "" {
		return abiType{kind: k, size: 256}, nil
	}
	n, err := strconv.Atoi(rest)
	if err != nil || n < 8 || n > 256 || n%8 != 0 {
		return abiType{}, fmt.Errorf("%w: bad int width %q", ErrInvalidParamsData, s)
	}
	return abiType{kind: k, size: n}, nil
}

func parseTuple(components []*SmartContractAbiEntryInput) (abiType, error) {
	t := abiType{kind: kindTuple}
	for _, c := range components {
		ft, err := parseType(c.Type, c.Components)
		if err != nil {
			return abiType{}, err
		}
		t.fields = append(t.fields, ft)
		t.names = append(t.names, c.Name)
	}
	return t, nil
}

// isDynamic reports whether the type has no fixed encoded width.
func (t abiType) isDynamic() bool {
	switch t.kind {
	case kindString, kindBytes, kindSlice:
		return true
	case kindArray:
		return t.elem.isDynamic()
	case kindTuple:
		for _, f := range t.fields {
			if f.isDynamic() {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// staticSize returns the encoded byte width of a static type. Result is
// undefined (0) for dynamic types; callers must check isDynamic first.
func (t abiType) staticSize() int {
	switch t.kind {
	case kindArray:
		return t.size * t.elem.staticSize()
	case kindTuple:
		total := 0
		for _, f := range t.fields {
			total += f.staticSize()
		}
		return total
	default:
		return 32
	}
}

// canonical returns the canonical type string used in function/event
// signatures (e.g. "uint256", "(uint256,address)[]").
func (t abiType) canonical() string {
	switch t.kind {
	case kindUint:
		return "uint" + strconv.Itoa(t.size)
	case kindInt:
		return "int" + strconv.Itoa(t.size)
	case kindBool:
		return "bool"
	case kindAddress:
		return "address"
	case kindFixedBytes:
		return "bytes" + strconv.Itoa(t.size)
	case kindBytes:
		return "bytes"
	case kindString:
		return "string"
	case kindSlice:
		return t.elem.canonical() + "[]"
	case kindArray:
		return t.elem.canonical() + "[" + strconv.Itoa(t.size) + "]"
	case kindTuple:
		parts := make([]string, len(t.fields))
		for i, f := range t.fields {
			parts[i] = f.canonical()
		}
		return "(" + strings.Join(parts, ",") + ")"
	default:
		return ""
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./abi/ -run TestParseType -v`
Expected: PASS (all four tests).

- [ ] **Step 5: Commit**

```bash
git add abi/abitype.go abi/abitype_test.go
git commit -m "feat(abi): add typed ABI type model and parser"
```

---

### Task 2: Decoded-value model (`value.go`)

**Files:**
- Create: `abi/value.go`
- Test: covered indirectly by Task 3+ (no standalone test — these are plain
  data structs with no behavior).

- [ ] **Step 1: Write minimal implementation**

```go
package abi

// DecodedValue is one decoded ABI value. Value holds one of:
// *big.Int (uint*/int*), bool, []byte (address/bytesN/bytes),
// string, or []DecodedValue (arrays, slices, tuples).
type DecodedValue struct {
	Name  string `json:"name,omitempty"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

// DecodedCall is a decoded method invocation: the method name and its inputs.
type DecodedCall struct {
	Method string         `json:"method"`
	Inputs []DecodedValue `json:"inputs"`
}

// DecodedEvent is a decoded log event: its name, the emitting contract, and
// the decoded indexed + non-indexed parameters in ABI order.
type DecodedEvent struct {
	Name     string         `json:"name"`
	Contract string         `json:"contract,omitempty"`
	Inputs   []DecodedValue `json:"inputs"`
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./abi/`
Expected: builds cleanly (no test yet — structs only).

- [ ] **Step 3: Commit**

```bash
git add abi/value.go
git commit -m "feat(abi): add neutral decoded-value model"
```

---

### Task 3: Static encode/decode + head/tail for elementary types (`codec.go`)

**Files:**
- Create: `abi/codec.go`
- Test: `abi/codec_test.go`

- [ ] **Step 1: Write the failing tests** (canonical `baz`, plus int/round-trip)

```go
package abi

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"
)

func hexToBytes(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.ReplaceAll(s, " ", ""))
	if err != nil {
		t.Fatalf("bad hex: %v", err)
	}
	return b
}

func TestEncodeParams_Baz_Static(t *testing.T) {
	// baz(uint32,bool) with (69,true)
	types := []abiType{{kind: kindUint, size: 32}, {kind: kindBool}}
	out, err := encodeParams(types, []any{big.NewInt(69), true})
	if err != nil {
		t.Fatal(err)
	}
	want := hexToBytes(t,
		"0000000000000000000000000000000000000000000000000000000000000045"+
			"0000000000000000000000000000000000000000000000000000000000000001")
	if !bytes.Equal(out, want) {
		t.Fatalf("got  %x\nwant %x", out, want)
	}
}

func TestDecodeParams_Baz_Static(t *testing.T) {
	types := []abiType{{kind: kindUint, size: 32}, {kind: kindBool}}
	data := hexToBytes(t,
		"0000000000000000000000000000000000000000000000000000000000000045"+
			"0000000000000000000000000000000000000000000000000000000000000001")
	vals, err := decodeParams(types, data)
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 2 {
		t.Fatalf("got %d values", len(vals))
	}
	if vals[0].Value.(*big.Int).Int64() != 69 {
		t.Fatalf("v0=%v", vals[0].Value)
	}
	if vals[1].Value.(bool) != true {
		t.Fatalf("v1=%v", vals[1].Value)
	}
}

func TestEncodeDecode_IntNegative_RoundTrip(t *testing.T) {
	types := []abiType{{kind: kindInt, size: 256}}
	for _, n := range []int64{-1, -255, -1 << 40, 0, 1, 1 << 40} {
		out, err := encodeParams(types, []any{big.NewInt(n)})
		if err != nil {
			t.Fatal(err)
		}
		vals, err := decodeParams(types, out)
		if err != nil {
			t.Fatal(err)
		}
		if vals[0].Value.(*big.Int).Int64() != n {
			t.Fatalf("n=%d round-tripped to %v", n, vals[0].Value)
		}
	}
}

func TestEncodeDecode_AddressAndFixedBytes(t *testing.T) {
	addr := hexToBytes(t, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	b4 := hexToBytes(t, "01020304")
	types := []abiType{{kind: kindAddress}, {kind: kindFixedBytes, size: 4}}
	out, err := encodeParams(types, []any{addr, b4})
	if err != nil {
		t.Fatal(err)
	}
	// address right-aligned in slot 0; bytes4 left-aligned in slot 1
	if !bytes.Equal(out[12:32], addr) {
		t.Fatalf("address slot: %x", out[0:32])
	}
	if !bytes.Equal(out[32:36], b4) {
		t.Fatalf("bytes4 slot: %x", out[32:64])
	}
	vals, err := decodeParams(types, out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(vals[0].Value.([]byte), addr) {
		t.Fatalf("decoded addr %x", vals[0].Value)
	}
	if !bytes.Equal(vals[1].Value.([]byte), b4) {
		t.Fatalf("decoded bytes4 %x", vals[1].Value)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./abi/ -run 'TestEncodeParams_Baz|TestDecodeParams_Baz|TestEncodeDecode_Int|TestEncodeDecode_Address' -v`
Expected: FAIL — `undefined: encodeParams`, `decodeParams`.

- [ ] **Step 3: Write minimal implementation** (full head/tail engine; later tasks extend the switch arms)

```go
package abi

import (
	"fmt"
	"math/big"
)

var bigOne = big.NewInt(1)

// leftPad32 right-aligns b into a 32-byte slot.
func leftPad32(b []byte) []byte {
	out := make([]byte, 32)
	if len(b) >= 32 {
		copy(out, b[len(b)-32:])
		return out
	}
	copy(out[32-len(b):], b)
	return out
}

// rightPad pads b on the right to a multiple of 32 bytes.
func rightPad(b []byte) []byte {
	if len(b)%32 == 0 {
		return b
	}
	out := make([]byte, ((len(b)/32)+1)*32)
	copy(out, b)
	return out
}

// encodeParams encodes a sequence of values using head/tail offset encoding.
func encodeParams(types []abiType, values []any) ([]byte, error) {
	if len(types) != len(values) {
		return nil, ErrSmartContractMethodParamsCountMismatch
	}
	heads := make([][]byte, len(types))
	tails := make([][]byte, len(types))
	headLen := 0
	for i := range types {
		enc, err := encodeValue(types[i], values[i])
		if err != nil {
			return nil, err
		}
		if types[i].isDynamic() {
			tails[i] = enc
			headLen += 32
		} else {
			heads[i] = enc
			headLen += len(enc)
		}
	}
	tailSeen := 0
	for i := range types {
		if types[i].isDynamic() {
			heads[i] = leftPad32(big.NewInt(int64(headLen + tailSeen)).Bytes())
			tailSeen += len(tails[i])
		}
	}
	out := make([]byte, 0, headLen+tailSeen)
	for i := range heads {
		out = append(out, heads[i]...)
	}
	for i := range tails {
		out = append(out, tails[i]...)
	}
	return out, nil
}

// encodeValue encodes a single value according to its type.
func encodeValue(t abiType, v any) ([]byte, error) {
	switch t.kind {
	case kindUint:
		n, err := asBigInt(v)
		if err != nil {
			return nil, err
		}
		return leftPad32(n.Bytes()), nil
	case kindInt:
		n, err := asBigInt(v)
		if err != nil {
			return nil, err
		}
		return encodeInt(n), nil
	case kindBool:
		b, ok := v.(bool)
		if !ok {
			return nil, fmt.Errorf("%w: bool expected", ErrInvalidParamsData)
		}
		out := make([]byte, 32)
		if b {
			out[31] = 1
		}
		return out, nil
	case kindAddress:
		b, ok := v.([]byte)
		if !ok || len(b) > 32 {
			return nil, fmt.Errorf("%w: address bytes expected", ErrInvalidParamsData)
		}
		return leftPad32(b), nil
	case kindFixedBytes:
		b, ok := v.([]byte)
		if !ok || len(b) != t.size {
			return nil, fmt.Errorf("%w: bytes%d expected", ErrInvalidParamsData, t.size)
		}
		out := make([]byte, 32)
		copy(out, b) // left-aligned
		return out, nil
	}
	return nil, fmt.Errorf("%w: unsupported type %q", ErrInvalidParamsData, t.canonical())
}

// encodeInt encodes a (possibly negative) integer as a 32-byte two's complement.
func encodeInt(n *big.Int) []byte {
	if n.Sign() >= 0 {
		return leftPad32(n.Bytes())
	}
	mod := new(big.Int).Lsh(bigOne, 256)
	tc := new(big.Int).Add(mod, n) // n is negative
	return leftPad32(tc.Bytes())
}

func asBigInt(v any) (*big.Int, error) {
	switch x := v.(type) {
	case *big.Int:
		return x, nil
	case int:
		return big.NewInt(int64(x)), nil
	case int64:
		return big.NewInt(x), nil
	case uint64:
		return new(big.Int).SetUint64(x), nil
	default:
		return nil, fmt.Errorf("%w: integer expected, got %T", ErrInvalidParamsData, v)
	}
}

// decodeParams decodes a head/tail-encoded sequence. All offsets in block are
// relative to block[0].
func decodeParams(types []abiType, block []byte) ([]DecodedValue, error) {
	out := make([]DecodedValue, len(types))
	head := 0
	for i := range types {
		if types[i].isDynamic() {
			if head+32 > len(block) {
				return nil, ErrInvalidParamsData
			}
			off := int(new(big.Int).SetBytes(block[head : head+32]).Int64())
			if off < 0 || off > len(block) {
				return nil, ErrInvalidParamsData
			}
			val, err := decodeValue(types[i], block, off)
			if err != nil {
				return nil, err
			}
			out[i] = val
			head += 32
		} else {
			sz := types[i].staticSize()
			if head+sz > len(block) {
				return nil, ErrInvalidParamsData
			}
			val, err := decodeValue(types[i], block, head)
			if err != nil {
				return nil, err
			}
			out[i] = val
			head += sz
		}
	}
	return out, nil
}

// decodeValue decodes a single value of type t located at offset at within block.
func decodeValue(t abiType, block []byte, at int) (DecodedValue, error) {
	dv := DecodedValue{Type: t.canonical()}
	switch t.kind {
	case kindUint:
		if at+32 > len(block) {
			return dv, ErrInvalidParamsData
		}
		dv.Value = new(big.Int).SetBytes(block[at : at+32])
		return dv, nil
	case kindInt:
		if at+32 > len(block) {
			return dv, ErrInvalidParamsData
		}
		dv.Value = decodeInt(block[at : at+32])
		return dv, nil
	case kindBool:
		if at+32 > len(block) {
			return dv, ErrInvalidParamsData
		}
		dv.Value = block[at+31] != 0
		return dv, nil
	case kindAddress:
		if at+32 > len(block) {
			return dv, ErrInvalidParamsData
		}
		b := make([]byte, 20)
		copy(b, block[at+12:at+32])
		dv.Value = b
		return dv, nil
	case kindFixedBytes:
		if at+32 > len(block) {
			return dv, ErrInvalidParamsData
		}
		b := make([]byte, t.size)
		copy(b, block[at:at+t.size])
		dv.Value = b
		return dv, nil
	}
	return dv, fmt.Errorf("%w: unsupported type %q", ErrInvalidParamsData, t.canonical())
}

// decodeInt reads a 32-byte two's complement integer.
func decodeInt(slot []byte) *big.Int {
	v := new(big.Int).SetBytes(slot)
	if v.Bit(255) == 1 {
		mod := new(big.Int).Lsh(bigOne, 256)
		v.Sub(v, mod)
	}
	return v
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./abi/ -run 'TestEncodeParams_Baz|TestDecodeParams_Baz|TestEncodeDecode_Int|TestEncodeDecode_Address' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add abi/codec.go abi/codec_test.go
git commit -m "feat(abi): head/tail codec for elementary static types"
```

---

### Task 4: Dynamic `bytes` and `string`

**Files:**
- Modify: `abi/codec.go` (add arms to `encodeValue`/`decodeValue`)
- Test: `abi/codec_test.go` (append)

- [ ] **Step 1: Write the failing tests**

```go
func TestEncodeDecode_BytesAndString(t *testing.T) {
	types := []abiType{{kind: kindString}, {kind: kindBytes}}
	out, err := encodeParams(types, []any{"dave", []byte("dave")})
	if err != nil {
		t.Fatal(err)
	}
	vals, err := decodeParams(types, out)
	if err != nil {
		t.Fatal(err)
	}
	if vals[0].Value.(string) != "dave" {
		t.Fatalf("string=%q", vals[0].Value)
	}
	if string(vals[1].Value.([]byte)) != "dave" {
		t.Fatalf("bytes=%q", vals[1].Value)
	}
}

func TestEncodeValue_BytesLayout(t *testing.T) {
	// "dave" → len 4 slot, then 64 61 76 65 left-aligned in a 32-byte slot.
	enc, err := encodeValue(abiType{kind: kindBytes}, []byte("dave"))
	if err != nil {
		t.Fatal(err)
	}
	want := hexToBytes(t,
		"0000000000000000000000000000000000000000000000000000000000000004"+
			"6461766500000000000000000000000000000000000000000000000000000000")
	if !bytes.Equal(enc, want) {
		t.Fatalf("got %x", enc)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./abi/ -run 'TestEncodeDecode_BytesAndString|TestEncodeValue_BytesLayout' -v`
Expected: FAIL — `unsupported type "bytes"`.

- [ ] **Step 3: Add the encode arms** to `encodeValue` (before the final `return nil, ...`)

```go
	case kindBytes:
		b, ok := v.([]byte)
		if !ok {
			return nil, fmt.Errorf("%w: bytes expected", ErrInvalidParamsData)
		}
		return append(leftPad32(big.NewInt(int64(len(b))).Bytes()), rightPad(b)...), nil
	case kindString:
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("%w: string expected", ErrInvalidParamsData)
		}
		b := []byte(s)
		return append(leftPad32(big.NewInt(int64(len(b))).Bytes()), rightPad(b)...), nil
```

- [ ] **Step 4: Add the decode arms** to `decodeValue` (before the final `return dv, ...`)

```go
	case kindBytes, kindString:
		if at+32 > len(block) {
			return dv, ErrInvalidParamsData
		}
		n := int(new(big.Int).SetBytes(block[at : at+32]).Int64())
		if n < 0 || at+32+n > len(block) {
			return dv, ErrInvalidParamsData
		}
		raw := make([]byte, n)
		copy(raw, block[at+32:at+32+n])
		if t.kind == kindString {
			dv.Value = string(raw)
		} else {
			dv.Value = raw
		}
		return dv, nil
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./abi/ -run 'TestEncodeDecode_BytesAndString|TestEncodeValue_BytesLayout' -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add abi/codec.go abi/codec_test.go
git commit -m "feat(abi): encode/decode dynamic bytes and string"
```

---

### Task 5: Arrays (`T[]` and `T[N]`)

**Files:**
- Modify: `abi/codec.go`
- Test: `abi/codec_test.go` (append)

- [ ] **Step 1: Write the failing tests**

```go
func TestEncodeDecode_DynamicArray(t *testing.T) {
	st, err := parseType("uint256[]", nil)
	if err != nil {
		t.Fatal(err)
	}
	out, err := encodeParams([]abiType{st}, []any{[]any{big.NewInt(1), big.NewInt(2), big.NewInt(3)}})
	if err != nil {
		t.Fatal(err)
	}
	vals, err := decodeParams([]abiType{st}, out)
	if err != nil {
		t.Fatal(err)
	}
	arr := vals[0].Value.([]DecodedValue)
	if len(arr) != 3 || arr[0].Value.(*big.Int).Int64() != 1 || arr[2].Value.(*big.Int).Int64() != 3 {
		t.Fatalf("array=%+v", arr)
	}
}

func TestEncodeDecode_FixedArray(t *testing.T) {
	ft, err := parseType("uint256[2]", nil)
	if err != nil {
		t.Fatal(err)
	}
	out, err := encodeParams([]abiType{ft}, []any{[]any{big.NewInt(7), big.NewInt(8)}})
	if err != nil {
		t.Fatal(err)
	}
	// fixed static array has no length prefix: exactly 64 bytes
	if len(out) != 64 {
		t.Fatalf("len=%d want 64", len(out))
	}
	vals, err := decodeParams([]abiType{ft}, out)
	if err != nil {
		t.Fatal(err)
	}
	arr := vals[0].Value.([]DecodedValue)
	if len(arr) != 2 || arr[1].Value.(*big.Int).Int64() != 8 {
		t.Fatalf("array=%+v", arr)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./abi/ -run 'TestEncodeDecode_DynamicArray|TestEncodeDecode_FixedArray' -v`
Expected: FAIL — `unsupported type "uint256[]"`.

- [ ] **Step 3: Add encode arms** to `encodeValue`

```go
	case kindArray:
		elems, ok := v.([]any)
		if !ok || len(elems) != t.size {
			return nil, fmt.Errorf("%w: array[%d] expected", ErrInvalidParamsData, t.size)
		}
		return encodeParams(repeatType(*t.elem, len(elems)), elems)
	case kindSlice:
		elems, ok := v.([]any)
		if !ok {
			return nil, fmt.Errorf("%w: slice expected", ErrInvalidParamsData)
		}
		body, err := encodeParams(repeatType(*t.elem, len(elems)), elems)
		if err != nil {
			return nil, err
		}
		return append(leftPad32(big.NewInt(int64(len(elems))).Bytes()), body...), nil
```

- [ ] **Step 4: Add decode arms** to `decodeValue`

```go
	case kindArray:
		vals, err := decodeParams(repeatType(*t.elem, t.size), block[at:])
		if err != nil {
			return dv, err
		}
		dv.Value = vals
		return dv, nil
	case kindSlice:
		if at+32 > len(block) {
			return dv, ErrInvalidParamsData
		}
		n := int(new(big.Int).SetBytes(block[at : at+32]).Int64())
		if n < 0 {
			return dv, ErrInvalidParamsData
		}
		vals, err := decodeParams(repeatType(*t.elem, n), block[at+32:])
		if err != nil {
			return dv, err
		}
		dv.Value = vals
		return dv, nil
```

- [ ] **Step 5: Add the `repeatType` helper** at the end of `codec.go`

```go
// repeatType returns a slice of n copies of t, used to treat array/tuple
// elements as a head/tail sequence.
func repeatType(t abiType, n int) []abiType {
	out := make([]abiType, n)
	for i := range out {
		out[i] = t
	}
	return out
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./abi/ -run 'TestEncodeDecode_DynamicArray|TestEncodeDecode_FixedArray' -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add abi/codec.go abi/codec_test.go
git commit -m "feat(abi): encode/decode fixed and dynamic arrays"
```

---

### Task 6: Tuples + the canonical `sam` vector

**Files:**
- Modify: `abi/codec.go`
- Test: `abi/codec_test.go` (append)

- [ ] **Step 1: Write the failing tests** (tuple round-trip + full `sam` vector)

```go
func TestEncodeDecode_Tuple(t *testing.T) {
	comps := []*SmartContractAbiEntryInput{
		{Name: "a", Type: "uint256"},
		{Name: "s", Type: "string"},
	}
	tt, err := parseType("tuple", comps)
	if err != nil {
		t.Fatal(err)
	}
	out, err := encodeParams([]abiType{tt}, []any{[]any{big.NewInt(42), "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	vals, err := decodeParams([]abiType{tt}, out)
	if err != nil {
		t.Fatal(err)
	}
	fields := vals[0].Value.([]DecodedValue)
	if fields[0].Value.(*big.Int).Int64() != 42 || fields[1].Value.(string) != "hi" {
		t.Fatalf("tuple=%+v", fields)
	}
	if fields[0].Name != "a" || fields[1].Name != "s" {
		t.Fatalf("tuple names not populated: %+v", fields)
	}
}

func TestEncodeParams_Sam_CanonicalVector(t *testing.T) {
	// sam(bytes,bool,uint256[]) with ("dave", true, [1,2,3])
	bts, _ := parseType("bytes", nil)
	bl, _ := parseType("bool", nil)
	arr, _ := parseType("uint256[]", nil)
	out, err := encodeParams([]abiType{bts, bl, arr},
		[]any{[]byte("dave"), true, []any{big.NewInt(1), big.NewInt(2), big.NewInt(3)}})
	if err != nil {
		t.Fatal(err)
	}
	want := hexToBytes(t,
		"0000000000000000000000000000000000000000000000000000000000000060"+
			"0000000000000000000000000000000000000000000000000000000000000001"+
			"00000000000000000000000000000000000000000000000000000000000000a0"+
			"0000000000000000000000000000000000000000000000000000000000000004"+
			"6461766500000000000000000000000000000000000000000000000000000000"+
			"0000000000000000000000000000000000000000000000000000000000000003"+
			"0000000000000000000000000000000000000000000000000000000000000001"+
			"0000000000000000000000000000000000000000000000000000000000000002"+
			"0000000000000000000000000000000000000000000000000000000000000003")
	if !bytes.Equal(out, want) {
		t.Fatalf("got\n%x\nwant\n%x", out, want)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./abi/ -run 'TestEncodeDecode_Tuple|TestEncodeParams_Sam' -v`
Expected: FAIL — tuple `unsupported type`.

- [ ] **Step 3: Add encode arm** to `encodeValue`

```go
	case kindTuple:
		elems, ok := v.([]any)
		if !ok || len(elems) != len(t.fields) {
			return nil, fmt.Errorf("%w: tuple of %d expected", ErrInvalidParamsData, len(t.fields))
		}
		return encodeParams(t.fields, elems)
```

- [ ] **Step 4: Add decode arm** to `decodeValue` (names applied from `t.names`)

```go
	case kindTuple:
		// Offsets inside a tuple body are relative to the tuple's own start,
		// so re-base at `at` for both static and dynamic tuples.
		vals, err := decodeParams(t.fields, block[at:])
		if err != nil {
			return dv, err
		}
		for i := range vals {
			if i < len(t.names) {
				vals[i].Name = t.names[i]
			}
		}
		dv.Value = vals
		return dv, nil
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./abi/ -run 'TestEncodeDecode_Tuple|TestEncodeParams_Sam' -v`
Expected: PASS.

- [ ] **Step 6: Run the whole codec suite**

Run: `go test ./abi/ -run 'TestEncode|TestDecode' -v`
Expected: PASS (all codec tests).

- [ ] **Step 7: Commit**

```bash
git add abi/codec.go abi/codec_test.go
git commit -m "feat(abi): encode/decode tuples; pass canonical sam vector"
```

---

### Task 7: Wire tuples into the ABI entry + canonical selector + typed decode

**Files:**
- Modify: `abi/smartcontractabi.go`
- Test: `abi/typed_entry_test.go` (create)

- [ ] **Step 1: Write the failing tests**

```go
package abi

import (
	"math/big"
	"testing"
)

func TestUpdateSignature_Erc20TransferUnchanged(t *testing.T) {
	// Regression: switching to canonical type strings must NOT change the
	// well-known ERC-20 transfer selector 0xa9059cbb.
	e := &SmartContractAbiEntry{
		Name: "transfer",
		Type: "Function",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "_to", Type: "address"},
			{Name: "_value", Type: "uint256"},
		},
	}
	sig := e.GetSignature()
	want := [4]byte{0xa9, 0x05, 0x9c, 0xbb}
	if sig != want {
		t.Fatalf("selector=%x want %x", sig, want)
	}
}

func TestUpdateSignature_TupleCanonical(t *testing.T) {
	// A method taking a tuple must hash the canonical "(uint256,address)"
	// form, not the literal "tuple".
	e := &SmartContractAbiEntry{
		Name: "order",
		Type: "Function",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "o", Type: "tuple", Components: []*SmartContractAbiEntryInput{
				{Name: "amt", Type: "uint256"},
				{Name: "maker", Type: "address"},
			}},
		},
	}
	got := e.canonicalSignature()
	if got != "order((uint256,address))" {
		t.Fatalf("canonicalSignature=%q", got)
	}
}

func TestDecodeInputsTyped_TransferWithTuple(t *testing.T) {
	e := &SmartContractAbiEntry{
		Name: "submit",
		Type: "Function",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "id", Type: "uint256"},
			{Name: "data", Type: "bytes"},
		},
	}
	sig := e.GetSignature()
	body, err := encodeParams(
		[]abiType{{kind: kindUint, size: 256}, {kind: kindBytes}},
		[]any{big.NewInt(5), []byte("hi")},
	)
	if err != nil {
		t.Fatal(err)
	}
	call := append(sig[:], body...)
	decoded, err := e.DecodeInputsTyped(call)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Method != "submit" || len(decoded.Inputs) != 2 {
		t.Fatalf("decoded=%+v", decoded)
	}
	if decoded.Inputs[0].Value.(*big.Int).Int64() != 5 {
		t.Fatalf("id=%v", decoded.Inputs[0].Value)
	}
	if string(decoded.Inputs[1].Value.([]byte)) != "hi" {
		t.Fatalf("data=%v", decoded.Inputs[1].Value)
	}
	if decoded.Inputs[0].Name != "id" {
		t.Fatalf("name not set: %+v", decoded.Inputs[0])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./abi/ -run 'TestUpdateSignature|TestDecodeInputsTyped' -v`
Expected: FAIL — `canonicalSignature`, `DecodeInputsTyped` undefined.
(The `Components` field already exists from Task 1, Step 0.)

- [ ] **Step 3: Switch `updateSignature` to canonical and add helpers** in `abi/smartcontractabi.go`

Replace the existing `updateSignature` method:

```go
func (e *SmartContractAbiEntry) updateSignature() {
	var params = make([]string, len(e.Inputs))
	for i, in := range e.Inputs {
		params[i] = in.Type
	}
	h := crypto.Keccak256([]byte(e.Name + "(" + strings.Join(params, ",") + ")"))
	copy(e.Signature[:], h[:4])
}
```

with:

```go
func (e *SmartContractAbiEntry) updateSignature() {
	h := crypto.Keccak256([]byte(e.canonicalSignature()))
	copy(e.Signature[:], h[:4])
}

// canonicalSignature returns "name(type1,type2,...)" using canonical type
// strings (tuples become "(...)"), matching how selectors/topics are hashed.
func (e *SmartContractAbiEntry) canonicalSignature() string {
	params := make([]string, len(e.Inputs))
	for i, in := range e.Inputs {
		typ, err := parseType(in.Type, in.Components)
		if err != nil {
			params[i] = in.Type // fall back to the raw string on parse failure
			continue
		}
		params[i] = typ.canonical()
	}
	return e.Name + "(" + strings.Join(params, ",") + ")"
}

// typedInputs parses every input into the typed model.
func (e *SmartContractAbiEntry) typedInputs() ([]abiType, error) {
	types := make([]abiType, len(e.Inputs))
	for i, in := range e.Inputs {
		typ, err := parseType(in.Type, in.Components)
		if err != nil {
			return nil, err
		}
		types[i] = typ
	}
	return types, nil
}

// DecodeInputsTyped decodes a method call (4-byte selector + ABI args) into a
// DecodedCall using the full typed engine. Unlike the legacy DecodeInputs, it
// supports dynamic types and tuples.
func (e *SmartContractAbiEntry) DecodeInputsTyped(data []byte) (*DecodedCall, error) {
	if len(data) < 4 {
		return nil, ErrInvalidParamsData
	}
	types, err := e.typedInputs()
	if err != nil {
		return nil, err
	}
	vals, err := decodeParams(types, data[4:])
	if err != nil {
		return nil, err
	}
	for i := range vals {
		if i < len(e.Inputs) {
			vals[i].Name = e.Inputs[i].Name
		}
	}
	return &DecodedCall{Method: e.Name, Inputs: vals}, nil
}

// EncodeInputsTyped encodes a method call (selector + ABI args) from typed values.
func (e *SmartContractAbiEntry) EncodeInputsTyped(values ...any) ([]byte, error) {
	types, err := e.typedInputs()
	if err != nil {
		return nil, err
	}
	body, err := encodeParams(types, values)
	if err != nil {
		return nil, err
	}
	sig := e.GetSignature()
	return append(sig[:], body...), nil
}
```

- [ ] **Step 4: Run the new tests to verify they pass**

Run: `go test ./abi/ -run 'TestUpdateSignature|TestDecodeInputsTyped' -v`
Expected: PASS.

- [ ] **Step 5: Run the FULL abi suite (regression — ERC-20 path must still pass)**

Run: `go test ./abi/ -count=1`
Expected: `ok` — all pre-existing tests (`abi_test.go`) plus the new ones pass.

- [ ] **Step 6: Commit**

```bash
git add abi/smartcontractabi.go abi/typed_entry_test.go
git commit -m "feat(abi): tuple components, canonical selectors, typed call decode"
```

---

### Task 8: Final verification and docs

**Files:**
- Modify: `todo/TASKS.md` (check off M1.1–M1.8)

- [ ] **Step 1: Run the entire abi package with the race detector**

Run: `go test ./abi/ -race -count=1`
Expected: `ok` — no failures, no data races.

- [ ] **Step 2: Confirm nothing else broke**

Run: `go build ./... && go vet ./abi/`
Expected: both exit 0.

- [ ] **Step 3: Mark M1 tasks done in `todo/TASKS.md`**

Change each of `M1.1`–`M1.8` from `- [ ]` to `- [x]` in
`todo/TASKS.md` (the "Milestone 1" section), since all are now implemented:
- M1.1 head/tail offsets → `encodeParams`/`decodeParams`
- M1.2 uintN/intN → `parseIntType`, `encodeInt`/`decodeInt`
- M1.3 bytesN → `kindFixedBytes`
- M1.4 bytes/string → Task 4
- M1.5 arrays → Task 5
- M1.6 tuples → Task 6/7
- M1.7 decoded model → `value.go`
- M1.8 canonical-vector tests → `codec_test.go` (`baz`, `sam`)

- [ ] **Step 4: Commit**

```bash
git add todo/TASKS.md
git commit -m "docs(abi): mark M1 (ABI dynamic types & tuples) complete"
```

---

## Notes for the implementer

- **Do not modify** `_parseParam`, `paramInput`, `DecodeInputs`, `encodeInputs`,
  or `erc20.go`. The typed engine is additive; the ERC-20 path stays as is.
- The decoded `Value` types are exactly: `*big.Int` (uint/int), `bool`,
  `[]byte` (address — 20 bytes; bytesN — N bytes; bytes — variable), `string`,
  and `[]DecodedValue` (array, slice, tuple). Tests rely on these concrete types.
- Offsets are always relative to the start of the current sequence/sub-block;
  `decodeValue` re-bases via `block[at:]` for dynamic array/slice/tuple bodies.
- `parseType` is recursive and shares `components` down through array suffixes,
  so `tuple[]`/`tuple[N]` resolve their element components correctly.
```
