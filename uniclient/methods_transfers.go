package uniclient

import "encoding/json"

// TransferAssetsResult contains the result of an asset transfer operation.
type TransferAssetsResult struct {
	TxID              string      `json:"tx_id"`
	Success           bool        `json:"success"`
	NativeCoin        bool        `json:"nativeCoin,omitempty"`
	SmartContract     bool        `json:"smartContract,omitempty"`
	Symbol            string      `json:"symbol,omitempty"`
	From              string      `json:"from"`
	To                string      `json:"to"`
	Amount            json.Number `json:"amount"`
	Fee               json.Number `json:"fee"`
	FeeSymbol         string      `json:"feeSymbol,omitempty"`
	Warning           string      `json:"warning,omitempty"`
	ChainSpecificData []byte      `json:"chainSpecificData,omitempty"`
}

// transferAssetsRequest is the request body for asset transfer operations.
type transferAssetsRequest struct {
	ServiceID      int         `json:"serviceId,omitempty"`
	PrivateKey     string      `json:"privateKey,omitempty"`
	From           string      `json:"from,omitempty"`
	To             string      `json:"to"`
	Amount         json.Number `json:"amount"`
	Symbol         string      `json:"symbol,omitempty"`
	Force          bool        `json:"force,omitempty"`
	Signature      string      `json:"signature,omitempty"`
	AmountFormated bool        `json:"amountFormated,omitempty"`
}

// TransferAsset initiates an asset transfer from one address to another.
func (c *Client) TransferAsset(addressFrom, addressTo string, amount json.Number, symbol string, formated bool) (transferResult *TransferAssetsResult, err error) {
	request := NewRequest("transferAssets", &transferAssetsRequest{
		From:           addressFrom,
		To:             addressTo,
		Amount:         amount,
		Symbol:         symbol,
		AmountFormated: formated,
	})
	transferResponse := new(TransferAssetsResult)
	err = c.rpcCall(request, transferResponse)
	if err != nil {
		return nil, err
	}
	return transferResponse, nil
}

// TransferGetEstimatedFee estimates the fee for a transfer operation.
func (c *Client) TransferGetEstimatedFee(addressFrom, addressTo string, amount json.Number, symbol string, formated bool) (estimatedFee json.Number, err error) {
	request := NewRequest("transferGetEstimatedFee", &transferAssetsRequest{
		From:           addressFrom,
		To:             addressTo,
		Amount:         amount,
		Symbol:         symbol,
		AmountFormated: formated,
	})
	err = c.rpcCall(request, &estimatedFee)
	if err != nil {
		return "", err
	}
	return estimatedFee, nil
}
