package uniclient

import "testing"

func TestServiceConfig_SendsServiceIdAndFlags(t *testing.T) {
	c, ft := newFakeClient(map[string]interface{}{
		"eventUrl": "https://example.com/hook", "reportIncomingTx": true,
	})

	cfg := ServiceConfig{
		EventUrl:         "https://example.com/hook",
		ReportIncomingTx: true,
		ReportOutgoingTx: true,
		ReportMainCoin:   true,
		ReportTokens:     []string{"USDT"},
	}
	result, err := c.ServiceConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if ft.lastRequest.Method != "serviceConfig" {
		t.Fatalf("method=%q", ft.lastRequest.Method)
	}
	var params struct {
		ServiceID        int      `json:"serviceId"`
		EventUrl         string   `json:"eventUrl"`
		ReportIncomingTx bool     `json:"reportIncomingTx"`
		ReportOutgoingTx bool     `json:"reportOutgoingTx"`
		ReportTokens     []string `json:"reportTokens"`
	}
	lastParams(t, ft, &params)
	if params.ServiceID != 42 || params.EventUrl != cfg.EventUrl {
		t.Fatalf("params=%+v", params)
	}
	if !params.ReportIncomingTx || !params.ReportOutgoingTx {
		t.Fatalf("report flags not sent: %+v", params)
	}
	if len(params.ReportTokens) != 1 || params.ReportTokens[0] != "USDT" {
		t.Fatalf("reportTokens=%v", params.ReportTokens)
	}
	if result.EventUrl != "https://example.com/hook" || !result.ReportIncomingTx {
		t.Fatalf("result=%+v", result)
	}
}
