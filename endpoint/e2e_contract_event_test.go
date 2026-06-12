package endpoint_test

// End-to-end test for the contract-event delivery path (M7.3) and the M5
// managed_only checksummed-address invariant. It wires the REAL components the
// way main.go does — abi.SmartContractsManager (decode), eventlog.Service
// (collect + scope filter), the EIP-55 address codec, a real managed-address
// pool, and endpoint.NewContractEventSink — then replays a block's log and
// asserts the delivered contractEvent payload (the exact JSON a client
// receives over its webhook).
//
// It lives in package endpoint_test (not endpoint) on purpose: importing
// clients/ethclient here for the real EIP-55 codec does NOT make the production
// endpoint package depend on ethclient, so the decoupling constraint holds.

import (
	"encoding/json"
	"math/big"
	"strings"
	"sync"
	"testing"

	"github.com/ITProLabDev/ethbacknode/abi"
	"github.com/ITProLabDev/ethbacknode/clients/ethclient"
	"github.com/ITProLabDev/ethbacknode/crypto"
	"github.com/ITProLabDev/ethbacknode/endpoint"
	"github.com/ITProLabDev/ethbacknode/eventlog"
	"github.com/ITProLabDev/ethbacknode/storage"
)

// managedSet is a faithful stand-in for address.Manager as seen by eventlog's
// ManagedAddresses seam. The production pool (address.Manager) resolves
// membership via fastPool.LookupString, which is an EXACT, case-sensitive byte
// match on the address string. This stub reproduces that keying exactly, so the
// checksummed-address invariant it exercises is identical to production — while
// avoiding the pool's Badger DB and async updatePool() (which would make
// IsAddressKnown racy in a unit test). The EIP-55 codec under test stays real.
type managedSet struct{ known map[string]bool }

func (m managedSet) IsAddressKnown(addr string) bool { return m.known[addr] }

// memBin is an in-memory storage.BinStorage for wiring the abi manager.
type memBin struct {
	mu     sync.Mutex
	data   []byte
	exists bool
}

func (s *memBin) IsExists() bool { return s.exists }
func (s *memBin) Save(b []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = append(s.data[:0], b...)
	s.exists = true
	return nil
}
func (s *memBin) Load() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data, nil
}

var _ storage.BinStorage = (*memBin)(nil)

// capturedEvent is one delivered contractEvent, captured at the notifier as the
// (serviceID, subject, payload) the subscriptions layer would send on the wire.
type capturedEvent struct {
	serviceID int64
	subject   string
	payload   interface{}
}

// leftPad32 left-pads b into a 32-byte slot (topic / uint encoding).
func leftPad32(b []byte) [32]byte {
	var out [32]byte
	copy(out[32-len(b):], b)
	return out
}

// TestE2E_ContractEvent_DeliveredEndToEnd registers a generic multi-token-class
// contract, replays an ERC-1155-style TransferSingle log through the real
// eventlog + abi + sink stack, and asserts the delivered contractEvent wire
// JSON for BOTH scopes, including the checksummed managed-address invariant.
func TestE2E_ContractEvent_DeliveredEndToEnd(t *testing.T) {
	codec := ethclient.GetAddressCodec() // real EIP-55 checksum codec

	// --- a real managed address (EIP-55 checksummed) -------------------------
	managedBytes := make([]byte, 20)
	for i := range managedBytes {
		managedBytes[i] = byte(0x10 + i)
	}
	managedChecksummed, err := codec.EncodeBytesToAddress(managedBytes)
	if err != nil {
		t.Fatal(err)
	}
	// Sanity: the codec must actually produce a mixed-case (checksummed) form,
	// otherwise the invariant under test would be vacuous.
	if managedChecksummed == strings.ToLower(managedChecksummed) {
		t.Fatalf("expected a checksummed (mixed-case) address, got %q", managedChecksummed)
	}

	// A non-managed counterparty address.
	otherBytes := make([]byte, 20)
	for i := range otherBytes {
		otherBytes[i] = byte(0xA0 + i)
	}

	// --- managed-address set (keyed by the checksummed string, like the pool) -
	pool := managedSet{known: map[string]bool{managedChecksummed: true}}
	if !pool.IsAddressKnown(managedChecksummed) {
		t.Fatalf("managed set must know %q", managedChecksummed)
	}

	// --- real abi manager: register a generic multi-token contract ----------
	abiManager := abi.NewManager(abi.WithStorage(&memBin{}), abi.WithAddressCodec(codec))
	if err := abiManager.Init(); err != nil {
		t.Fatalf("abi Init: %v", err)
	}
	const contractAddr = "0xCONTRACT00000000000000000000000000000001" // case-preserved
	const rawABI = `[
	  {"type":"event","name":"TransferSingle","inputs":[
	    {"name":"operator","type":"address","indexed":true},
	    {"name":"from","type":"address","indexed":true},
	    {"name":"to","type":"address","indexed":true},
	    {"name":"id","type":"uint256"},
	    {"name":"value","type":"uint256"}]}
	]`
	if err := abiManager.AddContractFromABI("MultiToken", "MT", contractAddr, []byte(rawABI)); err != nil {
		t.Fatalf("register contract: %v", err)
	}

	// Build an authentic log for TransferSingle(operator, from=other, to=managed, id, value)
	// using the engine's own topic0 so the test stays consistent with decoding.
	contract, err := abiManager.GetSmartContractByAddress(contractAddr)
	if err != nil {
		t.Fatalf("lookup contract: %v", err)
	}
	entry, err := contract.Abi.GetEventByTopic0(eventTopic0(t, "TransferSingle(address,address,address,uint256,uint256)"))
	if err != nil {
		t.Fatalf("event entry: %v", err)
	}
	topic0 := entry.Topic0()

	// indexed topics: operator, from(other), to(managed)
	operatorTopic := leftPad32(managedBytes) // operator == managed (also a hit vector)
	fromTopic := leftPad32(otherBytes)
	toTopic := leftPad32(managedBytes)
	topics := []string{
		"0x" + hexOf(topic0[:]),
		"0x" + hexOf(operatorTopic[:]),
		"0x" + hexOf(fromTopic[:]),
		"0x" + hexOf(toTopic[:]),
	}
	// non-indexed data: id=7, value=1e18
	id := leftPad32(big.NewInt(7).Bytes())
	val, _ := new(big.Int).SetString("1000000000000000000", 10)
	value := leftPad32(val.Bytes())
	data := append(append([]byte{}, id[:]...), value[:]...)

	rawLog := eventlog.RawLog{
		Address:          contractAddr,
		Topics:           topics,
		Data:             data,
		BlockNumber:      20123456,
		TransactionHash:  "0xdeadbeef",
		TransactionIndex: 2,
		LogIndex:         7,
	}

	// --- build the eventlog service for a given scope, capture delivery ------
	run := func(t *testing.T, scope string, subscriberAddr string) []capturedEvent {
		t.Helper()
		var mu sync.Mutex
		var captured []capturedEvent
		sink := endpoint.NewContractEventSink(endpoint.NotifierFunc(
			func(serviceID int64, subject string, payload interface{}) {
				mu.Lock()
				captured = append(captured, capturedEvent{serviceID, subject, payload})
				mu.Unlock()
			}))

		svc := eventlog.New(
			eventlog.WithLogSource(eventlog.RawLogSource{
				GetLogsFn: func(from, to int64, addresses, topics []string) ([]eventlog.RawLog, error) {
					return []eventlog.RawLog{rawLog}, nil
				},
			}),
			eventlog.WithDecoder(eventlog.NewDecoder(abiManager.DecodeLog)),
			eventlog.WithManaged(pool),
			eventlog.WithAddressCodec(codec),
			eventlog.WithSink(sink),
		)
		if err := svc.SubscribeAndSaveStrings("42", subscriberAddr, scope); err != nil {
			t.Fatalf("subscribe(%s): %v", scope, err)
		}
		svc.OnBlock(rawLog.BlockNumber, "0xblock")

		mu.Lock()
		defer mu.Unlock()
		out := make([]capturedEvent, len(captured))
		copy(out, captured)
		return out
	}

	t.Run("whole_contract delivers decoded event with safe JSON", func(t *testing.T) {
		got := run(t, "whole_contract", contractAddr)
		if len(got) != 1 {
			t.Fatalf("delivered %d events, want 1", len(got))
		}
		ce := got[0]
		if ce.serviceID != 42 || ce.subject != "contractEvent" {
			t.Fatalf("serviceID=%d subject=%q", ce.serviceID, ce.subject)
		}

		// Assert the wire JSON shape a client receives.
		b, err := json.Marshal(ce.payload)
		if err != nil {
			t.Fatal(err)
		}
		var wire struct {
			Event    string `json:"event"`
			Contract string `json:"contract"`
			BlockNum int64  `json:"blockNum"`
			TxHash   string `json:"txHash"`
			TxIndex  int64  `json:"txIndex"`
			LogIndex int64  `json:"logIndex"`
			Inputs   []struct {
				Name  string          `json:"name"`
				Type  string          `json:"type"`
				Value json.RawMessage `json:"value"`
			} `json:"inputs"`
		}
		if err := json.Unmarshal(b, &wire); err != nil {
			t.Fatalf("payload not valid JSON: %v\n%s", err, b)
		}
		if wire.Event != "TransferSingle" || wire.Contract != contractAddr {
			t.Fatalf("event=%q contract=%q", wire.Event, wire.Contract)
		}
		if wire.BlockNum != 20123456 || wire.TxHash != "0xdeadbeef" || wire.TxIndex != 2 || wire.LogIndex != 7 {
			t.Fatalf("context wrong: %+v", wire)
		}
		if len(wire.Inputs) != 5 {
			t.Fatalf("inputs=%d want 5", len(wire.Inputs))
		}
		// address inputs must be 0x-hex strings, NOT base64
		byName := map[string]string{}
		for _, in := range wire.Inputs {
			byName[in.Name] = string(in.Value)
		}
		if got := byName["to"]; got != `"0x`+hexOf(managedBytes)+`"` {
			t.Fatalf("to value=%s want 0x-hex of managed", got)
		}
		// uint256 value must be a decimal STRING (JS-safe), not a bare number
		if got := byName["value"]; got != `"1000000000000000000"` {
			t.Fatalf("value=%s want quoted decimal string", got)
		}
		if got := byName["id"]; got != `"7"` {
			t.Fatalf("id=%s want \"7\"", got)
		}
	})

	t.Run("managed_only matches via checksummed managed address", func(t *testing.T) {
		got := run(t, "managed_only", contractAddr)
		if len(got) != 1 {
			t.Fatalf("managed_only must deliver (managed addr is involved), got %d", len(got))
		}
	})

	t.Run("managed_only does NOT match when no managed address involved", func(t *testing.T) {
		// Register a second contract whose log involves only non-managed addresses.
		const c2 = "0xCONTRACT00000000000000000000000000000002"
		if err := abiManager.AddContractFromABI("MultiToken2", "MT2", c2, []byte(rawABI)); err != nil {
			t.Fatalf("register c2: %v", err)
		}
		other2 := make([]byte, 20)
		for i := range other2 {
			other2[i] = byte(0xB0 + i)
		}
		t0 := eventTopic0(t, "TransferSingle(address,address,address,uint256,uint256)")
		op := leftPad32(other2)
		fr := leftPad32(other2)
		to := leftPad32(other2)
		log2 := eventlog.RawLog{
			Address:         c2,
			Topics:          []string{"0x" + hexOf(t0[:]), "0x" + hexOf(op[:]), "0x" + hexOf(fr[:]), "0x" + hexOf(to[:])},
			Data:            data,
			BlockNumber:     20123457,
			TransactionHash: "0xfeed",
			LogIndex:        0,
		}
		var mu sync.Mutex
		var n int
		sink := endpoint.NewContractEventSink(endpoint.NotifierFunc(
			func(serviceID int64, subject string, payload interface{}) { mu.Lock(); n++; mu.Unlock() }))
		svc := eventlog.New(
			eventlog.WithLogSource(eventlog.RawLogSource{
				GetLogsFn: func(from, to int64, addresses, topics []string) ([]eventlog.RawLog, error) {
					return []eventlog.RawLog{log2}, nil
				},
			}),
			eventlog.WithDecoder(eventlog.NewDecoder(abiManager.DecodeLog)),
			eventlog.WithManaged(pool),
			eventlog.WithAddressCodec(codec),
			eventlog.WithSink(sink),
		)
		if err := svc.SubscribeAndSaveStrings("42", c2, "managed_only"); err != nil {
			t.Fatal(err)
		}
		svc.OnBlock(log2.BlockNumber, "0xblock2")
		mu.Lock()
		defer mu.Unlock()
		if n != 0 {
			t.Fatalf("managed_only must NOT deliver when no managed address involved, got %d", n)
		}
	})
}

// eventTopic0 computes keccak256(canonicalSig) as the engine does, for fixtures.
func eventTopic0(t *testing.T, canonicalSig string) [32]byte {
	t.Helper()
	h := crypto.Keccak256([]byte(canonicalSig))
	var out [32]byte
	copy(out[:], h)
	return out
}

// hexOf returns the lowercase hex (no 0x) of b.
func hexOf(b []byte) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, 0, len(b)*2)
	for _, by := range b {
		out = append(out, hexdigits[by>>4], hexdigits[by&0xf])
	}
	return string(out)
}
