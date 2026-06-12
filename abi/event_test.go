package abi

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"
)

func mustHex32(t *testing.T, s string) [32]byte {
	t.Helper()
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil || len(b) != 32 {
		t.Fatalf("bad 32-byte hex %q: %v", s, err)
	}
	var out [32]byte
	copy(out[:], b)
	return out
}

func TestTopic0_Erc20Transfer(t *testing.T) {
	// Transfer(address,address,uint256) — the canonical ERC-20 Transfer topic0.
	e := &SmartContractAbiEntry{
		Name: "Transfer",
		Type: "Event",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "from", Type: "address", Indexed: true},
			{Name: "to", Type: "address", Indexed: true},
			{Name: "value", Type: "uint256"},
		},
	}
	got := e.Topic0()
	want := mustHex32(t, "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	if got != want {
		t.Fatalf("topic0=%x want %x", got, want)
	}
}

func TestTopic0_Erc1155TransferSingle(t *testing.T) {
	// TransferSingle(address,address,address,uint256,uint256)
	e := &SmartContractAbiEntry{
		Name: "TransferSingle",
		Type: "Event",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "operator", Type: "address", Indexed: true},
			{Name: "from", Type: "address", Indexed: true},
			{Name: "to", Type: "address", Indexed: true},
			{Name: "id", Type: "uint256"},
			{Name: "value", Type: "uint256"},
		},
	}
	got := e.Topic0()
	want := mustHex32(t, "c3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62")
	if got != want {
		t.Fatalf("topic0=%x want %x", got, want)
	}
}

func TestEntryDecodeLog_Erc20Transfer(t *testing.T) {
	e := &SmartContractAbiEntry{
		Name: "Transfer",
		Type: "Event",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "from", Type: "address", Indexed: true},
			{Name: "to", Type: "address", Indexed: true},
			{Name: "value", Type: "uint256"},
		},
	}
	from := mustHex32(t, "000000000000000000000000"+"1111111111111111111111111111111111111111")
	to := mustHex32(t, "000000000000000000000000"+"2222222222222222222222222222222222222222")
	topics := [][32]byte{e.Topic0(), from, to}
	// data = the non-indexed uint256 value = 1000
	data := make([]byte, 32)
	big.NewInt(1000).FillBytes(data)

	ev, err := e.DecodeLog(topics, data)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Name != "Transfer" || len(ev.Inputs) != 3 {
		t.Fatalf("event=%+v", ev)
	}
	// indexed addresses decoded to 20-byte []byte
	wantFrom, _ := hex.DecodeString("1111111111111111111111111111111111111111")
	if !bytes.Equal(ev.Inputs[0].Value.([]byte), wantFrom) {
		t.Fatalf("from=%x", ev.Inputs[0].Value)
	}
	if ev.Inputs[0].Name != "from" {
		t.Fatalf("name=%q", ev.Inputs[0].Name)
	}
	// non-indexed value decoded from data
	if ev.Inputs[2].Value.(*big.Int).Int64() != 1000 {
		t.Fatalf("value=%v", ev.Inputs[2].Value)
	}
}

func TestEntryDecodeLog_RejectsNonEvent(t *testing.T) {
	e := &SmartContractAbiEntry{Name: "transfer", Type: "Function"}
	if _, err := e.DecodeLog(nil, nil); err == nil {
		t.Fatal("non-event must be rejected")
	}
}

func TestEntryDecodeLog_RejectsTopicCountMismatch(t *testing.T) {
	e := &SmartContractAbiEntry{
		Name: "Transfer",
		Type: "Event",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "from", Type: "address", Indexed: true},
			{Name: "to", Type: "address", Indexed: true},
			{Name: "value", Type: "uint256"},
		},
	}
	// only 1 topic past topic0, but 2 indexed params expected
	topics := [][32]byte{e.Topic0(), mustHex32(t, strings.Repeat("00", 32))}
	if _, err := e.DecodeLog(topics, nil); err == nil {
		t.Fatal("topic count mismatch must be rejected")
	}
}

func TestEntryDecodeLog_IndexedStaticArrayIsHashed(t *testing.T) {
	// An indexed uint256[2] is a STATIC type but, per the Solidity spec, all
	// indexed arrays are stored as keccak256(value) — so decoding must yield a
	// 32-byte hash placeholder, never an error or a wrongly-decoded value.
	e := &SmartContractAbiEntry{
		Name: "Arr",
		Type: "Event",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "pair", Type: "uint256[2]", Indexed: true},
			{Name: "n", Type: "uint256"},
		},
	}
	pairHash := mustHex32(t, "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	topics := [][32]byte{e.Topic0(), pairHash}
	data := make([]byte, 32)
	big.NewInt(9).FillBytes(data)

	ev, err := e.DecodeLog(topics, data)
	if err != nil {
		t.Fatalf("indexed static array must decode to a hash placeholder, got err: %v", err)
	}
	hash, ok := ev.Inputs[0].Value.([]byte)
	if !ok || len(hash) != 32 || !bytes.Equal(hash, pairHash[:]) {
		t.Fatalf("pair must be 32-byte hash placeholder, got %T %v", ev.Inputs[0].Value, ev.Inputs[0].Value)
	}
	if !strings.Contains(ev.Inputs[0].Type, "indexed") {
		t.Fatalf("type should mark indexed: %q", ev.Inputs[0].Type)
	}
	if ev.Inputs[1].Value.(*big.Int).Int64() != 9 {
		t.Fatalf("n=%v", ev.Inputs[1].Value)
	}
}

func TestEntryDecodeLog_IndexedDynamicPlaceholder(t *testing.T) {
	// An indexed string is stored as keccak256(value); decoding yields the
	// 32-byte hash placeholder, not the original string.
	e := &SmartContractAbiEntry{
		Name: "Named",
		Type: "Event",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "key", Type: "string", Indexed: true},
			{Name: "n", Type: "uint256"},
		},
	}
	keyHash := mustHex32(t, "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899")
	topics := [][32]byte{e.Topic0(), keyHash}
	data := make([]byte, 32)
	big.NewInt(7).FillBytes(data)

	ev, err := e.DecodeLog(topics, data)
	if err != nil {
		t.Fatal(err)
	}
	hash, ok := ev.Inputs[0].Value.([]byte)
	if !ok || len(hash) != 32 {
		t.Fatalf("indexed string must be a 32-byte hash placeholder, got %T", ev.Inputs[0].Value)
	}
	if !bytes.Equal(hash, keyHash[:]) {
		t.Fatalf("placeholder=%x want %x", hash, keyHash[:])
	}
	if !strings.Contains(ev.Inputs[0].Type, "indexed") {
		t.Fatalf("type should mark indexed: %q", ev.Inputs[0].Type)
	}
	if ev.Inputs[1].Value.(*big.Int).Int64() != 7 {
		t.Fatalf("n=%v", ev.Inputs[1].Value)
	}
}

func TestEntryDecodeLog_Erc1155TransferBatch(t *testing.T) {
	// TransferBatch(address operator, address from, address to,
	//               uint256[] ids, uint256[] values)
	// operator/from/to indexed; ids+values are non-indexed dynamic arrays in data.
	e := &SmartContractAbiEntry{
		Name: "TransferBatch",
		Type: "Event",
		Inputs: []*SmartContractAbiEntryInput{
			{Name: "operator", Type: "address", Indexed: true},
			{Name: "from", Type: "address", Indexed: true},
			{Name: "to", Type: "address", Indexed: true},
			{Name: "ids", Type: "uint256[]"},
			{Name: "values", Type: "uint256[]"},
		},
	}
	op := mustHex32(t, "000000000000000000000000"+"1111111111111111111111111111111111111111")
	from := mustHex32(t, "000000000000000000000000"+"2222222222222222222222222222222222222222")
	to := mustHex32(t, "000000000000000000000000"+"3333333333333333333333333333333333333333")
	topics := [][32]byte{e.Topic0(), op, from, to}

	// Build data = abi.encode(uint256[]{10,11}, uint256[]{20,21}) using the
	// engine itself (round-trip is the check).
	idsType, _ := parseType("uint256[]", nil)
	valsType, _ := parseType("uint256[]", nil)
	data, err := encodeParams([]abiType{idsType, valsType}, []any{
		[]any{big.NewInt(10), big.NewInt(11)},
		[]any{big.NewInt(20), big.NewInt(21)},
	})
	if err != nil {
		t.Fatal(err)
	}

	ev, err := e.DecodeLog(topics, data)
	if err != nil {
		t.Fatal(err)
	}
	if len(ev.Inputs) != 5 {
		t.Fatalf("inputs=%d", len(ev.Inputs))
	}
	ids := ev.Inputs[3].Value.([]DecodedValue)
	vals := ev.Inputs[4].Value.([]DecodedValue)
	if len(ids) != 2 || ids[0].Value.(*big.Int).Int64() != 10 || ids[1].Value.(*big.Int).Int64() != 11 {
		t.Fatalf("ids=%+v", ids)
	}
	if len(vals) != 2 || vals[0].Value.(*big.Int).Int64() != 20 || vals[1].Value.(*big.Int).Int64() != 21 {
		t.Fatalf("values=%+v", vals)
	}
	// indexed 'to' decoded correctly
	wantTo, _ := hex.DecodeString("3333333333333333333333333333333333333333")
	if !bytes.Equal(ev.Inputs[2].Value.([]byte), wantTo) {
		t.Fatalf("to=%x", ev.Inputs[2].Value)
	}
}
