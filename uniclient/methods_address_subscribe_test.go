package uniclient

import "testing"

func TestAddressSubscribe_WatchOnly_SendsServiceIdAndAddress(t *testing.T) {
	c, ft := newFakeClient(map[string]interface{}{"success": true})

	result, err := c.AddressSubscribe("0xabc", "", nil, 1, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	if ft.lastRequest.Method != "addressSubscribe" {
		t.Fatalf("method=%q", ft.lastRequest.Method)
	}
	var params struct {
		Address    string   `json:"address"`
		PrivateKey string   `json:"privateKey"`
		Mnemonic   []string `json:"mnemonic"`
		ServiceID  int      `json:"serviceId"`
		UserId     int64    `json:"userId"`
		WatchOnly  bool     `json:"watchOnly"`
	}
	lastParams(t, ft, &params)
	if params.Address != "0xabc" || params.ServiceID != 42 || params.UserId != 1 || !params.WatchOnly {
		t.Fatalf("params=%+v", params)
	}
	if params.PrivateKey != "" || len(params.Mnemonic) != 0 {
		t.Fatalf("expected no privateKey/mnemonic sent for a watch-only subscribe, got %+v", params)
	}
	if !result.Success {
		t.Fatalf("result=%+v", result)
	}
}

func TestAddressSubscribe_WithPrivateKey(t *testing.T) {
	c, ft := newFakeClient(map[string]interface{}{"success": true})
	if _, err := c.AddressSubscribe("", "0xdeadbeef", nil, 1, 2, false); err != nil {
		t.Fatal(err)
	}
	var params struct {
		Address    string `json:"address"`
		PrivateKey string `json:"privateKey"`
	}
	lastParams(t, ft, &params)
	if params.Address != "" || params.PrivateKey != "0xdeadbeef" {
		t.Fatalf("params=%+v", params)
	}
}

func TestAddressSubscribe_SurfacesFailureInResultNotError(t *testing.T) {
	// The server reports address-subscribe failures inside the result
	// (success/error fields), not as a JSON-RPC error -- mirror that.
	c, _ := newFakeClient(map[string]interface{}{"success": false, "error": "Invalid address"})
	result, err := c.AddressSubscribe("bad", "", nil, 1, 0, true)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.Success || result.Error != "Invalid address" {
		t.Fatalf("result=%+v", result)
	}
}

func TestAddressRecover_ParsesResult(t *testing.T) {
	c, ft := newFakeClient(map[string]interface{}{
		"success": true, "address": "0xabc", "privateKey": "0xdead",
		"bip39Mnemonic": []string{"a", "b"},
	})

	result, err := c.AddressRecover([]string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if ft.lastRequest.Method != "addressRecover" {
		t.Fatalf("method=%q", ft.lastRequest.Method)
	}
	var params struct {
		Mnemonic []string `json:"mnemonic"`
	}
	lastParams(t, ft, &params)
	if len(params.Mnemonic) != 2 {
		t.Fatalf("params=%+v", params)
	}
	if !result.Success || result.Address != "0xabc" || len(result.Bip39Mnemonic) != 2 {
		t.Fatalf("result=%+v", result)
	}
}

func TestAddressRecover_SurfacesFailureInResultNotError(t *testing.T) {
	c, _ := newFakeClient(map[string]interface{}{"success": false, "error": "Invalid mnemonic"})
	result, err := c.AddressRecover([]string{"bad"})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.Success || result.Error != "Invalid mnemonic" {
		t.Fatalf("result=%+v", result)
	}
}

func TestAddressGenerate_DefaultsMnemonicLenTo12(t *testing.T) {
	c, ft := newFakeClient(map[string]interface{}{"success": true, "address": "0xabc"})

	result, err := c.AddressGenerate(0)
	if err != nil {
		t.Fatal(err)
	}
	if ft.lastRequest.Method != "addressGenerate" {
		t.Fatalf("method=%q", ft.lastRequest.Method)
	}
	var params struct {
		MnemonicLen int `json:"mnemonicLen"`
	}
	lastParams(t, ft, &params)
	if params.MnemonicLen != 12 {
		t.Fatalf("mnemonicLen=%d want 12 (default)", params.MnemonicLen)
	}
	if !result.Success || result.Address != "0xabc" {
		t.Fatalf("result=%+v", result)
	}
}

func TestAddressGenerate_ExplicitMnemonicLen(t *testing.T) {
	c, ft := newFakeClient(map[string]interface{}{"success": true})
	if _, err := c.AddressGenerate(24); err != nil {
		t.Fatal(err)
	}
	var params struct {
		MnemonicLen int `json:"mnemonicLen"`
	}
	lastParams(t, ft, &params)
	if params.MnemonicLen != 24 {
		t.Fatalf("mnemonicLen=%d want 24", params.MnemonicLen)
	}
}
