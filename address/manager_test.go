package address

import (
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/ITProLabDev/ethbacknode/storage"
)

// --- test doubles ---------------------------------------------------------

// memSimpleStorage is an in-memory implementation of storage.SimpleStorage.
type memSimpleStorage struct {
	mu   sync.Mutex
	data map[string][]byte // hex(key) -> encoded value
}

func newMemSimpleStorage() *memSimpleStorage {
	return &memSimpleStorage{data: make(map[string][]byte)}
}

func (s *memSimpleStorage) Save(d storage.Data) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[hex.EncodeToString(d.GetKey())] = d.Encode()
	return nil
}

func (s *memSimpleStorage) Read(k storage.Key, d storage.Data) error {
	s.mu.Lock()
	raw, ok := s.data[hex.EncodeToString(k.GetKey())]
	s.mu.Unlock()
	if !ok {
		return errors.New("not found")
	}
	return d.Decode(raw)
}

func (s *memSimpleStorage) ReadAll(processor func(raw []byte) error) error {
	s.mu.Lock()
	rows := make([][]byte, 0, len(s.data))
	for _, raw := range s.data {
		rows = append(rows, raw)
	}
	s.mu.Unlock()
	for _, raw := range rows {
		if err := processor(raw); err != nil {
			return err
		}
	}
	return nil
}

func (s *memSimpleStorage) Delete(rowKey []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, hex.EncodeToString(rowKey))
	return nil
}

func (s *memSimpleStorage) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.data)
}

// memBinStorage is an in-memory implementation of storage.BinStorage,
// used for the address-pool config.
type memBinStorage struct {
	mu     sync.Mutex
	data   []byte
	exists bool
}

func (s *memBinStorage) IsExists() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exists
}

func (s *memBinStorage) Save(raw []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = append(s.data[:0], raw...)
	s.exists = true
	return nil
}

func (s *memBinStorage) Load() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.exists {
		return nil, errors.New("not found")
	}
	out := make([]byte, len(s.data))
	copy(out, s.data)
	return out, nil
}

// --- helpers --------------------------------------------------------------

// noGenConfig pre-populates a config that disables auto-generation so the
// manager doesn't try to mint real BIP-44 keys during tests.
const noGenConfig = `{
  "debug": false,
  "enableAddressGenerate": false,
  "minFreePoolSize": 0,
  "generatePoolUpTo": 0,
  "bip39Support": false,
  "bip36MnemonicLen": 12,
  "bip44CoinType": "Ether",
  "bip32DerivationPath": "m/44'/60'/0'/0/0"
}`

func newTestManager(t *testing.T) (*Manager, *memSimpleStorage) {
	t.Helper()
	addrStore := newMemSimpleStorage()
	cfgStore := &memBinStorage{}
	if err := cfgStore.Save([]byte(noGenConfig)); err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(
		WithAddressStorage(addrStore),
		WithConfigStorage(cfgStore),
		WithAddressCodec(&MockAddressCodec{}),
	)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	return m, addrStore
}

// makeAddr constructs a fake but well-formed address for testing.
// addressBytes is derived deterministically from i so collisions are explicit.
func makeAddr(i int) *Address {
	b := make([]byte, 20)
	// fill with a recognisable, unique pattern
	for j := range b {
		b[j] = byte((i + j) & 0xff)
	}
	pk := make([]byte, 32)
	for j := range pk {
		pk[j] = byte((i*7 + j) & 0xff)
	}
	return &Address{
		Address:      "0x" + hex.EncodeToString(b),
		AddressBytes: b,
		PrivateKey:   pk,
	}
}

func walkCount(m *Manager) int {
	n := 0
	m.WalkAllAddresses(func(*Address) { n++ })
	return n
}

// findInPool looks up a record via the synchronously-maintained allAddresses
// map. We avoid GetAddress in tests because it reads from fastPool, which is
// rebuilt asynchronously and lags Add* by an unknown delay (audit #12).
func findInPool(m *Manager, addr string) *Address {
	var hit *Address
	m.WalkAllAddresses(func(a *Address) {
		if a.Address == addr {
			hit = a
		}
	})
	return hit
}

// --- tests ----------------------------------------------------------------

func TestAddAddressRecordsBulk_EmptyInput(t *testing.T) {
	m, _ := newTestManager(t)
	if err := m.AddAddressRecordsBulk(nil); err != nil {
		t.Fatalf("nil input: %v", err)
	}
	if err := m.AddAddressRecordsBulk([]*Address{}); err != nil {
		t.Fatalf("empty input: %v", err)
	}
	if walkCount(m) != 0 {
		t.Fatalf("pool must remain empty")
	}
}

func TestAddAddressRecordsBulk_AddsAllNew(t *testing.T) {
	m, store := newTestManager(t)
	batch := []*Address{makeAddr(1), makeAddr(2), makeAddr(3)}

	if err := m.AddAddressRecordsBulk(batch); err != nil {
		t.Fatalf("bulk add: %v", err)
	}
	if got := walkCount(m); got != 3 {
		t.Fatalf("walk count = %d, want 3", got)
	}
	if got := store.count(); got != 3 {
		t.Fatalf("store count = %d, want 3", got)
	}
}

// Regression for the data-loss bug: the old second-pass loop blindly wrote
// `allAddresses[a.Address] = a` for every input record, clobbering subscription
// state that was already populated for an existing address.
func TestAddAddressRecordsBulk_DoesNotOverwriteExistingInMemory(t *testing.T) {
	m, _ := newTestManager(t)

	existing := makeAddr(42)
	existing.ServiceId = 1234
	existing.UserId = 5678
	existing.InvoiceId = 9999
	existing.Subscribed = true
	if err := m.AddAddressRecord(existing); err != nil {
		t.Fatalf("seed AddAddressRecord: %v", err)
	}

	// Build a duplicate that, under the old bug, would overwrite the live
	// record with all zero subscription fields.
	dup := makeAddr(42)
	dup.ServiceId = 0
	dup.UserId = 0
	dup.InvoiceId = 0
	dup.Subscribed = false

	other := makeAddr(43)

	if err := m.AddAddressRecordsBulk([]*Address{dup, other}); err != nil {
		t.Fatalf("bulk: %v", err)
	}

	got := findInPool(m, existing.Address)
	if got == nil {
		t.Fatalf("existing record disappeared")
	}
	if got.ServiceId != 1234 || got.UserId != 5678 || got.InvoiceId != 9999 || !got.Subscribed {
		t.Fatalf("existing record was clobbered: %+v", got)
	}
	if walkCount(m) != 2 {
		t.Fatalf("walk count = %d, want 2 (existing + other)", walkCount(m))
	}
}

// Per user request: the bulk path must consult persistent storage too — a
// record present on disk but absent from RAM should not be re-added (which
// would call db.Save again and could clobber on-disk state if the in-memory
// copy were ever stale).
func TestAddAddressRecordsBulk_SkipsRecordPresentInDB(t *testing.T) {
	m, store := newTestManager(t)

	onDisk := makeAddr(7)
	onDisk.ServiceId = 111
	onDisk.UserId = 222
	onDisk.Subscribed = true
	// Inject directly into storage AFTER preLoad so it is in DB but not in RAM.
	if err := store.Save(onDisk); err != nil {
		t.Fatal(err)
	}

	// Bulk-add a "duplicate" with different (zero) fields.
	dup := makeAddr(7)
	if err := m.AddAddressRecordsBulk([]*Address{dup}); err != nil {
		t.Fatalf("bulk: %v", err)
	}

	// db.Save must not have been called again (still 1 row in storage),
	// and the on-disk record's fields must be intact.
	if got := store.count(); got != 1 {
		t.Fatalf("store count = %d, want 1 (db record preserved)", got)
	}
	var fromDisk Address
	if err := store.Read(onDisk, &fromDisk); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if fromDisk.ServiceId != 111 || fromDisk.UserId != 222 || !fromDisk.Subscribed {
		t.Fatalf("DB record was overwritten: %+v", fromDisk)
	}
}

func TestAddAddressRecordsBulk_SkipsNilAndEmpty(t *testing.T) {
	m, _ := newTestManager(t)
	valid := makeAddr(99)
	batch := []*Address{
		nil,
		{Address: "", AddressBytes: nil},
		valid,
	}
	if err := m.AddAddressRecordsBulk(batch); err != nil {
		t.Fatalf("bulk: %v", err)
	}
	if walkCount(m) != 1 {
		t.Fatalf("only the valid record should land in the pool")
	}
	if findInPool(m, valid.Address) == nil {
		t.Fatalf("valid record not found in pool")
	}
}

// Stress: many concurrent bulk-add calls with overlapping addresses must
// not race, deadlock, or produce duplicates. Run with -race.
func TestAddAddressRecordsBulk_ConcurrentNoRace(t *testing.T) {
	m, store := newTestManager(t)

	const goroutines = 8
	const perBatch = 10
	const overlapStride = 5 // batches share addresses for indices that overlap

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			batch := make([]*Address, perBatch)
			for j := 0; j < perBatch; j++ {
				batch[j] = makeAddr(g*overlapStride + j)
			}
			if err := m.AddAddressRecordsBulk(batch); err != nil {
				t.Errorf("g=%d: %v", g, err)
			}
		}(g)
	}
	wg.Wait()

	// Compute the exact number of unique indices that were submitted.
	unique := make(map[int]struct{})
	for g := 0; g < goroutines; g++ {
		for j := 0; j < perBatch; j++ {
			unique[g*overlapStride+j] = struct{}{}
		}
	}
	wantUnique := len(unique)

	if got := walkCount(m); got != wantUnique {
		t.Fatalf("walk count = %d, want %d (unique addresses)", got, wantUnique)
	}
	if got := store.count(); got != wantUnique {
		t.Fatalf("store count = %d, want %d", got, wantUnique)
	}
}

func TestAddAddressRecordsBulk_PartialFailureDoesNotCorruptPool(t *testing.T) {
	// First N records valid, then a record whose Save would fail. We don't
	// have a hook for forcing a Save error in memSimpleStorage, so we
	// instead assert no records survive that were never inserted, and
	// inserted ones remain consistent.
	m, store := newTestManager(t)
	good := []*Address{makeAddr(100), makeAddr(101)}
	if err := m.AddAddressRecordsBulk(good); err != nil {
		t.Fatalf("bulk: %v", err)
	}
	if walkCount(m) != len(good) || store.count() != len(good) {
		t.Fatalf("expected %d records, got walk=%d store=%d",
			len(good), walkCount(m), store.count())
	}

	// Re-running with the same batch must be a no-op and must not error.
	if err := m.AddAddressRecordsBulk(good); err != nil {
		t.Fatalf("idempotent bulk: %v", err)
	}
	if walkCount(m) != len(good) || store.count() != len(good) {
		t.Fatalf("idempotent bulk altered counts: walk=%d store=%d",
			walkCount(m), store.count())
	}
}

// Smoke check that fmt-formatted addresses from makeAddr are unique.
func TestMakeAddr_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		a := makeAddr(i)
		if seen[a.Address] {
			t.Fatalf("makeAddr collision at i=%d (%s)", i, a.Address)
		}
		seen[a.Address] = true
	}
	_ = fmt.Sprintf
}

// --- BIP-44 recovery regressions (#6, #7) --------------------------------

// Regression for #6: prior version returned (nil, nil) on bad mnemonics
// because RecoverBit44Address discarded the underlying error and always
// returned nil. A caller that checks `if err != nil` would treat invalid
// input as success and dereference a nil *Address.
func TestRecoverBit44Address_RejectsUnknownWord(t *testing.T) {
	m, _ := newTestManager(t)

	// "zzzz" is not in the BIP-39 wordlist → bip39.EntropyFromMnemonic
	// returns ErrInvalidMnemonic.
	bad := []string{
		"zzzz", "zzzz", "zzzz", "zzzz", "zzzz", "zzzz",
		"zzzz", "zzzz", "zzzz", "zzzz", "zzzz", "zzzz",
	}
	rec, err := m.RecoverBit44Address(bad)
	if err == nil {
		t.Fatal("invalid mnemonic must produce an error")
	}
	if rec != nil {
		t.Fatalf("invalid mnemonic must return nil record, got %+v", rec)
	}
}

func TestRecoverBit44Address_RejectsBadChecksum(t *testing.T) {
	m, _ := newTestManager(t)

	// 12 valid BIP-39 words but combined they fail the checksum.
	// All-"abandon" with last word "abandon" is invalid; the canonical
	// valid one ends with "about".
	badChecksum := []string{
		"abandon", "abandon", "abandon", "abandon",
		"abandon", "abandon", "abandon", "abandon",
		"abandon", "abandon", "abandon", "abandon",
	}
	rec, err := m.RecoverBit44Address(badChecksum)
	if err == nil {
		t.Fatal("bad-checksum mnemonic must produce an error")
	}
	if rec != nil {
		t.Fatalf("bad-checksum mnemonic must return nil record, got %+v", rec)
	}
}

func TestRecoverBit44Address_RejectsWrongLength(t *testing.T) {
	m, _ := newTestManager(t)

	// 5 words is not a valid BIP-39 length (12/15/18/21/24 only).
	short := []string{"abandon", "abandon", "abandon", "abandon", "abandon"}
	rec, err := m.RecoverBit44Address(short)
	if err == nil {
		t.Fatal("wrong-length mnemonic must produce an error")
	}
	if rec != nil {
		t.Fatalf("wrong-length mnemonic must return nil record")
	}
}

// Sanity: a valid round-trip still works after the fix. Generates a fresh
// mnemonic, recovers the address from it, and asserts equality.
func TestRecoverBit44Address_ValidRoundTrip(t *testing.T) {
	m, _ := newTestManager(t)

	original, err := m.GenerateBit44AddressWithLen(12)
	if err != nil {
		t.Fatalf("GenerateBit44AddressWithLen: %v", err)
	}
	if len(original.Bip39Mnemonic) != 12 {
		t.Fatalf("expected 12-word mnemonic, got %d", len(original.Bip39Mnemonic))
	}

	recovered, err := m.RecoverBit44Address(original.Bip39Mnemonic)
	if err != nil {
		t.Fatalf("recover from own mnemonic failed: %v", err)
	}
	if recovered == nil {
		t.Fatal("recovered record is nil")
	}
	if recovered.Address != original.Address {
		t.Fatalf("address mismatch: got %s want %s",
			recovered.Address, original.Address)
	}
}
