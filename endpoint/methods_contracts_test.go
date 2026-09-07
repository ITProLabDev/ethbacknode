package endpoint

import (
	"encoding/json"
	"errors"
	"testing"
)

// errBadScope is a test sentinel for an invalid-scope error returned by the
// fake subscriber.
var errBadScope = errors.New("invalid scope")

// --- fakes for the contract RPC deps ---

type fakeRegistry struct {
	list  map[string]string
	known map[string]bool
}

func (f *fakeRegistry) GetSmartContractList() map[string]string { return f.list }
func (f *fakeRegistry) IsContractKnown(address string) bool     { return f.known[address] }

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
	gotSelectors                  [][4]byte
	unsubService, unsubAddr       string
	err                           error
	subs                          []map[string]any
}

func (f *fakeSubscriber) SubscribeAndSaveStrings(serviceID, contractAddress, scope string, selectors [][4]byte) error {
	f.gotService, f.gotAddr, f.gotScope, f.gotSelectors = serviceID, contractAddress, scope, selectors
	return f.err
}
func (f *fakeSubscriber) UnsubscribeStrings(serviceID, contractAddress string) error {
	f.unsubService, f.unsubAddr = serviceID, contractAddress
	return f.err
}
func (f *fakeSubscriber) ListSubscriptions() []map[string]any { return f.subs }

// fakeReq/fakeResp implement RpcRequest/RpcResponse for processor tests.
type fakeReq struct{ params interface{} }

func (r *fakeReq) GetMethod() RpcMethod { return "" }
func (r *fakeReq) ParseParams(p interface{}) error {
	b, _ := json.Marshal(r.params)
	return json.Unmarshal(b, p)
}
func (r *fakeReq) GetParamString(string) (string, error) { return "", nil }
func (r *fakeReq) GetParamInt(string) (int64, error)     { return 0, nil }
func (r *fakeReq) GetParamBool(string) (bool, error)     { return false, nil }

type fakeResp struct {
	result   interface{}
	errCode  int
	errMsg   string
	hasError bool
}

func (r *fakeResp) SetResult(v interface{}) { r.result = v }
func (r *fakeResp) SetError(code int, msg string) {
	r.errCode = code
	r.errMsg = msg
	r.hasError = true
}
func (r *fakeResp) SetErrorWithData(code int, msg, d string) {
	r.errCode = code
	r.errMsg = msg
	r.hasError = true
}

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
	reg := &fakeRegistry{known: map[string]bool{"0xabc": true}}
	r := &BackRpc{eventLog: sub, abiManager: reg}
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

// A subscriber can name a raw 4-byte hex selector directly -- works even for
// a method not in this node's registered ABI, since it needs no lookup.
func TestContractSubscribe_ResolvesRawSelectors(t *testing.T) {
	sub := &fakeSubscriber{}
	reg := &fakeRegistry{known: map[string]bool{"0xabc": true}}
	r := &BackRpc{eventLog: sub, abiManager: reg}
	req := &fakeReq{params: map[string]interface{}{
		"serviceId": 7, "address": "0xabc", "scope": "whole_contract",
		"selectors": []string{"0xa9059cbb"},
	}}
	resp := &fakeResp{}
	r.rpcProcessContractSubscribe(nil, req, resp)
	if resp.hasError {
		t.Fatalf("unexpected error: %s", resp.errMsg)
	}
	want := [4]byte{0xa9, 0x05, 0x9c, 0xbb}
	if len(sub.gotSelectors) != 1 || sub.gotSelectors[0] != want {
		t.Fatalf("selectors=%x want [%x]", sub.gotSelectors, want)
	}
}

// A subscriber can instead name a canonical signature, resolved to its
// selector the same way abi's own signature hashing does -- no ABI lookup
// needed, so this works for any signature the caller knows, registered or not.
func TestContractSubscribe_ResolvesCanonicalMethodSignatures(t *testing.T) {
	sub := &fakeSubscriber{}
	reg := &fakeRegistry{known: map[string]bool{"0xabc": true}}
	r := &BackRpc{eventLog: sub, abiManager: reg}
	req := &fakeReq{params: map[string]interface{}{
		"serviceId": 7, "address": "0xabc", "scope": "whole_contract",
		"methods": []string{"transfer(address,uint256)"},
	}}
	resp := &fakeResp{}
	r.rpcProcessContractSubscribe(nil, req, resp)
	if resp.hasError {
		t.Fatalf("unexpected error: %s", resp.errMsg)
	}
	want := [4]byte{0xa9, 0x05, 0x9c, 0xbb}
	if len(sub.gotSelectors) != 1 || sub.gotSelectors[0] != want {
		t.Fatalf("selectors=%x want [%x]", sub.gotSelectors, want)
	}
}

// Both forms together (the hybrid) merge into one filter set.
func TestContractSubscribe_MergesSelectorsAndMethods(t *testing.T) {
	sub := &fakeSubscriber{}
	reg := &fakeRegistry{known: map[string]bool{"0xabc": true}}
	r := &BackRpc{eventLog: sub, abiManager: reg}
	req := &fakeReq{params: map[string]interface{}{
		"serviceId": 7, "address": "0xabc", "scope": "whole_contract",
		"selectors": []string{"0xdeadbeef"},
		"methods":   []string{"transfer(address,uint256)"},
	}}
	resp := &fakeResp{}
	r.rpcProcessContractSubscribe(nil, req, resp)
	if resp.hasError {
		t.Fatalf("unexpected error: %s", resp.errMsg)
	}
	if len(sub.gotSelectors) != 2 {
		t.Fatalf("selectors=%x want 2 entries", sub.gotSelectors)
	}
}

// No selectors/methods named at all must mean no filter (nil), not an empty
// slice that would (mis)match nothing -- preserving today's behavior for a
// caller that never asks for filtering.
func TestContractSubscribe_NoSelectorsMeansNoFilter(t *testing.T) {
	sub := &fakeSubscriber{}
	reg := &fakeRegistry{known: map[string]bool{"0xabc": true}}
	r := &BackRpc{eventLog: sub, abiManager: reg}
	req := &fakeReq{params: map[string]interface{}{
		"serviceId": 7, "address": "0xabc", "scope": "whole_contract",
	}}
	resp := &fakeResp{}
	r.rpcProcessContractSubscribe(nil, req, resp)
	if resp.hasError {
		t.Fatalf("unexpected error: %s", resp.errMsg)
	}
	if sub.gotSelectors != nil {
		t.Fatalf("selectors=%x want nil (no filter)", sub.gotSelectors)
	}
}

func TestContractSubscribe_RejectsMalformedSelector(t *testing.T) {
	sub := &fakeSubscriber{}
	reg := &fakeRegistry{known: map[string]bool{"0xabc": true}}
	r := &BackRpc{eventLog: sub, abiManager: reg}
	req := &fakeReq{params: map[string]interface{}{
		"serviceId": 7, "address": "0xabc", "scope": "whole_contract",
		"selectors": []string{"not-hex"},
	}}
	resp := &fakeResp{}
	r.rpcProcessContractSubscribe(nil, req, resp)
	if !resp.hasError {
		t.Fatal("a malformed selector must be rejected")
	}
	if sub.gotAddr != "" {
		t.Fatalf("subscriber must not be called on a malformed selector, got addr=%q", sub.gotAddr)
	}
}

func TestContractSubscribe_RejectsSelectorOfWrongLength(t *testing.T) {
	sub := &fakeSubscriber{}
	reg := &fakeRegistry{known: map[string]bool{"0xabc": true}}
	r := &BackRpc{eventLog: sub, abiManager: reg}
	req := &fakeReq{params: map[string]interface{}{
		"serviceId": 7, "address": "0xabc", "scope": "whole_contract",
		"selectors": []string{"0xa9059c"}, // 3 bytes, not 4
	}}
	resp := &fakeResp{}
	r.rpcProcessContractSubscribe(nil, req, resp)
	if !resp.hasError {
		t.Fatal("a 3-byte selector must be rejected")
	}
}

func TestContractSubscribe_RejectsBadScope(t *testing.T) {
	sub := &fakeSubscriber{err: errBadScope}
	reg := &fakeRegistry{known: map[string]bool{"0xabc": true}}
	r := &BackRpc{eventLog: sub, abiManager: reg}
	req := &fakeReq{params: map[string]interface{}{"serviceId": 7, "address": "0xabc", "scope": "nonsense"}}
	resp := &fakeResp{}
	r.rpcProcessContractSubscribe(nil, req, resp)
	if !resp.hasError {
		t.Fatal("bad scope must error")
	}
}

// Subscribing to an address with no registered ABI must be rejected, not
// silently accepted (the subscriber would otherwise receive nothing because
// decode skips unknown contracts).
func TestContractSubscribe_RejectsUnknownContract(t *testing.T) {
	sub := &fakeSubscriber{}
	reg := &fakeRegistry{known: map[string]bool{}} // nothing registered
	r := &BackRpc{eventLog: sub, abiManager: reg}
	req := &fakeReq{params: map[string]interface{}{
		"serviceId": 7, "address": "0xdead", "scope": "whole_contract",
	}}
	resp := &fakeResp{}
	r.rpcProcessContractSubscribe(nil, req, resp)
	if !resp.hasError {
		t.Fatal("subscribing to an unregistered contract must error")
	}
	if sub.gotAddr != "" {
		t.Fatalf("subscriber must not be called for an unknown contract, got addr=%q", sub.gotAddr)
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
