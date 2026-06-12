package endpoint

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ITProLabDev/ethbacknode/abi"
)

type fakeCaller struct {
	gotContract, gotMethod string
	gotArgs                []any
	out                    []abi.DecodedValue
	err                    error
}

func (f *fakeCaller) CallMethod(contract, method string, args ...any) ([]abi.DecodedValue, error) {
	f.gotContract, f.gotMethod, f.gotArgs = contract, method, args
	return f.out, f.err
}

func TestContractCall_ReturnsDecodedOutputs(t *testing.T) {
	caller := &fakeCaller{out: []abi.DecodedValue{{Name: "", Type: "uint256", Value: big.NewInt(42)}}}
	r := &BackRpc{contractCaller: caller}
	req := &fakeReq{params: map[string]interface{}{
		"address": "0xabc", "method": "totalSupply",
	}}
	resp := &fakeResp{}
	r.rpcProcessContractCall(nil, req, resp)
	if resp.hasError {
		t.Fatalf("unexpected error: %s", resp.errMsg)
	}
	if caller.gotContract != "0xabc" || caller.gotMethod != "totalSupply" {
		t.Fatalf("caller got %q.%q", caller.gotContract, caller.gotMethod)
	}
	out, ok := resp.result.([]abi.DecodedValue)
	if !ok || len(out) != 1 || out[0].Value.(*big.Int).Int64() != 42 {
		t.Fatalf("result=%v", resp.result)
	}
}

func TestContractCall_PropagatesError(t *testing.T) {
	caller := &fakeCaller{err: errBadScope} // any non-nil error
	r := &BackRpc{contractCaller: caller}
	req := &fakeReq{params: map[string]interface{}{"address": "0xabc", "method": "x"}}
	resp := &fakeResp{}
	r.rpcProcessContractCall(nil, req, resp)
	if !resp.hasError {
		t.Fatal("call error must surface as RPC error")
	}
}

func TestContractCall_RequiresAddressAndMethod(t *testing.T) {
	r := &BackRpc{contractCaller: &fakeCaller{}}
	resp := &fakeResp{}
	r.rpcProcessContractCall(nil, &fakeReq{params: map[string]interface{}{"address": "0xabc"}}, resp)
	if !resp.hasError {
		t.Fatal("missing method must error")
	}
}

func TestContractCall_RejectsArgs(t *testing.T) {
	r := &BackRpc{contractCaller: &fakeCaller{}}
	req := &fakeReq{params: map[string]interface{}{
		"address": "0xabc", "method": "transfer", "args": []interface{}{"0xdef", 100},
	}}
	resp := &fakeResp{}
	r.rpcProcessContractCall(nil, req, resp)
	if !resp.hasError {
		t.Fatal("args not yet supported must error")
	}
	if !strings.Contains(resp.errMsg, "not yet supported") {
		t.Fatalf("expected args error, got: %s", resp.errMsg)
	}
}
