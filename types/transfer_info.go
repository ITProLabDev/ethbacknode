package types

import (
	"math/big"
)

// TransferInfo represents a blockchain transaction/transfer with all relevant details.
// It supports both native coin transfers and ERC-20 token transfers.
type TransferInfo struct {
	// TxID is the transaction hash (e.g., 0x...).
	TxID string `json:"txId"`
	// Timestamp is the Unix timestamp of the transaction.
	Timestamp int64 `json:"timestamp"`
	// BlockNum is the block number containing this transaction (-1 if pending).
	BlockNum int `json:"blockNum"`
	// Success indicates whether the transaction was successful.
	Success bool `json:"success"`
	// Transfer indicates if this is a value transfer (not just a contract call).
	Transfer bool `json:"transfer"`
	// NativeCoin is true if this is a native coin transfer (ETH, not token).
	NativeCoin bool `json:"nativeCoin,omitempty"`
	// Symbol is the currency symbol (e.g., "ETH").
	Symbol string `json:"symbol,omitempty"`
	// SmartContract is true if the transaction involves a smart contract.
	SmartContract bool `json:"smartContract,omitempty"`
	// From is the sender address.
	From string `json:"from"`
	// To is the recipient address.
	To string `json:"to"`
	// Amount is the transfer amount in the smallest unit (wei for ETH).
	Amount *big.Int `json:"amount"`
	// Token is the token contract address (for token transfers).
	Token string `json:"token,omitempty"`
	// TokenSymbol is the token symbol (e.g., "USDT").
	TokenSymbol string `json:"tokenSymbol,omitempty"`
	// Fee is the transaction fee in the smallest unit.
	Fee *big.Int `json:"fee"`
	// InPool is true if the transaction is still in the mempool.
	InPool bool `json:"inPool"`
	// Confirmed is true if the transaction has enough confirmations.
	Confirmed bool `json:"confirmed"`
	// Confirmations is the number of block confirmations.
	Confirmations int `json:"confirmations"`
	// Decimals is the decimal precision for the currency/token.
	Decimals int `json:"decimals"`
	// ChainSpecificData holds chain-specific data in encoded form.
	ChainSpecificData []byte `json:"chainSpecificData,omitempty"`
}

// DataDecoder is a function type for decoding chain-specific data.
type DataDecoder func([]byte) error

// DecodeChainSpecificData decodes the chain-specific data using the provided decoder.
// This allows chain-specific data to be decoded without the types package
// needing to know about chain-specific structures.
func (t *TransferInfo) DecodeChainSpecificData(decoder DataDecoder) error {
	return decoder(t.ChainSpecificData)
}
