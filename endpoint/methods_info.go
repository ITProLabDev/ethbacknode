package endpoint

import (
	"github.com/ITProLabDev/ethbacknode/types"
	"time"
)

func (r *BackRpc) rpcProcessPing(ctx RequestContext, request RpcRequest, response RpcResponse) {
	result := &struct {
		Result    string `json:"result"`
		Timestamp int64  `json:"timestamp"`
	}{
		Result:    "pong",
		Timestamp: time.Now().UnixNano(),
	}
	response.SetResult(result)
}
func (r *BackRpc) rpcProcessNodeInfo(ctx RequestContext, request RpcRequest, response RpcResponse) {
	info := &struct {
		Name           string             `json:"blockchain"`
		Id             string             `json:"id"`
		Symbol         string             `json:"symbol"`
		Decimals       int                `json:"decimals"`
		TokenProtocols []string           `json:"protocols,omitempty"`
		Tokens         []*types.TokenInfo `json:"tokens,omitempty"`
	}{
		Name:           r.chainClient.GetChainName(),
		Id:             r.chainClient.GetChainId(),
		Symbol:         r.chainClient.GetChainSymbol(),
		Decimals:       r.chainClient.Decimals(),
		TokenProtocols: r.chainClient.TokenProtocols(),
		Tokens:         r.chainClient.TokensList(),
	}
	response.SetResult(info)
}

func (r *BackRpc) rpcProcessInfoGetTokenList(ctx RequestContext, request RpcRequest, response RpcResponse) {
	type TokenResponseRow struct {
		Name            string `json:"name"`
		Symbol          string `json:"symbol"`
		Decimals        int    `json:"decimals"`
		Token           bool   `json:"token,omitempty"`
		ContractAddress string `json:"contractAddress"`
	}
	var tokenList = make([]*TokenResponseRow, 1)
	tokenListInternal := r.chainClient.TokensList()
	tokenList[0] = &TokenResponseRow{
		Name:            r.chainClient.GetChainName(),
		Symbol:          r.chainClient.GetChainSymbol(),
		Decimals:        r.chainClient.Decimals(),
		ContractAddress: "",
	}
	tokenList = append(tokenList)
	for _, token := range tokenListInternal {
		tokenList = append(tokenList, &TokenResponseRow{
			Name:            token.Name,
			Symbol:          token.Symbol,
			Decimals:        token.Decimals,
			Token:           true,
			ContractAddress: token.ContractAddress,
		})
	}
	response.SetResult(tokenList)
}
