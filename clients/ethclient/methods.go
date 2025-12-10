package ethclient

import (
	"backnode/clients/urpc"
	"backnode/common/hexnum"
	"backnode/tools/log"
	"math/big"
)

const (
	ethChainId                             = "eth_chainId"
	ethGetBalance                          = "eth_getBalance"
	ethGetTransactionByHash                = "eth_getTransactionByHash"
	ethGetTransactionReceipt               = "eth_getTransactionReceipt"
	ethGetTransactionByBlockHashAndIndex   = "eth_getTransactionByBlockHashAndIndex"
	ethGetTransactionByBlockNumberAndIndex = "eth_getTransactionByBlockNumberAndIndex"
	ethGetBlockNumber                      = "eth_blockNumber"
	ethGetBlockByHash                      = "eth_getBlockByHash"
	ethGetBlockByNumber                    = "eth_getBlockByNumber"
	ethEstimateGas                         = "eth_estimateGas"
	ethGasPrice                            = "eth_gasPrice"
	ethSendRawTransaction                  = "eth_sendRawTransaction"
	ethGetTransactionCount                 = "eth_getTransactionCount"
	ethCall                                = "eth_call"
	txpoolСontent                          = "txpool_content"

	web3Version = "web3_version"

	tagBlockLatest = "latest"
)

func (c *Client) GetNetId() (netId int64, err error) {
	req := urpc.NewRequest(ethChainId)
	result, err := c.rpcClient.Call(req)
	if err != nil {
		return 0, err
	}
	var netIdStr string
	err = result.ParseResult(&netIdStr)
	if err != nil {
		return 0, err
	}
	netId, err = hexnum.ParseHexInt64(netIdStr)
	if err != nil {
		return 0, err
	}
	return netId, nil
}

// GetBalance returns the balance of the account of given address.
// The balance is returned in wei. if you want to convert it to ether,
// use WeiToEther function.
func (c *Client) GetBalance(address string) (*big.Int, error) {
	req := urpc.NewRequest(ethGetBalance)
	req.AddParams(address, "latest")
	result, err := c.rpcClient.Call(req)
	if err != nil {
		return nil, err
	}
	var balanceStr string
	err = result.ParseResult(&balanceStr)
	if err != nil {
		return nil, err
	}
	balance, err := hexnum.ParseBigInt(balanceStr)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

// GasPrice returns the current price per gas in wei.
// The gas price is determined by the last few blocks
// median gas price. You need know the gas price for
// calculate the fee of transaction send/execute.
func (c *Client) GasPrice() (*big.Int, error) {
	req := urpc.NewRequest(ethGasPrice)
	result, err := c.rpcClient.Call(req)
	if err != nil {
		return nil, err
	}
	var gasPriceStr string
	err = result.ParseResult(&gasPriceStr)
	if err != nil {
		return nil, err
	}
	gasPrice, err := hexnum.ParseBigInt(gasPriceStr)
	if err != nil {
		return nil, err
	}
	return gasPrice, nil
}

// GetTransactionByHash returns the information about a transaction requested by
// transaction hash. If the transaction not found (geth rpc call return null), it returns error.
func (c *Client) GetTransactionByHash(hash string) (*Transaction, error) {
	req := urpc.NewRequest(ethGetTransactionByHash)
	req.SetParams(hash)
	result, err := c.rpcClient.Call(req)
	if err != nil {
		return nil, err
	}
	if result.Result == nil || string(result.Result) == "null" {
		return nil, ErrTransactionNotFound
	}
	tx := new(Transaction)
	err = result.ParseResult(tx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// GetTransactionByBlockHashAndIndex returns the information about a transaction requested by
// block hash and tx index. If the transaction not found (geth rpc call return null),
// it returns error.
func (c *Client) GetTransactionByBlockHashAndIndex(hash string, index int) (*Transaction, error) {
	req := urpc.NewRequest(ethGetTransactionByBlockHashAndIndex)
	req.AddParams(hash, hexnum.IntToHex(index))
	result, err := c.rpcClient.Call(req)
	if err != nil {
		return nil, err
	}
	if result.Result == nil || string(result.Result) == "null" {
		return nil, ErrTransactionNotFound
	}
	tx := new(Transaction)
	err = result.ParseResult(tx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// GetTransactionByBlockNumberAndIndex returns the information about a transaction requested by
// block number and tx index. If the transaction not found (geth rpc call return null),
// it returns error.
func (c *Client) GetTransactionByBlockNumberAndIndex(blockNumber int64, index int) (*Transaction, error) {
	req := urpc.NewRequest(ethGetTransactionByBlockNumberAndIndex)
	req.AddParams(hexnum.Int64ToHex(blockNumber), hexnum.IntToHex(index))
	result, err := c.rpcClient.Call(req)
	if err != nil {
		return nil, err
	}
	if result.Result == nil || string(result.Result) == "null" {
		return nil, ErrTransactionNotFound
	}
	tx := new(Transaction)
	err = result.ParseResult(tx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// GetBlockNumber returns the number of most recent block.
func (c *Client) GetBlockNumber() (int64, error) {
	req := urpc.NewRequest(ethGetBlockNumber)
	result, err := c.rpcClient.Call(req)
	if err != nil {
		return 0, err
	}
	var blockNumberStr string
	err = result.ParseResult(&blockNumberStr)
	if err != nil {
		return 0, err
	}
	blockNumber, err := hexnum.ParseHexInt64(blockNumberStr)
	if err != nil {
		return 0, err
	}
	return blockNumber, nil
}

// GetBlockByHash returns information about a block by hash.
// If "fullTransactions" is true it returns the full transaction objects,
// if "fullTransactions" is false, only the hashes of the transactions.
func (c *Client) GetBlockByHash(hash string, fullTransactions bool) (*Block, error) {
	req := urpc.NewRequest(ethGetBlockByHash)
	req.AddParams(hash, fullTransactions)
	result, err := c.rpcClient.Call(req)
	if err != nil {
		return nil, err
	}
	block := new(Block)
	block.FullTransactions = fullTransactions
	err = result.ParseResult(block)
	if err != nil {
		return nil, err
	}
	return block, nil
}

// GetBlockByNumber returns information about a block by block number.
// If "fullTransactions" is true it returns the full transaction objects,
// if "fullTransactions" is false, only the hashes of the transactions.
func (c *Client) GetBlockByNumber(number int64, fullTransactions bool) (*Block, error) {
	req := urpc.NewRequest(ethGetBlockByNumber)
	req.AddParams(hexnum.Int64ToHex(number), fullTransactions)
	result, err := c.rpcClient.Call(req)
	if err != nil {
		return nil, err
	}
	block := new(Block)
	block.FullTransactions = fullTransactions
	err = result.ParseResult(block)
	if err != nil {
		return nil, err
	}
	return block, nil
}

// SendRawTransaction sends the signed and RPL encoded transaction to the network.
// In fact, any transaction - transfer of funds, call of a smart contract function
// or deployment of a smart contract is carried out by calling this function
func (c *Client) SendRawTransaction(data string) (txHash string, err error) {
	req := urpc.NewRequest(ethSendRawTransaction)
	req.AddParams(data)
	result, err := c.rpcClient.Call(req)
	if err != nil {
		return "", err
	}
	err = result.ParseResult(&txHash)
	if err != nil {
		return "", err
	}
	return txHash, nil
}

// Call executes a new message call immediately without creating a
// transaction on the block chain. By default "latest" is used for
// the block tag. If you need call with specific block number, use
// CallByBlockNumber.
// Often used for executing read-only smart contract functions,
// for example the balanceOf for an ERC-20 contract.
func (c *Client) Call(contractAddress, data string) (callResult string, err error) {
	callTx := &struct {
		To    string `json:"to"`
		Input string `json:"input"`
	}{
		To:    contractAddress,
		Input: data,
	}
	req := urpc.NewRequest(ethCall)
	req.AddParams(callTx, tagBlockLatest)
	result, err := c.rpcClient.Call(req)
	if err != nil {
		return "", err
	}
	err = result.ParseResult(&callResult)
	if err != nil {
		return "", err
	}
	return callResult, nil
}

// CallByBlockNumber executes a new message call immediately without creating a
// transaction on the block chain.
// Often used for executing read-only smart contract functions,
// for example the balanceOf for an ERC-20 contract.
func (c *Client) CallByBlockNumber(contractAddress, data string, blockNumber int64) (callResult string, err error) {
	callTx := &struct {
		To    string `json:"to"`
		Input string `json:"input"`
	}{
		To:    contractAddress,
		Input: data,
	}
	req := urpc.NewRequest(ethCall)
	req.AddParams(callTx, hexnum.Int64ToHex(blockNumber))
	result, err := c.rpcClient.Call(req)
	if err != nil {
		return "", err
	}
	err = result.ParseResult(&callResult)
	if err != nil {
		return "", err
	}
	return callResult, nil
}

// GetTxPoolContent returns the information about the transaction pool.
// It returns two maps: pending and queued transactions.
func (c *Client) GetTxPoolContent() (pending, queued map[string]map[string]*Transaction, err error) {
	req := urpc.NewRequest(txpoolСontent)
	result, err := c.rpcClient.Call(req)
	if err != nil {
		return nil, nil, err
	}
	txPoolContent := &struct {
		Pending map[string]map[string]*Transaction `json:"pending"`
		Queued  map[string]map[string]*Transaction `json:"queued"`
	}{
		Pending: make(map[string]map[string]*Transaction),
		Queued:  make(map[string]map[string]*Transaction),
	}
	err = result.ParseResult(&txPoolContent)
	if err != nil {
		return nil, nil, err
	}
	return txPoolContent.Pending, txPoolContent.Queued, nil
}

type estimateGasRequest struct {
	FromAddress string `json:"from,omitempty"`
	ToAddress   string `json:"to,omitempty"`
	Amount      string `json:"value,omitempty"`
	Data        string `json:"input,omitempty"`
}

func (c *Client) GetEstimatedGas(from, to, data string, amount *big.Int) (gas int64, err error) {
	req := urpc.NewRequest(ethEstimateGas)
	req.AddParams(&estimateGasRequest{
		FromAddress: from,
		ToAddress:   to,
		Amount:      hexnum.BigIntToHex(amount),
		Data:        data,
	})
	log.Dump(req)
	result, err := c.rpcClient.Call(req)
	if err != nil {
		return 0, err
	}
	var gasStr string
	err = result.ParseResult(&gasStr)
	if err != nil {
		return 0, err
	}
	gas, err = hexnum.ParseHexInt64(gasStr)
	if err != nil {
		return 0, err
	}
	return gas, nil
}

func (c *Client) GetEstimatedGasPrice() (gasPrice *big.Int, err error) {
	req := urpc.NewRequest(ethGasPrice)
	result, err := c.rpcClient.Call(req)
	if err != nil {
		return nil, err
	}
	var gasPriceStr string
	err = result.ParseResult(&gasPriceStr)
	if err != nil {
		return nil, err
	}
	gasPrice, err = hexnum.ParseBigInt(gasPriceStr)
	if err != nil {
		return nil, err
	}
	return gasPrice, nil
}

func (c *Client) GetEstimatedFee(from, to, data string, amount *big.Int) (fee, gasPrice *big.Int, gas int64, err error) {
	gas, err = c.GetEstimatedGas(from, to, data, amount)
	if err != nil {
		return nil, nil, 0, err
	}
	log.Warning("Estimated Gas:", gas)
	gasPrice, err = c.GetEstimatedGasPrice()
	if err != nil {
		return nil, nil, 0, err
	}
	log.Warning("Estimated Gas Price:", gasPrice)
	fee = new(big.Int).Mul(gasPrice, big.NewInt(gas))
	return fee, gasPrice, gas, nil
}

func (c *Client) PendingNonceAt(address string) (nonce int64, err error) {
	req := urpc.NewRequest(ethGetTransactionCount)
	req.AddParams(address, "pending")
	result, err := c.rpcClient.Call(req)
	if err != nil {
		return 0, err
	}
	var nonceStr string
	err = result.ParseResult(&nonceStr)
	if err != nil {
		return 0, err
	}
	nonce, err = hexnum.ParseHexInt64(nonceStr)
	if err != nil {
		return 0, err
	}
	return nonce, nil
}
