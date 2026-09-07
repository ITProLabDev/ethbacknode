package uniclient

import (
	"encoding/json"
	"testing"
)

// fakeTransport is an in-memory Transport for unit-testing RPC methods
// without a live server: it records the outgoing request and replays a
// canned result (or RPC error) into the response.
type fakeTransport struct {
	lastRequest *Request
	result      interface{}
	rpcErr      *RpcError
}

func (f *fakeTransport) Call(request *Request, response interface{}) error {
	f.lastRequest = request
	resp := response.(*Response)
	if f.rpcErr != nil {
		resp.Error = f.rpcErr
		return nil
	}
	b, err := json.Marshal(f.result)
	if err != nil {
		return err
	}
	resp.Result = b
	return nil
}

func newFakeClient(result interface{}) (*Client, *fakeTransport) {
	ft := &fakeTransport{result: result}
	c := &Client{rpcClient: ft, serviceId: 42}
	return c, ft
}

// lastParams re-marshals the fake transport's last recorded request params
// into dst, so a test can assert on the JSON actually sent without depending
// on Go struct field order.
func lastParams(t *testing.T, ft *fakeTransport, dst interface{}) {
	t.Helper()
	b, err := json.Marshal(ft.lastRequest.Params)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, dst); err != nil {
		t.Fatal(err)
	}
}

func TestContractRegister_SendsRequestAndParsesResult(t *testing.T) {
	c, ft := newFakeClient(map[string]string{"name": "Tok", "address": "0xabc", "status": "registered"})

	result, err := c.ContractRegister("Tok", "TOK", "0xabc", `[{"type":"function"}]`)
	if err != nil {
		t.Fatal(err)
	}
	if ft.lastRequest.Method != "contractRegister" {
		t.Fatalf("method=%q", ft.lastRequest.Method)
	}
	var params struct {
		Name, Symbol, Address, ABI string
	}
	lastParams(t, ft, &params)
	if params.Name != "Tok" || params.Symbol != "TOK" || params.Address != "0xabc" || params.ABI == "" {
		t.Fatalf("params=%+v", params)
	}
	if result.Name != "Tok" || result.Address != "0xabc" || result.Status != "registered" {
		t.Fatalf("result=%+v", result)
	}
}

func TestContractRegister_PropagatesRpcError(t *testing.T) {
	c, _ := newFakeClient(nil)
	c.rpcClient.(*fakeTransport).rpcErr = &RpcError{Code: -32600, Message: "address and abi are required"}
	if _, err := c.ContractRegister("", "", "", ""); err == nil {
		t.Fatal("expected an error")
	}
}

func TestContractSubscribe_SendsServiceIdAndFilters(t *testing.T) {
	c, ft := newFakeClient(map[string]interface{}{
		"serviceId": 42, "address": "0xabc", "scope": "whole_contract", "status": "subscribed",
	})

	result, err := c.ContractSubscribe("0xabc", "whole_contract", []string{"0xa9059cbb"}, []string{"transfer(address,uint256)"})
	if err != nil {
		t.Fatal(err)
	}
	if ft.lastRequest.Method != "contractSubscribe" {
		t.Fatalf("method=%q", ft.lastRequest.Method)
	}
	var params struct {
		ServiceID int      `json:"serviceId"`
		Address   string   `json:"address"`
		Scope     string   `json:"scope"`
		Selectors []string `json:"selectors"`
		Methods   []string `json:"methods"`
	}
	lastParams(t, ft, &params)
	if params.ServiceID != 42 || params.Address != "0xabc" || params.Scope != "whole_contract" {
		t.Fatalf("params=%+v", params)
	}
	if len(params.Selectors) != 1 || params.Selectors[0] != "0xa9059cbb" {
		t.Fatalf("selectors=%v", params.Selectors)
	}
	if len(params.Methods) != 1 || params.Methods[0] != "transfer(address,uint256)" {
		t.Fatalf("methods=%v", params.Methods)
	}
	if result.Status != "subscribed" {
		t.Fatalf("result=%+v", result)
	}
}

func TestContractSubscribe_WithoutFilters(t *testing.T) {
	c, ft := newFakeClient(map[string]interface{}{"status": "subscribed"})
	if _, err := c.ContractSubscribe("0xabc", "managed_only", nil, nil); err != nil {
		t.Fatal(err)
	}
	var params struct {
		Selectors []string `json:"selectors"`
		Methods   []string `json:"methods"`
	}
	lastParams(t, ft, &params)
	if len(params.Selectors) != 0 || len(params.Methods) != 0 {
		t.Fatalf("expected no filters sent, got selectors=%v methods=%v", params.Selectors, params.Methods)
	}
}

func TestContractUnsubscribe_SendsServiceIdAndAddress(t *testing.T) {
	c, ft := newFakeClient(map[string]interface{}{"serviceId": 42, "address": "0xabc", "status": "unsubscribed"})

	result, err := c.ContractUnsubscribe("0xabc")
	if err != nil {
		t.Fatal(err)
	}
	if ft.lastRequest.Method != "contractUnsubscribe" {
		t.Fatalf("method=%q", ft.lastRequest.Method)
	}
	var params struct {
		ServiceID int    `json:"serviceId"`
		Address   string `json:"address"`
	}
	lastParams(t, ft, &params)
	if params.ServiceID != 42 || params.Address != "0xabc" {
		t.Fatalf("params=%+v", params)
	}
	if result.Status != "unsubscribed" {
		t.Fatalf("result=%+v", result)
	}
}

func TestContractList_ParsesNameToAddressMap(t *testing.T) {
	c, ft := newFakeClient(map[string]string{"Tok": "0xabc", "Other": "0xdef"})

	result, err := c.ContractList()
	if err != nil {
		t.Fatal(err)
	}
	if ft.lastRequest.Method != "contractList" {
		t.Fatalf("method=%q", ft.lastRequest.Method)
	}
	if result["Tok"] != "0xabc" || result["Other"] != "0xdef" {
		t.Fatalf("result=%v", result)
	}
}

func TestContractSubscriptions_ParsesList(t *testing.T) {
	c, ft := newFakeClient([]map[string]interface{}{
		{"serviceId": "42", "address": "0xabc", "scope": "whole_contract", "selectors": []string{"0xa9059cbb"}},
	})

	result, err := c.ContractSubscriptions()
	if err != nil {
		t.Fatal(err)
	}
	if ft.lastRequest.Method != "contractSubscriptions" {
		t.Fatalf("method=%q", ft.lastRequest.Method)
	}
	if len(result) != 1 || result[0].ServiceID != "42" || result[0].Address != "0xabc" || result[0].Scope != "whole_contract" {
		t.Fatalf("result=%+v", result)
	}
	if len(result[0].Selectors) != 1 || result[0].Selectors[0] != "0xa9059cbb" {
		t.Fatalf("selectors=%v", result[0].Selectors)
	}
}

func TestContractCall_NoArgs_ParsesDecodedValues(t *testing.T) {
	c, ft := newFakeClient([]map[string]interface{}{
		{"type": "uint256", "value": "100000000000000000000"},
	})

	result, err := c.ContractCall("0xabc", "totalSupply")
	if err != nil {
		t.Fatal(err)
	}
	if ft.lastRequest.Method != "contractCall" {
		t.Fatalf("method=%q", ft.lastRequest.Method)
	}
	var params struct {
		Address string        `json:"address"`
		Method  string        `json:"method"`
		Args    []interface{} `json:"args,omitempty"`
	}
	lastParams(t, ft, &params)
	if params.Address != "0xabc" || params.Method != "totalSupply" || len(params.Args) != 0 {
		t.Fatalf("params=%+v", params)
	}
	if len(result) != 1 || result[0].Type != "uint256" || result[0].Value != "100000000000000000000" {
		t.Fatalf("result=%+v", result)
	}
}

func TestContractCall_WithArgs_SendsThemPositionally(t *testing.T) {
	c, ft := newFakeClient([]map[string]interface{}{})
	if _, err := c.ContractCall("0xabc", "balanceOf", "0xdef"); err != nil {
		t.Fatal(err)
	}
	var params struct {
		Args []interface{} `json:"args"`
	}
	lastParams(t, ft, &params)
	if len(params.Args) != 1 || params.Args[0] != "0xdef" {
		t.Fatalf("args=%v", params.Args)
	}
}
