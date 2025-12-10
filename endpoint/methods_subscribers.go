package endpoint

import (
	"module github.com/ITProLabDev/ethbacknode/subscriptions"
	"module github.com/ITProLabDev/ethbacknode/tools/log"
)

func (r *BackRpc) rpcProcessServiceRegister(ctx RequestContext, request RpcRequest, response RpcResponse) {
	panic("Not implemented")
}

func (r *BackRpc) rpcProcessServiceConfig(ctx RequestContext, request RpcRequest, response RpcResponse) {
	type serviceConfigRequest struct {
		ServiceId        subscriptions.ServiceId `json:"serviceId"`
		ApiToken         string                  `json:"apiToken,omitempty"`
		EndpointUrl      string                  `json:"eventUrl"`
		ReportNewBlock   bool                    `json:"reportNewBlock"`
		ReportIncomingTx bool                    `json:"reportIncomingTx"`
		ReportOutgoingTx bool                    `json:"reportOutgoingTx"`
		ReportMainCoin   bool                    `json:"reportMainCoin"`
		ReportTokens     []string                `json:"reportTokens"`
		GatherToMaster   bool                    `json:"gatherToMaster"`
		MasterList       []string                `json:"masterList"`
	}
	params := &serviceConfigRequest{
		ReportMainCoin: true,
	}
	err := request.ParseParams(params)
	if err != nil {
		response.SetError(ERROR_CODE_PARSE_ERROR, ERROR_MESSAGE_PARSE_ERROR)
		return
	}
	subscription, err := r.subscriptions.SubscriptionGet(params.ServiceId)
	if err != nil {
		response.SetError(ERROR_CODE_SERVER_ERROR, err.Error())
		return
	}
	if subscription.Internal {
		response.SetError(ERROR_CODE_SERVER_ERROR, "unknown serviceId")
		return
	}
	if subscription.ApiToken != "" || subscription.ApiKey != "" {
		//TODO: check authorization
		log.Warning("TODO: Authorization needed")
	}
	tokenListInternal := r.chainClient.TokensList()
	//TODO validate url
	//TODO validate master address
	//TODO validate tokens

	err = r.subscriptions.SubscriptionEdit(params.ServiceId, func(subscription *subscriptions.Subscription) {
		subscription.EndpointUrl = params.EndpointUrl
		subscription.ReportNewBlock = params.ReportNewBlock
		subscription.ReportIncomingTx = params.ReportIncomingTx
		subscription.ReportOutgoingTx = params.ReportOutgoingTx
		subscription.ReportMainCoin = params.ReportMainCoin
		subscription.ReportTokens = make(map[string]bool)
		for _, token := range tokenListInternal {
			subscription.ReportTokens[token.Symbol] = false
		}
		for _, token := range params.ReportTokens {
			subscription.ReportTokens[token] = true
		}
		subscription.GatherToMaster = params.GatherToMaster
		subscription.MasterList = params.MasterList
	})
	if err != nil {
		response.SetError(ERROR_CODE_SERVER_ERROR, err.Error())
		return
	}
	//log.Dump(params)
	params.ApiToken = ""
	response.SetResult(params)
}
