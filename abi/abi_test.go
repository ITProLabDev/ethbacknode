package abi

import (
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"sync"
	"testing"
)

// --- test doubles ---------------------------------------------------------

type memStorage struct {
	mu     sync.Mutex
	data   []byte
	exists bool
}

func (s *memStorage) IsExists() bool { return s.exists }

func (s *memStorage) Save(raw []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = append(s.data[:0], raw...)
	s.exists = true
	return nil
}

func (s *memStorage) Load() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.exists {
		return nil, errors.New("not found")
	}
	out := make([]byte, len(s.data))
	copy(out, s.data)
	return out, nil
}

// hexCodec encodes/decodes between 0x-prefixed lowercase hex strings and
// 20-byte slices. Sufficient for exercising the abi package in isolation.
type hexCodec struct {
	decodeErr error
	encodeErr error
}

func (h *hexCodec) EncodeBytesToAddress(b []byte) (string, error) {
	if h.encodeErr != nil {
		return "", h.encodeErr
	}
	if len(b) != 20 {
		return "", errors.New("invalid bytes")
	}
	return "0x" + hex.EncodeToString(b), nil
}

func (h *hexCodec) DecodeAddressToBytes(s string) ([]byte, error) {
	if h.decodeErr != nil {
		return nil, h.decodeErr
	}
	s = strings.TrimPrefix(strings.ToLower(s), "0x")
	if len(s) != 40 {
		return nil, errors.New("invalid address length")
	}
	return hex.DecodeString(s)
}

func (h *hexCodec) PrivateKeyToAddress([]byte) (string, []byte, error) {
	return "", nil, errors.New("not implemented")
}

func (h *hexCodec) IsValid(s string) bool {
	_, err := h.DecodeAddressToBytes(s)
	return err == nil
}

func newTestManager(t *testing.T) *SmartContractsManager {
	t.Helper()
	m := NewManager(
		WithStorage(&memStorage{}),
		WithAddressCodec(&hexCodec{}),
	)
	if err := m.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	return m
}

// --- helpers --------------------------------------------------------------

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}

// transferCallData builds canonical ABI calldata for transfer(addr, amount).
func transferCallData(t *testing.T, addr20 []byte, amount *big.Int) []byte {
	t.Helper()
	if len(addr20) != 20 {
		t.Fatalf("addr must be 20 bytes, got %d", len(addr20))
	}
	out := mustHex(t, "a9059cbb") // selector for transfer(address,uint256)
	out = append(out, bytePad(addr20, 32, 0)...)
	out = append(out, bytePad(amount.Bytes(), 32, 0)...)
	return out
}

// --- helpers tests --------------------------------------------------------

func TestBytePad_LeftPads(t *testing.T) {
	got := bytePad([]byte{0x01, 0x02}, 4, 0)
	want := []byte{0, 0, 0x01, 0x02}
	if !equalBytes(got, want) {
		t.Fatalf("got %x want %x", got, want)
	}
}

func TestBytePad_OversizedSrcDoesNotPanic(t *testing.T) {
	src := []byte{1, 2, 3, 4, 5}
	got := bytePad(src, 3, 0)
	// Implementation choice: keep the trailing N bytes (big-endian numeric semantics).
	want := []byte{3, 4, 5}
	if !equalBytes(got, want) {
		t.Fatalf("got %x want %x", got, want)
	}
}

func TestBytePad_NonZeroPad(t *testing.T) {
	got := bytePad([]byte{0xab}, 4, 0xff)
	want := []byte{0xff, 0xff, 0xff, 0xab}
	if !equalBytes(got, want) {
		t.Fatalf("got %x want %x", got, want)
	}
}

// --- _extractMethodId / _parseParam ---------------------------------------

func TestExtractMethodId_ShortInputNoPanic(t *testing.T) {
	for _, n := range []int{0, 1, 2, 3} {
		_, ok := _extractMethodId(make([]byte, n))
		if ok {
			t.Fatalf("len=%d should be rejected", n)
		}
	}
	sig, ok := _extractMethodId([]byte{1, 2, 3, 4, 5})
	if !ok || sig != [4]byte{1, 2, 3, 4} {
		t.Fatalf("unexpected: sig=%v ok=%v", sig, ok)
	}
}

func TestParseParam_ShortDataReturnsError(t *testing.T) {
	cases := []string{"uint256", "int256", "address", "bool"}
	for _, typ := range cases {
		var p paramInput
		_, err := _parseParam(&p, typ, make([]byte, 8))
		if err == nil {
			t.Fatalf("type=%s short data should error", typ)
		}
	}
}

func TestParseParam_UnknownType(t *testing.T) {
	var p paramInput
	_, err := _parseParam(&p, "string", make([]byte, 32))
	if err == nil {
		t.Fatal("unknown type must error")
	}
}

// --- paramInput SetBool / GetBool ----------------------------------------

func TestParamInput_BoolRoundTrip_NoPanicOnZeroValue(t *testing.T) {
	var p paramInput
	// Pre-fix this would panic: p.Data was nil and SetBool(true) wrote p.Data[0].
	p.SetBool(true)
	if len(p.Data) != 32 {
		t.Fatalf("SetBool must produce 32-byte slot, got %d", len(p.Data))
	}
	if !p.GetBool() {
		t.Fatal("expected true after SetBool(true)")
	}
	p2 := paramInput{}
	p2.SetBool(false)
	if p2.GetBool() {
		t.Fatal("expected false after SetBool(false)")
	}
	// Empty data must not panic.
	empty := paramInput{}
	if empty.GetBool() {
		t.Fatal("empty data must read as false")
	}
}

func TestParamInput_SetAddressPadsTo32(t *testing.T) {
	var p paramInput
	addr := mustHex(t, "0102030405060708090a0b0c0d0e0f1011121314")
	if err := p.SetAddress(addr); err != nil {
		t.Fatal(err)
	}
	if len(p.Data) != 32 {
		t.Fatalf("padded len=%d want 32", len(p.Data))
	}
	if !equalBytes(p.Data[12:], addr) {
		t.Fatalf("address must be right-aligned, got %x", p.Data)
	}
	for i := 0; i < 12; i++ {
		if p.Data[i] != 0 {
			t.Fatalf("left pad must be zero at %d, got %x", i, p.Data[i])
		}
	}
}

// --- DecodeInputs ---------------------------------------------------------

func TestDecodeInputs_ShortDataNoPanic(t *testing.T) {
	m := newTestManager(t)
	method, err := m.erc20abi.GetMethodByName("transfer")
	if err != nil {
		t.Fatalf("GetMethodByName: %v", err)
	}
	cases := [][]byte{
		nil,
		{},
		{1},
		{1, 2, 3},
		{1, 2, 3, 4},          // selector only, no params
		{1, 2, 3, 4, 5, 6, 7}, // selector + truncated params
	}
	for _, c := range cases {
		_, err := method.DecodeInputs(c) // must return an error, not panic
		if err == nil {
			t.Fatalf("len=%d should error", len(c))
		}
	}
}

// --- ERC-20 high level ---------------------------------------------------

func TestErc20IsTransfer_ShortDataNoPanic(t *testing.T) {
	m := newTestManager(t)
	for _, n := range []int{0, 1, 2, 3} {
		if m.Erc20IsTransfer(make([]byte, n)) {
			t.Fatalf("len=%d should not be a transfer", n)
		}
	}
}

func TestErc20IsTransfer_NilAbi(t *testing.T) {
	// Manager constructed but Init never called → erc20abi is nil.
	m := NewManager(WithStorage(&memStorage{}), WithAddressCodec(&hexCodec{}))
	if m.Erc20IsTransfer([]byte{1, 2, 3, 4, 5}) {
		t.Fatal("nil abi must return false (no panic)")
	}
}

func TestErc20DecodeIfTransfer_HappyPath(t *testing.T) {
	m := newTestManager(t)
	addr := mustHex(t, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	amount := big.NewInt(123456789)
	data := transferCallData(t, addr, amount)

	gotAddr, gotAmount, err := m.Erc20DecodeIfTransfer(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.EqualFold(gotAddr, "0x"+hex.EncodeToString(addr)) {
		t.Fatalf("addr mismatch: got %s", gotAddr)
	}
	if gotAmount.Cmp(amount) != 0 {
		t.Fatalf("amount mismatch: got %s want %s", gotAmount, amount)
	}
}

func TestErc20DecodeIfTransfer_NotATransfer(t *testing.T) {
	m := newTestManager(t)
	// Selector 0xdeadbeef, plus filler.
	data := append(mustHex(t, "deadbeef"), make([]byte, 64)...)
	_, _, err := m.Erc20DecodeIfTransfer(data)
	if !errors.Is(err, ErrNotTransferMethod) {
		t.Fatalf("want ErrNotTransferMethod, got %v", err)
	}
}

func TestErc20DecodeIfTransfer_ShortData(t *testing.T) {
	m := newTestManager(t)
	for _, n := range []int{0, 3, 10, 67} {
		_, _, err := m.Erc20DecodeIfTransfer(make([]byte, n))
		if err == nil {
			t.Fatalf("len=%d must error", n)
		}
	}
}

func TestErc20DecodeIfTransfer_NilAbi(t *testing.T) {
	m := NewManager(WithStorage(&memStorage{}), WithAddressCodec(&hexCodec{}))
	_, _, err := m.Erc20DecodeIfTransfer(make([]byte, 68))
	if err == nil {
		t.Fatal("nil abi must return error, not panic")
	}
}

func TestErc20DecodeIfTransfer_AddressEncodeError(t *testing.T) {
	codec := &hexCodec{encodeErr: errors.New("boom")}
	m := NewManager(WithStorage(&memStorage{}), WithAddressCodec(codec))
	if err := m.Init(); err != nil {
		t.Fatal(err)
	}
	addr := mustHex(t, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	data := transferCallData(t, addr, big.NewInt(1))
	_, _, err := m.Erc20DecodeIfTransfer(data)
	if err == nil {
		t.Fatal("expected error from codec")
	}
}

func TestErc20CallGetBalance_BuildsCorrectSelector(t *testing.T) {
	m := newTestManager(t)
	out, err := m.Erc20CallGetBalance("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	if err != nil {
		t.Fatal(err)
	}
	// 0x + 4-byte selector + 32-byte address slot = 0x + 8 + 64 = 74 chars
	if len(out) != 74 {
		t.Fatalf("len=%d want 74", len(out))
	}
	// balanceOf(address) selector = 0x70a08231
	if !strings.HasPrefix(strings.ToLower(out), "0x70a08231") {
		t.Fatalf("wrong selector: %s", out)
	}
	// trailing 20 bytes must equal the address
	addrPart := out[len(out)-40:]
	if addrPart != "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef" {
		t.Fatalf("encoded address wrong: %s", addrPart)
	}
}

func TestErc20CallGetBalance_NilAbi(t *testing.T) {
	m := NewManager(WithStorage(&memStorage{}), WithAddressCodec(&hexCodec{}))
	if _, err := m.Erc20CallGetBalance("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"); err == nil {
		t.Fatal("expected error when erc20abi is nil")
	}
}

func TestErc20DecodeAmount(t *testing.T) {
	m := newTestManager(t)
	// 32-byte big-endian representation of 0x1234.
	data := bytePad(big.NewInt(0x1234).Bytes(), 32, 0)
	got := m.Erc20DecodeAmount(data)
	if got.Cmp(big.NewInt(0x1234)) != 0 {
		t.Fatalf("amount mismatch: %s", got)
	}
}

// --- manager + storage ---------------------------------------------------

func TestManager_AddAndLookup(t *testing.T) {
	m := newTestManager(t)
	c := &SmartContractInfo{
		Name:            "TestToken",
		Symbol:          "TST",
		ContractAddress: "0xAaAaaaAAaaAaaAAAAAAAAAAAAAAAAAAAAaaaAAAa",
		Decimals:        18,
	}
	m.Add(c)

	got, err := m.GetSmartContractByAddress(strings.ToLower(c.ContractAddress))
	if err != nil {
		t.Fatalf("by address: %v", err)
	}
	if got.Name != c.Name {
		t.Fatalf("got %s", got.Name)
	}

	// case-insensitive address lookup
	got2, err := m.GetSmartContractByAddress(c.ContractAddress)
	if err != nil || got2.Name != c.Name {
		t.Fatalf("upper-case lookup failed: %v", err)
	}

	// symbol lookup must use the original case (regression: prior version
	// lowercased the input but stored mixed case, so it never matched).
	gotSym, err := m.GetSmartContractByToken("TST")
	if err != nil || gotSym.Name != c.Name {
		t.Fatalf("by symbol: %v %v", gotSym, err)
	}
	if _, err := m.GetSmartContractByToken("tst"); err == nil {
		t.Fatal("symbol lookup must be case-sensitive")
	}

	// name lookup
	addr, err := m.GetSmartContractAddressByName(c.Name)
	if err != nil || addr != c.ContractAddress {
		t.Fatalf("by name: %s %v", addr, err)
	}

	if _, err := m.GetSmartContractAddressByName("nope"); !errors.Is(err, ErrUnknownContract) {
		t.Fatalf("missing must return ErrUnknownContract, got %v", err)
	}
}

func TestManager_LoadResetsStaleEntries(t *testing.T) {
	// Regression: Load used to call afterLoad without clearing maps. After
	// rewriting storage with a different contract set, stale entries from
	// the prior load remained in byName/byAddress.
	storage := &memStorage{}
	m := NewManager(WithStorage(storage), WithAddressCodec(&hexCodec{}))
	if err := m.Init(); err != nil {
		t.Fatal(err)
	}

	a := &SmartContractInfo{Name: "A", Symbol: "AA", ContractAddress: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	m.Add(a)

	// Overwrite storage with a single different contract, then reload.
	storage.Save([]byte(`[{"name":"B","symbol":"BB","contract_address":"0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","decimals":0}]`))
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if _, err := m.GetSmartContractByAddress(a.ContractAddress); err == nil {
		t.Fatal("stale contract A must be gone after reload")
	}
	got, err := m.GetSmartContractByAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	if err != nil || got.Name != "B" {
		t.Fatalf("contract B should be present: %v %v", got, err)
	}
}

func TestManager_ColdStartLoadsTemplate(t *testing.T) {
	m := newTestManager(t)
	list := m.GetSmartContractList()
	if len(list) == 0 {
		t.Fatal("cold start should load at least one template contract")
	}
}

func TestManager_NilStorageInit(t *testing.T) {
	m := &SmartContractsManager{
		bySymbol:  make(map[string]*SmartContractInfo),
		byName:    make(map[string]*SmartContractInfo),
		byAddress: make(map[string]*SmartContractInfo),
	}
	if err := m.Init(); !errors.Is(err, ErrConfigStorageEmpty) {
		t.Fatalf("want ErrConfigStorageEmpty, got %v", err)
	}
}

// Concurrent reads across the lookup paths must not race. Run with -race.
func TestManager_ConcurrentLookupsNoRace(t *testing.T) {
	m := newTestManager(t)
	c := &SmartContractInfo{
		Name:            "Race",
		Symbol:          "RC",
		ContractAddress: "0x1111111111111111111111111111111111111111",
	}
	m.Add(c)

	var wg sync.WaitGroup
	stop := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_, _ = m.GetSmartContractByAddress(c.ContractAddress)
					_, _ = m.GetSmartContractByToken("RC")
					_ = m.GetSmartContractList()
					_ = m.Erc20IsTransfer(make([]byte, 4))
				}
			}
		}()
	}
	// Concurrent writers triggering Save → re-marshal; exercises mux.
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				select {
				case <-stop:
					return
				default:
					m.Add(&SmartContractInfo{
						Name:            "X" + string(rune('A'+i)) + string(rune('A'+j%26)),
						Symbol:          "S" + string(rune('A'+j%26)),
						ContractAddress: "0x" + strings.Repeat(string(rune('a'+i)), 40),
					})
				}
			}
		}(i)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		// brief stress window
		for i := 0; i < 1000; i++ {
			_ = m.GetSmartContractList()
		}
		close(stop)
	}()
	wg.Wait()
}

// --- internal helpers ----------------------------------------------------

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
