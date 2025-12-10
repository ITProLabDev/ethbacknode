package endpoint

import (
	"math/big"
	"module github.com/ITProLabDev/ethbacknode/tools/log"
	"module github.com/ITProLabDev/ethbacknode/types"
	"strings"
)

func (r *BackRpc) rpcProcessGetTransferInfo(ctx RequestContext, request RpcRequest, response RpcResponse) {
	type transferInfoRequest struct {
		TxId             string `json:"txId"`
		AmountsFormatted bool   `json:"amountsFormatted,omitempty"`
	}
	params := &transferInfoRequest{
		AmountsFormatted: true,
	}
	err := request.ParseParams(params)
	if err != nil {
		response.SetError(ERROR_CODE_PARSE_ERROR, ERROR_MESSAGE_PARSE_ERROR)
		return
	}
	format := func(balance *big.Int, decimals int) amount {
		if params.AmountsFormatted {
			str := balance.String()
			if len(str) > decimals {
				return amount(str[:len(str)-decimals] + "." + str[len(str)-decimals:])
			} else {
				return amount("0." + strings.Repeat("0", decimals-len(str)) + str)
			}
		}
		return amount(balance.String())
	}
	result := &TransferInfoResponse{}
	transferInfo, err := r.txCache.GetTransferInfo(params.TxId)
	if err != nil {
		if r.debugMode {
			log.Error("Can not get transaction info: ", err)
		}
		response.SetError(ERROR_CODE_SERVER_ERROR, "unknown or unsupported transaction")
		return
	}
	result.fill(transferInfo)
	result.Amount = format(transferInfo.Amount, transferInfo.Decimals)
	result.Fee = format(transferInfo.Fee, transferInfo.Decimals)
	response.SetResult(result)
}

func (r *BackRpc) rpcProcessGetTransfersForAddress(ctx RequestContext, request RpcRequest, response RpcResponse) {
	type transferInfoRequest struct {
		Address          string `json:"address"`
		AmountsFormatted bool   `json:"amountsFormatted,omitempty"`
	}
	params := &transferInfoRequest{
		AmountsFormatted: true,
	}
	err := request.ParseParams(params)
	if err != nil {
		response.SetError(ERROR_CODE_PARSE_ERROR, ERROR_MESSAGE_PARSE_ERROR)
		return
	}
	address, err := r.addressNormalise(params.Address)
	if err != nil {
		response.SetError(ERROR_CODE_INVALID_REQUEST, err.Error())
		return
	}
	format := func(balance *big.Int, decimals int) amount {
		if params.AmountsFormatted {
			str := balance.String()
			if len(str) > decimals {
				return amount(str[:len(str)-decimals] + "." + str[len(str)-decimals:])
			} else {
				return amount("0." + strings.Repeat("0", decimals-len(str)) + str)
			}
		}
		return amount(balance.String())
	}
	var result []*TransferInfoResponse

	txList, err := r.txCache.GetTransfersByAddress(address)
	if err != nil {
		log.Error("Can not get transactions list: ", err)
		response.SetError(ERROR_CODE_SERVER_ERROR, "server error")
		return

	}
	result = make([]*TransferInfoResponse, len(txList))
	for i, transactionInfo := range txList {
		row := &TransferInfoResponse{}
		row.fill(transactionInfo)
		row.Amount = format(transactionInfo.Amount, transactionInfo.Decimals)
		row.Fee = format(transactionInfo.Fee, transactionInfo.Decimals)
		result[i] = row
	}
	response.SetResult(result)
}

type TransferInfoResponse struct {
	TxID              string `json:"tx_id"`
	Timestamp         int64  `json:"timestamp"`
	BlockNum          int    `json:"blockNum"`
	Success           bool   `json:"success"`
	Transfer          bool   `json:"transfer"`
	NativeCoin        bool   `json:"nativeCoin,omitempty"`
	Symbol            string `json:"symbol,omitempty"`
	Decimals          int    `json:"decimals"`
	SmartContract     bool   `json:"smartContract,omitempty"`
	From              string `json:"from"`
	To                string `json:"to"`
	Amount            amount `json:"amount"`
	Token             string `json:"token,omitempty"`
	TokenSymbol       string `json:"tokenSymbol,omitempty"`
	Fee               amount `json:"fee"`
	InPool            bool   `json:"inPool"`
	Confirmed         bool   `json:"confirmed"`
	Confirmations     int    `json:"confirmations,omitempty"`
	ChainSpecificData []byte `json:"chainSpecificData,omitempty"`
}

func (r *TransferInfoResponse) fill(ti *types.TransferInfo) {
	r.TxID = ti.TxID
	r.Timestamp = ti.Timestamp
	r.BlockNum = ti.BlockNum
	r.Success = ti.Success
	r.Transfer = ti.Transfer
	r.NativeCoin = ti.NativeCoin
	r.Symbol = ti.Symbol
	r.SmartContract = ti.SmartContract
	r.From = ti.From
	r.To = ti.To
	r.Amount = amount(ti.Amount.String())
	r.Token = ti.Token
	r.TokenSymbol = ti.TokenSymbol
	r.Fee = amount(ti.Fee.String())
	r.InPool = ti.InPool
	r.Confirmed = ti.Confirmed
	r.Confirmations = ti.Confirmations
	r.Decimals = ti.Decimals
	r.ChainSpecificData = ti.ChainSpecificData
}
