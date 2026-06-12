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

func TestEncodeValue_RejectsNegativeUint(t *testing.T) {
	_, err := encodeParams([]abiType{{kind: kindUint, size: 256}}, []any{big.NewInt(-1)})
	if err == nil {
		t.Fatal("negative uint must be rejected")
	}
}

func TestEncodeValue_RejectsBadAddressLength(t *testing.T) {
	for _, n := range []int{19, 21, 32} {
		if _, err := encodeParams([]abiType{{kind: kindAddress}}, []any{make([]byte, n)}); err == nil {
			t.Fatalf("address of %d bytes must be rejected (want exactly 20)", n)
		}
	}
}

func TestDecodeParams_RejectsHugeOffset(t *testing.T) {
	// A dynamic param whose 32-byte offset slot is enormous must be rejected,
	// not truncated by Int64() into a valid-looking index.
	block := make([]byte, 32)
	for i := range block {
		block[i] = 0xff
	}
	if _, err := decodeParams([]abiType{{kind: kindBytes}}, block); err == nil {
		t.Fatal("huge offset must be rejected")
	}
}

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

func TestDecodeValue_RejectsHugeBytesLength(t *testing.T) {
	// A bytes value whose 32-byte length slot is enormous must be rejected,
	// not truncated by Int64() into a value that overflows the bounds check.
	block := make([]byte, 64)
	for i := 0; i < 32; i++ {
		block[i] = 0xff // length slot = 2^256-1
	}
	if _, err := decodeValue(abiType{kind: kindBytes}, block, 0); err == nil {
		t.Fatal("huge bytes length must be rejected")
	}
	// Also a length that fits int64 but exceeds the available data must fail.
	block2 := make([]byte, 64)
	block2[31] = 0xff // length = 255, but only 32 data bytes follow
	if _, err := decodeValue(abiType{kind: kindBytes}, block2, 0); err == nil {
		t.Fatal("length exceeding available data must be rejected")
	}
	// Low 64 bits = 0x7fffffffffffffff (huge positive), high bits set too.
	block3 := make([]byte, 64)
	for i := 0; i < 24; i++ {
		block3[i] = 0xff
	}
	block3[24] = 0x7f
	for i := 25; i < 32; i++ {
		block3[i] = 0xff
	}
	if _, err := decodeValue(abiType{kind: kindBytes}, block3, 0); err == nil {
		t.Fatal("huge positive bytes length must be rejected")
	}
}

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

func TestDecodeParams_RejectsHugeSliceLength(t *testing.T) {
	// A slice whose 32-byte element-count slot is enormous must be rejected,
	// not truncated by Int64() into a value that overflows a bounds check.
	st, err := parseType("uint256[]", nil)
	if err != nil {
		t.Fatal(err)
	}
	// head: offset 0x20; tail: a 32-byte all-0xff length, then nothing.
	block := make([]byte, 96)
	block[31] = 0x20 // offset points at byte 32
	for i := 32; i < 64; i++ {
		block[i] = 0xff // element count = 2^256-1
	}
	if _, err := decodeParams([]abiType{st}, block); err == nil {
		t.Fatal("huge slice length must be rejected")
	}
}

func TestEncodeDecode_ArrayOfDynamicElement(t *testing.T) {
	// string[] exercises nested head/tail: the outer slice has a length +
	// element offsets, and each string is itself dynamic.
	st, err := parseType("string[]", nil)
	if err != nil {
		t.Fatal(err)
	}
	out, err := encodeParams([]abiType{st}, []any{[]any{"foo", "bar"}})
	if err != nil {
		t.Fatal(err)
	}
	vals, err := decodeParams([]abiType{st}, out)
	if err != nil {
		t.Fatal(err)
	}
	arr := vals[0].Value.([]DecodedValue)
	if len(arr) != 2 || arr[0].Value.(string) != "foo" || arr[1].Value.(string) != "bar" {
		t.Fatalf("array=%+v", arr)
	}
}

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
