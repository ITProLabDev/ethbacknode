package ethclient

import (
	"encoding/json"
	"errors"
	"strings"
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
	raw, _ := json.Marshal(request)
	var env struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	_ = json.Unmarshal(raw, &env)
	f.lastMethod = env.Method
	f.lastParams = env.Params
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
	if !strings.Contains(string(ft.lastParams), "0xabc0000000000000000000000000000000000000000000000000000000000001") {
		t.Fatalf("txHash not passed in params: %s", ft.lastParams)
	}
	if r.Status != 1 || len(r.Logs) != 1 {
		t.Fatalf("receipt not decoded: %+v", r)
	}
}

func TestGetTransactionReceipt_NotFound(t *testing.T) {
	c, _ := newFakeClient("null")
	_, err := c.GetTransactionReceipt("0xdead")
	if !errors.Is(err, ErrTransactionNotFound) {
		t.Fatalf("null result must return ErrTransactionNotFound, got %v", err)
	}
}

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
	params := string(ft.lastParams)
	for _, want := range []string{"0x3e8", "0x7d0", "0xdac17f958d2ee523a2206206994597c13d831ec7", "ddf252ad"} {
		if !strings.Contains(params, want) {
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

// jsonString wraps a raw JSON string value as a json.RawMessage result.
func jsonString(hexResult string) json.RawMessage {
	b, _ := json.Marshal(hexResult)
	return b
}

// urpcClientWith builds a urpc.Client around a fake transport (test seam).
func urpcClientWith(ft *fakeTransport) *urpc.Client {
	return urpc.NewClientWithTransport(ft)
}

// memABIStore is an in-memory storage.BinStorage for the abi manager in tests.
type memABIStore struct {
	data   []byte
	exists bool
}

func (s *memABIStore) IsExists() bool        { return s.exists }
func (s *memABIStore) Save(b []byte) error   { s.data = append(s.data[:0], b...); s.exists = true; return nil }
func (s *memABIStore) Load() ([]byte, error) { return s.data, nil }

func newMemABIStorage() *memABIStore { return &memABIStore{} }
