package subscriptions

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"strings"
)

type TransferNotification struct {
	ChainId       string   `json:"chainId"`
	TxID          string   `json:"tx_id"`
	Timestamp     int64    `json:"timestamp"`
	BlockNum      int      `json:"blockNum"`
	Success       bool     `json:"success"`
	Transfer      bool     `json:"transfer"`
	NativeCoin    bool     `json:"nativeCoin,omitempty"`
	Symbol        string   `json:"symbol,omitempty"`
	SmartContract bool     `json:"smartContract,omitempty"`
	From          string   `json:"from"`
	To            string   `json:"to"`
	Amount        *big.Int `json:"amount"`
	Token         string   `json:"token,omitempty"`
	TokenSymbol   string   `json:"tokenSymbol,omitempty"`
	Fee           *big.Int `json:"fee"`
	InPool        bool     `json:"inPool"`
	Confirmed     bool     `json:"confirmed"`
	Confirmations int      `json:"confirmations"`
	UserId        int64    `json:"userId,omitempty"`
	InvoiceId     int64    `json:"invoiceId,omitempty"`
	Signature     string   `json:"sign,omitempty"`
}

func (n *TransferNotification) Sign(apiKey string) {
	bodyParts := []string{
		n.TxID,
		fmt.Sprintf("%d", n.Timestamp),
		fmt.Sprintf("%d", n.BlockNum),
		fmt.Sprintf("%t", n.Success),
		fmt.Sprintf("%t", n.NativeCoin),
		n.Symbol,
		n.From,
		n.To,
		n.Amount.String(),
		n.Token,
		n.TokenSymbol,
		n.Fee.String(),
		fmt.Sprintf("%t", n.InPool),
		fmt.Sprintf("%t", n.Confirmed),
		fmt.Sprintf("%d", n.Confirmations),
		apiKey,
	}
	n.Signature = fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(bodyParts, ":"))))
}
func (n *TransferNotification) fill(tx *TransferInfoRecord) *TransferNotification {
	n.TxID = tx.TxID
	n.Timestamp = tx.Timestamp
	n.BlockNum = tx.BlockNum
	n.Success = tx.Success
	n.Transfer = tx.Transfer
	n.NativeCoin = tx.NativeCoin
	n.Symbol = tx.Symbol
	n.SmartContract = tx.SmartContract
	n.From = tx.From
	n.To = tx.To
	n.Amount = tx.Amount
	n.Token = tx.Token
	n.TokenSymbol = tx.TokenSymbol
	n.Fee = tx.Fee
	n.InPool = tx.InPool
	n.Confirmed = tx.Confirmed
	return n
}
