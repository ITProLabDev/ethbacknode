package ethclient

import (
	"github.com/ITProLabDev/ethbacknode/types"
	"strings"
)

func (c *Client) tokenGetIfExistByAddress(contractAddress string) (tokenInfo *types.TokenInfo, found bool) {
	for _, token := range c.tokens {
		if strings.ToLower(token.ContractAddress) == strings.ToLower(contractAddress) {
			return token, true
		}
	}
	return nil, false
}

func (c *Client) tokenGetIfExistBySymbol(symbol string) (tokenInfo *types.TokenInfo, found bool) {
	for _, token := range c.tokens {
		if strings.ToLower(token.Symbol) == strings.ToLower(symbol) {
			return token, true
		}
	}
	return nil, false
}
