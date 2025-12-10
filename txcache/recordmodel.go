package txcache

import (
	"math/big"
	"module github.com/ITProLabDev/ethbacknode/types"
)

type TransferInfoCachedRecord struct {
	TxID              string   `json:"txId" badgerhold:"key"`
	Timestamp         int64    `json:"timestamp"`
	BlockNum          int      `json:"blockNum"`
	Success           bool     `json:"success"`
	Transfer          bool     `json:"transfer"`
	NativeCoin        bool     `json:"nativeCoin,omitempty"`
	Symbol            string   `json:"symbol,omitempty"`
	SmartContract     bool     `json:"smartContract,omitempty"`
	From              string   `json:"from" storm:"index"`
	To                string   `json:"to" storm:"index"`
	Fee               *big.Int `json:"fee"`
	Amount            *big.Int `json:"amount"`
	Token             string   `json:"token,omitempty"`
	TokenSymbol       string   `json:"tokenSymbol,omitempty"`
	InPool            bool     `json:"inPool"`
	Confirmed         bool     `json:"confirmed"`
	Confirmations     int      `json:"confirmations"`
	Decimals          int      `json:"decimals"`
	ChainSpecificData []byte   `json:"chainSpecificData,omitempty"`
}

func (r *TransferInfoCachedRecord) loadFromTransferInfo(info *types.TransferInfo) {
	r.TxID = info.TxID
	r.Timestamp = info.Timestamp
	r.BlockNum = info.BlockNum
	r.Success = info.Success
	r.Transfer = info.Transfer
	r.NativeCoin = info.NativeCoin
	r.Symbol = info.Symbol
	r.SmartContract = info.SmartContract
	r.From = info.From
	r.To = info.To
	r.Token = info.Token
	r.TokenSymbol = info.TokenSymbol
	r.InPool = info.InPool
	r.Confirmed = info.Confirmed
	r.Confirmations = info.Confirmations
	r.Decimals = info.Decimals
	r.ChainSpecificData = info.ChainSpecificData

	r.Fee = new(big.Int).Set(info.Fee)
	r.Amount = new(big.Int).Set(info.Amount)
}

func (r *TransferInfoCachedRecord) getTransferInfo() *types.TransferInfo {
	return &types.TransferInfo{
		TxID:              r.TxID,
		Timestamp:         r.Timestamp,
		BlockNum:          r.BlockNum,
		Success:           r.Success,
		Transfer:          r.Transfer,
		NativeCoin:        r.NativeCoin,
		Symbol:            r.Symbol,
		SmartContract:     r.SmartContract,
		From:              r.From,
		Token:             r.Token,
		TokenSymbol:       r.TokenSymbol,
		InPool:            r.InPool,
		Confirmed:         r.Confirmed,
		Confirmations:     r.Confirmations,
		Decimals:          r.Decimals,
		ChainSpecificData: r.ChainSpecificData,
		To:                r.To,
		Fee:               new(big.Int).Set(r.Fee),
		Amount:            new(big.Int).Set(r.Amount),
	}
}
