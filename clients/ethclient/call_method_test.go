package ethclient

import (
	"math/big"
	"testing"

	"github.com/ITProLabDev/ethbacknode/abi"
)

// newCallClient wires a fake transport (canned eth_call hex result) and a real
// abi manager holding one contract, so CallMethod can be exercised offline.
func newCallClient(t *testing.T, contractAddr, rawABI, callResultHex string) *Client {
	t.Helper()
	mgr := abi.NewManager(
		abi.WithStorage(newMemABIStorage()),
		abi.WithAddressCodec(GetAddressCodec()),
	)
	if err := mgr.Init(); err != nil {
		t.Fatal(err)
	}
	c, err := abi.NewContractFromABI("C", "C", contractAddr, []byte(rawABI))
	if err != nil {
		t.Fatal(err)
	}
	mgr.Add(c)

	ft := &fakeTransport{result: jsonString(callResultHex)}
	return &Client{
		rpcClient:    urpcClientWith(ft),
		abi:          mgr,
		addressCodec: GetAddressCodec(),
	}
}

func TestCallMethod_DecodesUintResult(t *testing.T) {
	rawABI := `[{"type":"function","name":"balanceOf","stateMutability":"view",
	  "inputs":[{"name":"a","type":"address"}],"outputs":[{"name":"","type":"uint256"}]}]`
	resultHex := "0x00000000000000000000000000000000000000000000000000000000000003e8"
	c := newCallClient(t, "0xabc0000000000000000000000000000000000000", rawABI, resultHex)

	addrBytes, _ := GetAddressCodec().DecodeAddressToBytes("0x1111111111111111111111111111111111111111")
	out, err := c.CallMethod("0xabc0000000000000000000000000000000000000", "balanceOf", addrBytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Value.(*big.Int).Int64() != 1000 {
		t.Fatalf("call output=%+v", out)
	}
}

func TestCallMethod_UnknownContract(t *testing.T) {
	rawABI := `[{"type":"function","name":"x","inputs":[],"outputs":[]}]`
	c := newCallClient(t, "0xabc0000000000000000000000000000000000000", rawABI, "0x")
	if _, err := c.CallMethod("0x9999999999999999999999999999999999999999", "x"); err == nil {
		t.Fatal("unknown contract must error")
	}
}

func TestCallMethod_UnknownMethod(t *testing.T) {
	rawABI := `[{"type":"function","name":"x","inputs":[],"outputs":[]}]`
	c := newCallClient(t, "0xabc0000000000000000000000000000000000000", rawABI, "0x")
	if _, err := c.CallMethod("0xabc0000000000000000000000000000000000000", "nope"); err == nil {
		t.Fatal("unknown method must error")
	}
}
