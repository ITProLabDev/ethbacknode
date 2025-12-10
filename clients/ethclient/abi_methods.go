package ethclient

import (
	"math/big"
	"module github.com/ITProLabDev/ethbacknode/common/hexnum"
)

func (c *Client) ContractGetBalanceOf(contractAddress, address string) (balance *big.Int, err error) {
	callTx, err := c.abi.Erc20CallGetBalance(address)
	if err != nil {
		return nil, err
	}
	result, err := c.Call(contractAddress, callTx)
	if err != nil {
		return nil, err
	}
	b, err := hexnum.ParseHexBytes(result)
	if err != nil {
		return nil, err
	}
	return c.abi.Erc20DecodeAmount(b), nil
}
