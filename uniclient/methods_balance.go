package uniclient

import (
	"encoding/json"
	"github.com/ITProLabDev/ethbacknode/tools/log"
)

// BalanceGetForAddress retrieves the balance of a specific asset for an address.
func (c *Client) BalanceGetForAddress(address string, symbol string) (balance string, err error) {
	type addressBalanceRequest struct {
		Address   string `json:"address"`
		Assets    string `json:"assets"`
		AllAssets bool   `json:"allAssets"`
		Formatted bool   `json:"formatted"`
		Extended  bool   `json:"extended"`
	}
	request := NewRequest("addressGetBalance", &addressBalanceRequest{
		Address:   address,
		Assets:    symbol,
		Formatted: true,
		AllAssets: false,
	})

	balanceResponse := make(map[string]json.Number)
	err = c.rpcCall(request, &balanceResponse)
	if err != nil {
		return "", err
	}
	log.Dump(balanceResponse)
	if balance, ok := balanceResponse[symbol]; ok {
		return balance.String(), nil
	}
	return "", ErrInvalidBalanceResponse
}
// BalanceGetForAddressAllAssets retrieves balances for all assets at an address.
func (c *Client) BalanceGetForAddressAllAssets(address string) (balance map[string]json.Number, err error) {
	type addressBalanceRequest struct {
		Address   string `json:"address"`
		AllAssets bool   `json:"allAssets"`
		Formatted bool   `json:"formatted"`
		Extended  bool   `json:"extended"`
	}
	request := NewRequest("addressGetBalance", &addressBalanceRequest{
		Address:   address,
		Formatted: true,
		AllAssets: true,
	})
	err = c.rpcCall(request, &balance)
	if err != nil {
		return nil, err
	}
	log.Dump(balance)
	if len(balance) != 0 {
		return balance, nil
	}
	return nil, ErrInvalidBalanceResponse
}
