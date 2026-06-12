package ethclient

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

// memBinStorage is an in-memory storage.BinStorage for tests.
type memBinStorage struct {
	mu     sync.Mutex
	data   []byte
	exists bool
}

func (s *memBinStorage) IsExists() bool { return s.exists }
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
	out := make([]byte, len(s.data))
	copy(out, s.data)
	return out, nil
}

func TestChainProfileByName_Eth(t *testing.T) {
	p, err := ChainProfileByName("Eth")
	if err != nil {
		t.Fatal(err)
	}
	if p.ChainName != "Ethereum" || p.ChainId != "ethereum" {
		t.Fatalf("eth identity: %+v", p)
	}
	if p.ChainSymbol != "ETH" || p.Decimals != 18 {
		t.Fatalf("eth symbol/decimals: %+v", p)
	}
	if p.Confirmations != 20 {
		t.Fatalf("eth confirmations=%d want 20", p.Confirmations)
	}
	if len(p.Tokens) == 0 {
		t.Fatal("eth profile should seed default tokens")
	}
}

func TestChainProfileByName_Arc(t *testing.T) {
	p, err := ChainProfileByName("Arc")
	if err != nil {
		t.Fatal(err)
	}
	if p.ChainName != "Arc Testnet" || p.ChainId != "arc-testnet" {
		t.Fatalf("arc identity: %+v", p)
	}
	// Circle Arc's native gas token is USDC.
	if p.ChainSymbol != "USDC" || p.Decimals != 18 {
		t.Fatalf("arc symbol/decimals: %+v", p)
	}
	if p.Confirmations != 1 {
		t.Fatalf("arc confirmations=%d want 1", p.Confirmations)
	}
	if len(p.Tokens) != 0 {
		t.Fatalf("arc profile should seed no tokens, got %d", len(p.Tokens))
	}
}

func TestChainProfileByName_CaseInsensitive(t *testing.T) {
	for _, n := range []string{"eth", "ETH", "EtH", "arc", "ARC"} {
		if _, err := ChainProfileByName(n); err != nil {
			t.Fatalf("%q should resolve: %v", n, err)
		}
	}
}

func TestChainProfileByName_Unknown(t *testing.T) {
	for _, n := range []string{"", "btc", "tron", "polygon"} {
		_, err := ChainProfileByName(n)
		if err == nil {
			t.Fatalf("%q must be rejected", n)
		}
		if !strings.Contains(err.Error(), "unknown chain") {
			t.Fatalf("%q: error should mention unknown chain, got %v", n, err)
		}
	}
}

func TestApplyProfile_PopulatesConfig(t *testing.T) {
	p, err := ChainProfileByName("Arc")
	if err != nil {
		t.Fatal(err)
	}
	c := &Config{}
	c.ApplyProfile(p)
	if c.ChainName != "Arc Testnet" || c.ChainSymbol != "USDC" || c.Decimals != 18 || c.Confirmations != 1 {
		t.Fatalf("config not populated from profile: %+v", c)
	}
}

func TestInitClientConfig_WritesProfile(t *testing.T) {
	st := &memBinStorage{}
	if err := InitClientConfig(st, arcProfile()); err != nil {
		t.Fatal(err)
	}
	if !st.exists {
		t.Fatal("client config should have been written")
	}
	raw, _ := st.Load()
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("written config not valid json: %v", err)
	}
	if c.ChainName != "Arc Testnet" || c.ChainSymbol != "USDC" {
		t.Fatalf("written config wrong identity: %+v", c)
	}
}

func TestInitClientConfig_NilStorage(t *testing.T) {
	if err := InitClientConfig(nil, ethProfile()); err == nil {
		t.Fatal("nil storage must error")
	}
}
