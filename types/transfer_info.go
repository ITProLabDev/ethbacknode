package types

import (
	"math/big"
)

type TransferInfo struct {
	TxID              string   `json:"txId"`
	Timestamp         int64    `json:"timestamp"`
	BlockNum          int      `json:"blockNum"`
	Success           bool     `json:"success"`
	Transfer          bool     `json:"transfer"`
	NativeCoin        bool     `json:"nativeCoin,omitempty"`
	Symbol            string   `json:"symbol,omitempty"`
	SmartContract     bool     `json:"smartContract,omitempty"`
	From              string   `json:"from"`
	To                string   `json:"to"`
	Amount            *big.Int `json:"amount"`
	Token             string   `json:"token,omitempty"`
	TokenSymbol       string   `json:"tokenSymbol,omitempty"`
	Fee               *big.Int `json:"fee"`
	InPool            bool     `json:"inPool"`
	Confirmed         bool     `json:"confirmed"`
	Confirmations     int      `json:"confirmations"`
	Decimals          int      `json:"decimals"`
	ChainSpecificData []byte   `json:"chainSpecificData,omitempty"`
}

type DataDecoder func([]byte) error

func (t *TransferInfo) DecodeChainSpecificData(decoder DataDecoder) error {
	return decoder(t.ChainSpecificData)
}
