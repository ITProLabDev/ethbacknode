package uniclient

import "encoding/json"

// TransferInfo contains details about a blockchain transfer.
type TransferInfo struct {
	TxID              string      `json:"tx_id"`
	Timestamp         int64       `json:"timestamp"`
	BlockNum          int         `json:"blockNum"`
	Success           bool        `json:"success"`
	Transfer          bool        `json:"transfer"`
	NativeCoin        bool        `json:"nativeCoin,omitempty"`
	Symbol            string      `json:"symbol,omitempty"`
	Decimals          int         `json:"decimals"`
	SmartContract     bool        `json:"smartContract,omitempty"`
	From              string      `json:"from"`
	To                string      `json:"to"`
	Amount            json.Number `json:"amount"`
	Token             string      `json:"token,omitempty"`
	TokenSymbol       string      `json:"tokenSymbol,omitempty"`
	Fee               json.Number `json:"fee"`
	InPool            bool        `json:"inPool"`
	Confirmed         bool        `json:"confirmed"`
	Confirmations     int         `json:"confirmations,omitempty"`
	ChainSpecificData []byte      `json:"chainSpecificData,omitempty"`
}

// TransferInfo retrieves details about a specific transaction by ID.
func (c *Client) TransferInfo(txID string) (transferResult *TransferInfo, err error) {
	request := NewRequest("transferInfo", map[string]interface{}{
		"txId":           txID,
		"amountFormated": true,
	})
	transferResponse := new(TransferInfo)
	err = c.rpcCall(request, transferResponse)
	if err != nil {
		return nil, err
	}
	return transferResponse, nil
}

// TransfersByAddress retrieves all transfers involving the given address.
func (c *Client) TransfersByAddress(address string) (transfersList []*TransferInfo, err error) {
	request := NewRequest("transferInfoForAddress", map[string]interface{}{
		"address":        address,
		"amountFormated": true,
	})
	err = c.rpcCall(request, &transfersList)
	if err != nil {
		return nil, err
	}
	return transfersList, nil
}
