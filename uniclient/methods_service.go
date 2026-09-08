package uniclient

// ServiceConfig is the client's configured service's webhook and event
// report settings, as both the request to ServiceConfig and its result (the
// server echoes back what it stored).
type ServiceConfig struct {
	EventUrl         string   `json:"eventUrl"`
	ReportNewBlock   bool     `json:"reportNewBlock,omitempty"`
	ReportIncomingTx bool     `json:"reportIncomingTx,omitempty"`
	ReportOutgoingTx bool     `json:"reportOutgoingTx,omitempty"`
	ReportMainCoin   bool     `json:"reportMainCoin,omitempty"`
	ReportTokens     []string `json:"reportTokens,omitempty"`
	GatherToMaster   bool     `json:"gatherToMaster,omitempty"`
	MasterList       []string `json:"masterList,omitempty"`
}

// ServiceConfig sets the client's configured service's (WithServiceId)
// webhook URL and which events it wants reported to it. Secured.
//
// serviceRegister -- the call that would create a new service in the first
// place -- is not implemented server-side yet (see todo/TASKS.md), so
// ServiceConfig only edits a service that already exists.
func (c *Client) ServiceConfig(cfg ServiceConfig) (result *ServiceConfig, err error) {
	type serviceConfigRequest struct {
		ServiceID int `json:"serviceId"`
		ServiceConfig
	}
	request := NewRequest("serviceConfig", &serviceConfigRequest{ServiceID: c.serviceId, ServiceConfig: cfg})
	result = new(ServiceConfig)
	if err = c.rpcCall(request, result); err != nil {
		return nil, err
	}
	return result, nil
}
