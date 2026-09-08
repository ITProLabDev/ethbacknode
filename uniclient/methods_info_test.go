package uniclient

import "testing"

func TestPing_ParsesResult(t *testing.T) {
	c, ft := newFakeClient(map[string]interface{}{"result": "pong", "timestamp": 1718789894210866000})

	result, err := c.Ping()
	if err != nil {
		t.Fatal(err)
	}
	if ft.lastRequest.Method != "ping" {
		t.Fatalf("method=%q", ft.lastRequest.Method)
	}
	if result.Result != "pong" || result.Timestamp != 1718789894210866000 {
		t.Fatalf("result=%+v", result)
	}
}

func TestInfoGetTokenList_ParsesEntries(t *testing.T) {
	c, ft := newFakeClient([]map[string]interface{}{
		{"name": "Ethereum", "symbol": "ETH", "decimals": 18, "contractAddress": ""},
		{"name": "Tether", "symbol": "USDT", "decimals": 6, "token": true, "contractAddress": "0xabc"},
	})

	result, err := c.InfoGetTokenList()
	if err != nil {
		t.Fatal(err)
	}
	if ft.lastRequest.Method != "infoGetTokenList" {
		t.Fatalf("method=%q", ft.lastRequest.Method)
	}
	if len(result) != 2 {
		t.Fatalf("result=%+v", result)
	}
	if result[0].Symbol != "ETH" || result[0].Token {
		t.Fatalf("native entry=%+v", result[0])
	}
	if result[1].Symbol != "USDT" || !result[1].Token || result[1].ContractAddress != "0xabc" {
		t.Fatalf("token entry=%+v", result[1])
	}
}
