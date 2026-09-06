package ethclient

import (
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/ITProLabDev/ethbacknode/clients/urpc"
)

// methodFakeTransport answers a different canned result (or error) per RPC
// method name, so a code path calling several distinct methods -- as
// resolveFees does -- can be exercised without a real node.
type methodFakeTransport struct {
	results map[string]json.RawMessage
	errs    map[string]error
}

func (f *methodFakeTransport) Call(request interface{}, response interface{}) error {
	raw, _ := json.Marshal(request)
	var env struct {
		Method string `json:"method"`
	}
	_ = json.Unmarshal(raw, &env)
	if err, ok := f.errs[env.Method]; ok {
		return err
	}
	resp, _ := response.(*urpc.Response)
	if resp != nil {
		resp.Result = f.results[env.Method]
	}
	return nil
}

func newMethodFakeClient(results map[string]json.RawMessage, errs map[string]error) *Client {
	return &Client{
		rpcClient: urpc.NewClientWithTransport(&methodFakeTransport{results: results, errs: errs}),
		config:    &Config{},
	}
}

func TestBaseFeeReturnsTheLatestBlocksBaseFee(t *testing.T) {
	c := newMethodFakeClient(map[string]json.RawMessage{
		ethGetBlockByNumber: json.RawMessage(`{"baseFeePerGas":"0x3b9aca00"}`), // 1 gwei
	}, nil)

	got, err := c.BaseFee()
	if err != nil {
		t.Fatalf("BaseFee: %v", err)
	}
	if got == nil || got.Cmp(big.NewInt(1_000_000_000)) != 0 {
		t.Fatalf("BaseFee = %v, want 1000000000", got)
	}
}

func TestBaseFeeReturnsNilOnAPreLondonChain(t *testing.T) {
	c := newMethodFakeClient(map[string]json.RawMessage{
		ethGetBlockByNumber: json.RawMessage(`{}`),
	}, nil)

	got, err := c.BaseFee()
	if err != nil {
		t.Fatalf("BaseFee: %v", err)
	}
	if got != nil {
		t.Fatalf("BaseFee = %v, want nil (no baseFeePerGas field)", got)
	}
}

func TestSuggestPriorityFeeReturnsTheNodesSuggestion(t *testing.T) {
	c := newMethodFakeClient(map[string]json.RawMessage{
		ethMaxPriorityFeePerGas: jsonString("0x77359400"), // 2 gwei
	}, nil)

	got, err := c.SuggestPriorityFee()
	if err != nil {
		t.Fatalf("SuggestPriorityFee: %v", err)
	}
	if got.Cmp(big.NewInt(2_000_000_000)) != 0 {
		t.Fatalf("SuggestPriorityFee = %s, want 2000000000", got)
	}
}

func TestSuggestPriorityFeeFallsBackWhenUnsupported(t *testing.T) {
	c := newMethodFakeClient(nil, map[string]error{
		ethMaxPriorityFeePerGas: errors.New("method eth_maxPriorityFeePerGas not found"),
	})

	got, err := c.SuggestPriorityFee()
	if err != nil {
		t.Fatalf("SuggestPriorityFee: %v", err)
	}
	if got.Cmp(defaultPriorityFeeWei) != 0 {
		t.Fatalf("SuggestPriorityFee fallback = %s, want %s", got, defaultPriorityFeeWei)
	}
}

func TestResolveFeesChoosesDynamicOnAChainWithABaseFee(t *testing.T) {
	c := newMethodFakeClient(map[string]json.RawMessage{
		ethGetBlockByNumber:     json.RawMessage(`{"baseFeePerGas":"0x3b9aca00"}`), // 1 gwei
		ethMaxPriorityFeePerGas: jsonString("0x3b9aca00"),                          // 1 gwei
	}, nil)

	fees, err := c.resolveFees()
	if err != nil {
		t.Fatalf("resolveFees: %v", err)
	}
	if !fees.dynamic {
		t.Fatal("resolveFees must choose the dynamic-fee envelope when the chain reports a base fee")
	}
	// cap = baseFee * 200% + tip = 2 gwei + 1 gwei = 3 gwei
	if want := big.NewInt(3_000_000_000); fees.cap().Cmp(want) != 0 {
		t.Fatalf("fee cap = %s, want %s", fees.cap(), want)
	}
	if fees.maxPriorityFeePerGas.Cmp(big.NewInt(1_000_000_000)) != 0 {
		t.Fatalf("tip = %s, want 1000000000", fees.maxPriorityFeePerGas)
	}
}

func TestResolveFeesChoosesLegacyOnAPreLondonChain(t *testing.T) {
	c := newMethodFakeClient(map[string]json.RawMessage{
		ethGetBlockByNumber: json.RawMessage(`{}`),
		ethGasPrice:         jsonString("0x77359400"), // 2 gwei
	}, nil)

	fees, err := c.resolveFees()
	if err != nil {
		t.Fatalf("resolveFees: %v", err)
	}
	if fees.dynamic {
		t.Fatal("resolveFees must choose the legacy envelope on a chain with no base fee")
	}
	if want := big.NewInt(2_000_000_000); fees.cap().Cmp(want) != 0 {
		t.Fatalf("fee cap = %s, want %s", fees.cap(), want)
	}
}

func TestResolveFeesRefusesAPriceAboveTheConfiguredCeiling(t *testing.T) {
	c := newMethodFakeClient(map[string]json.RawMessage{
		ethGetBlockByNumber: json.RawMessage(`{}`),
		ethGasPrice:         jsonString("0x77359400"), // 2 gwei
	}, nil)
	c.config.FeeCeilingGwei = 1 // below the 2 gwei the node suggests

	_, err := c.resolveFees()
	if !errors.Is(err, ErrFeeCeilingExceeded) {
		t.Fatalf("resolveFees err = %v, want ErrFeeCeilingExceeded", err)
	}
}

func TestResolveFeesWithinTheCeilingSucceeds(t *testing.T) {
	c := newMethodFakeClient(map[string]json.RawMessage{
		ethGetBlockByNumber: json.RawMessage(`{}`),
		ethGasPrice:         jsonString("0x77359400"), // 2 gwei
	}, nil)
	c.config.FeeCeilingGwei = 5 // above the 2 gwei the node suggests

	fees, err := c.resolveFees()
	if err != nil {
		t.Fatalf("resolveFees: %v", err)
	}
	if want := big.NewInt(2_000_000_000); fees.cap().Cmp(want) != 0 {
		t.Fatalf("fee cap = %s, want %s", fees.cap(), want)
	}
}
