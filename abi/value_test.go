package abi

import (
	"encoding/json"
	"math/big"
	"testing"
)

// DecodedValue must serialize to a client-friendly JSON shape:
//   - byte types (address, bytesN, bytes) -> 0x-prefixed lowercase hex string
//   - big integers (uint*/int*)           -> decimal STRING (JS-safe, no precision loss)
//   - bool/string                          -> native JSON
//   - arrays/tuples ([]DecodedValue)       -> recurse
// The in-memory Value type is unchanged; only JSON output is transformed.

func TestDecodedValueJSON_AddressAsHex(t *testing.T) {
	addr := make([]byte, 20)
	addr[19] = 0x01
	dv := DecodedValue{Name: "from", Type: "address", Value: addr}
	b, err := json.Marshal(dv)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	want := `{"name":"from","type":"address","value":"0x0000000000000000000000000000000000000001"}`
	if got != want {
		t.Fatalf("address json:\n got %s\nwant %s", got, want)
	}
}

func TestDecodedValueJSON_Uint256AsDecimalString(t *testing.T) {
	// 1e18 exceeds JS Number.MAX_SAFE_INTEGER; must be a quoted decimal string.
	val, _ := new(big.Int).SetString("1000000000000000000", 10)
	dv := DecodedValue{Name: "value", Type: "uint256", Value: val}
	b, err := json.Marshal(dv)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	want := `{"name":"value","type":"uint256","value":"1000000000000000000"}`
	if got != want {
		t.Fatalf("uint256 json:\n got %s\nwant %s", got, want)
	}
}

func TestDecodedValueJSON_BytesAsHex(t *testing.T) {
	dv := DecodedValue{Type: "bytes", Value: []byte{0xde, 0xad, 0xbe, 0xef}}
	b, _ := json.Marshal(dv)
	want := `{"type":"bytes","value":"0xdeadbeef"}`
	if string(b) != want {
		t.Fatalf("bytes json:\n got %s\nwant %s", string(b), want)
	}
}

func TestDecodedValueJSON_EmptyBytesIsHexPrefix(t *testing.T) {
	dv := DecodedValue{Type: "bytes", Value: []byte{}}
	b, _ := json.Marshal(dv)
	want := `{"type":"bytes","value":"0x"}`
	if string(b) != want {
		t.Fatalf("empty bytes json:\n got %s\nwant %s", string(b), want)
	}
}

func TestDecodedValueJSON_BoolAndStringNative(t *testing.T) {
	bv, _ := json.Marshal(DecodedValue{Type: "bool", Value: true})
	if string(bv) != `{"type":"bool","value":true}` {
		t.Fatalf("bool json: %s", string(bv))
	}
	sv, _ := json.Marshal(DecodedValue{Type: "string", Value: "hi"})
	if string(sv) != `{"type":"string","value":"hi"}` {
		t.Fatalf("string json: %s", string(sv))
	}
}

func TestDecodedValueJSON_NestedArrayRecurses(t *testing.T) {
	dv := DecodedValue{
		Name: "ids",
		Type: "uint256[]",
		Value: []DecodedValue{
			{Type: "uint256", Value: big.NewInt(10)},
			{Type: "uint256", Value: big.NewInt(11)},
		},
	}
	b, _ := json.Marshal(dv)
	want := `{"name":"ids","type":"uint256[]","value":[{"type":"uint256","value":"10"},{"type":"uint256","value":"11"}]}`
	if string(b) != want {
		t.Fatalf("nested array json:\n got %s\nwant %s", string(b), want)
	}
}
