package endpoint

import (
	"encoding/json"
	"testing"
)

// --- fakes for the contract RPC deps ---

type fakeRegistry struct{ list map[string]string }

func (f *fakeRegistry) GetSmartContractList() map[string]string { return f.list }

type fakeAdder struct {
	name, symbol, address string
	rawABI                []byte
	err                   error
}

func (f *fakeAdder) AddContractFromABI(name, symbol, address string, rawABI []byte) error {
	f.name, f.symbol, f.address, f.rawABI = name, symbol, address, rawABI
	return f.err
}

type fakeSubscriber struct {
	gotService, gotAddr, gotScope string
	unsubService, unsubAddr       string
	err                           error
	subs                          []map[string]string
}

func (f *fakeSubscriber) SubscribeAndSaveStrings(serviceID, contractAddress, scope string) error {
	f.gotService, f.gotAddr, f.gotScope = serviceID, contractAddress, scope
	return f.err
}
func (f *fakeSubscriber) UnsubscribeStrings(serviceID, contractAddress string) error {
	f.unsubService, f.unsubAddr = serviceID, contractAddress
	return f.err
}
func (f *fakeSubscriber) ListSubscriptions() []map[string]string { return f.subs }

// fakeReq/fakeResp implement RpcRequest/RpcResponse for processor tests.
type fakeReq struct{ params interface{} }

func (r *fakeReq) GetMethod() RpcMethod                  { return "" }
func (r *fakeReq) ParseParams(p interface{}) error       { b, _ := json.Marshal(r.params); return json.Unmarshal(b, p) }
func (r *fakeReq) GetParamString(string) (string, error) { return "", nil }
func (r *fakeReq) GetParamInt(string) (int64, error)     { return 0, nil }
func (r *fakeReq) GetParamBool(string) (bool, error)     { return false, nil }

type fakeResp struct {
	result   interface{}
	errCode  int
	errMsg   string
	hasError bool
}

func (r *fakeResp) SetResult(v interface{})                  { r.result = v }
func (r *fakeResp) SetError(code int, msg string)            { r.errCode = code; r.errMsg = msg; r.hasError = true }
func (r *fakeResp) SetErrorWithData(code int, msg, d string) { r.errCode = code; r.errMsg = msg; r.hasError = true }

func TestContractRegister_AddsContract(t *testing.T) {
	adder := &fakeAdder{}
	r := &BackRpc{contractAdder: adder}
	req := &fakeReq{params: map[string]interface{}{
		"name": "Tok", "symbol": "TOK", "address": "0xabc",
		"abi": `[{"type":"event","name":"E","inputs":[]}]`,
	}}
	resp := &fakeResp{}
	r.rpcProcessContractRegister(nil, req, resp)
	if resp.hasError {
		t.Fatalf("unexpected error: %s", resp.errMsg)
	}
	if adder.name != "Tok" || adder.address != "0xabc" || len(adder.rawABI) == 0 {
		t.Fatalf("adder got name=%q addr=%q abilen=%d", adder.name, adder.address, len(adder.rawABI))
	}
}

func TestContractList_ReturnsRegistry(t *testing.T) {
	r := &BackRpc{abiManager: &fakeRegistry{list: map[string]string{"Tok": "0xabc"}}}
	resp := &fakeResp{}
	r.rpcProcessContractList(nil, &fakeReq{}, resp)
	if resp.hasError {
		t.Fatal("unexpected error")
	}
	m, ok := resp.result.(map[string]string)
	if !ok || m["Tok"] != "0xabc" {
		t.Fatalf("result=%v", resp.result)
	}
}

func TestContractSubscribe_RegistersSubscription(t *testing.T) {
	sub := &fakeSubscriber{}
	r := &BackRpc{eventLog: sub}
	// serviceId is a JSON number; the handler converts it to its decimal string.
	req := &fakeReq{params: map[string]interface{}{
		"serviceId": 7, "address": "0xabc", "scope": "managed_only",
	}}
	resp := &fakeResp{}
	r.rpcProcessContractSubscribe(nil, req, resp)
	if resp.hasError {
		t.Fatalf("unexpected error: %s", resp.errMsg)
	}
	if sub.gotService != "7" || sub.gotAddr != "0xabc" || sub.gotScope != "managed_only" {
		t.Fatalf("subscriber got %q %q %q", sub.gotService, sub.gotAddr, sub.gotScope)
	}
}

func TestContractSubscribe_RejectsBadScope(t *testing.T) {
	sub := &fakeSubscriber{err: errBadScope}
	r := &BackRpc{eventLog: sub}
	req := &fakeReq{params: map[string]interface{}{"serviceId": 7, "address": "0xabc", "scope": "nonsense"}}
	resp := &fakeResp{}
	r.rpcProcessContractSubscribe(nil, req, resp)
	if !resp.hasError {
		t.Fatal("bad scope must error")
	}
}

func TestContractUnsubscribe_RemovesSubscription(t *testing.T) {
	sub := &fakeSubscriber{}
	r := &BackRpc{eventLog: sub}
	req := &fakeReq{params: map[string]interface{}{"serviceId": 7, "address": "0xabc"}}
	resp := &fakeResp{}
	r.rpcProcessContractUnsubscribe(nil, req, resp)
	if resp.hasError {
		t.Fatalf("unexpected error: %s", resp.errMsg)
	}
	if sub.unsubService != "7" || sub.unsubAddr != "0xabc" {
		t.Fatalf("unsubscribe got %q %q", sub.unsubService, sub.unsubAddr)
	}
}
