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
