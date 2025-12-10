package endpoint

import (
	"backnode/tools/log"
	"backnode/types"
	"math/big"
	"strings"
)

func (r *BackRpc) rpcProcessGetBalance(ctx RequestContext, request RpcRequest, response RpcResponse) {
	type addressBalanceRequest struct {
		Address   string `json:"address"`
		Assets    string `json:"assets"`
		AllAssets bool   `json:"allAssets"`
		Formatted bool   `json:"formatted"`
		Extended  bool   `json:"extended"`
	}
	//TODO add extended response
	type balanceExtendedResponse struct {
		Symbol   string   `json:"symbol"`
		Amount   *big.Int `json:"amount"`
		Decimals int      `json:"decimals"`
	}
	params := &addressBalanceRequest{
		AllAssets: true,
		Formatted: true,
	}
	err := request.ParseParams(params)
	if err != nil {
		response.SetError(ERROR_CODE_PARSE_ERROR, ERROR_MESSAGE_PARSE_ERROR)
		return
	}
	format := func(balance *big.Int, decimals int) amount {
		if params.Formatted {
			str := balance.String()
			if len(str) > decimals {
				return amount(str[:len(str)-decimals] + "." + str[len(str)-decimals:])
			} else {
				return amount("0." + strings.Repeat("0", decimals-len(str)) + str)
			}
		}
		return amount(balance.String())
	}
	if !r.addressCodec.IsValid(params.Address) {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "Invalid address")
		return
	}
	result := make(map[string]interface{})
	if params.Assets != "" {
		params.AllAssets = false
	}
	if params.Assets == "" && !params.AllAssets {
		log.Debug("Get Native coin balance")
		balance, err := r.chainClient.BalanceOf(params.Address)
		if err != nil {
			log.Error("Error getting balance: ", err)
			response.SetError(ERROR_CODE_SERVER_ERROR, err.Error())
			return
		}
		d := r.chainClient.Decimals()
		result[r.chainClient.GetChainSymbol()] = format(balance, d)
		response.SetResult(result)
		return
	}
	var assetList []string
	var knownAssets = []string{r.chainClient.GetChainSymbol()}
	var tokenMap = make(map[string]*types.TokenInfo)
	for _, token := range r.chainClient.TokensList() {
		if token.Symbol != "" {
			knownAssets = append(knownAssets, token.Symbol)
			tokenMap[token.Symbol] = token
		}
	}
	if params.AllAssets {
		assetList = knownAssets
	} else if params.Assets != "" {
		assetList = strings.Split(strings.TrimSpace(strings.ToUpper(params.Assets)), ",")
		for i, asset := range assetList {
			assetList[i] = strings.TrimSpace(asset)
		}
		for _, asset := range assetList {
			if !_contains(knownAssets, asset) {
				response.SetError(ERROR_CODE_INVALID_REQUEST, "Unknown asset: "+asset)
				return
			}
		}
	}
	for _, asset := range assetList {
		var balance *big.Int
		var decimals int
		if asset == r.chainClient.GetChainSymbol() {
			balance, err = r.chainClient.BalanceOf(params.Address)
			decimals = r.chainClient.Decimals()
		} else {
			balance, err = r.chainClient.TokensBalanceOf(params.Address, asset)
			decimals = tokenMap[asset].Decimals
		}
		if err != nil {
			log.Error("Error getting balance: ", err)
			response.SetError(ERROR_CODE_SERVER_ERROR, ERROR_MESSAGE_SERVER_ERROR)
			return
		}
		result[asset] = format(balance, decimals)
	}
	response.SetResult(result)
}

type amount string

func (a amount) MarshalJSON() ([]byte, error) {
	return []byte(a), nil
}
