// Package ethclient provides an Ethereum blockchain client that implements
// the types.ChainClient interface. It supports balance queries, transaction
// monitoring, and coin/token transfers via JSON-RPC.
package ethclient

import (
	"errors"
	"github.com/ITProLabDev/ethbacknode/abi"
	"github.com/ITProLabDev/ethbacknode/address"
	"github.com/ITProLabDev/ethbacknode/clients/urpc"
	"github.com/ITProLabDev/ethbacknode/common/hexnum"
	"github.com/ITProLabDev/ethbacknode/tools/log"
	"github.com/ITProLabDev/ethbacknode/types"
	"math/big"
	"strings"
)

// Option is a function that configures a Client.
type Option func(*Client)

// Default confirmation count constants.
const (
	// DEFAULT_CONFIRMATIONS is the default number of block confirmations
	// required before a transaction is considered finalized.
	DEFAULT_CONFIRMATIONS = 12
)

// NewClient creates a new Ethereum client with the specified options.
// Default configuration: Ethereum mainnet, 18 decimals, 12 confirmations.
func NewClient(options ...Option) *Client {
	client := &Client{
		config:           &Config{storage: _configDefaultStorage()},
		chainName:        "Ethereum",
		chainId:          "ethereum",
		chainSymbol:      "ETH",
		decimals:         18,
		addressCodec:     GetAddressCodec(),
		minConfirmations: DEFAULT_CONFIRMATIONS,
	}
	for _, option := range options {
		option(client)
	}
	return client
}

// Client is the main Ethereum blockchain client.
// It implements the types.ChainClient interface.
type Client struct {
	config           *Config                    // Client configuration
	chainName        string                     // Human-readable chain name
	chainId          string                     // Chain identifier
	chainSymbol      string                     // Native currency symbol (ETH)
	decimals         int                        // Native currency decimals (18)
	abi              *abi.SmartContractsManager // Smart contract ABI manager
	rpcClient        *urpc.Client               // JSON-RPC client
	addressCodec     address.AddressCodec       // Address encoder/decoder
	tokens           []*types.TokenInfo         // Supported tokens list
	minConfirmations int                        // Required confirmations
}

// BalanceOf returns the native coin balance of an address in wei.
func (c *Client) BalanceOf(address string) (balance *big.Int, err error) {
	return c.GetBalance(address)
}

// TokensBalanceOf returns the token balance of an address.
// Accepts token symbol or contract address.
func (c *Client) TokensBalanceOf(address string, token string) (balance *big.Int, err error) {
	tokenInfo, ok := c.tokenGetIfExistBySymbol(token)
	if !ok {
		tokenInfo, ok = c.tokenGetIfExistByAddress(token)
		if !ok {
			return nil, ErrUnknownToken
		}
	}
	return c.ContractGetBalanceOf(tokenInfo.ContractAddress, address)
}

// Init loads configuration and initializes the client.
// Must be called before using the client.
func (c *Client) Init() error {
	err := c.config.Load()
	if err != nil {
		return err
	}
	if c.config.ChainName != "" {
		c.chainName = c.config.ChainName
	}
	if c.config.ChainId != "" {
		c.chainId = c.config.ChainId
	}
	if c.config.ChainSymbol != "" {
		c.chainSymbol = c.config.ChainSymbol
	}
	if c.config.Decimals != 0 {
		c.decimals = c.config.Decimals
	}
	if c.config.Confirmations != 0 {
		c.minConfirmations = c.config.Confirmations
	}
	c.tokens = c.config.Tokens
	return nil
}
func (c *Client) SetConfirmations(confirmations int) {
	c.minConfirmations = confirmations
}

func (c *Client) GetChainId() (chainId string) {
	return c.chainId
}

func (c *Client) GetChainName() (chainName string) {
	return c.chainName
}

func (c *Client) GetChainSymbol() (chainSymbol string) {
	return c.chainSymbol
}

func (c *Client) GetAddressCodec() address.AddressCodec {
	return &AddressCodec{}
}

func (c *Client) MinConfirmations() (confirmations int) {
	return c.minConfirmations
}

func (c *Client) TokenProtocols() []string {
	var protocols []string
	for _, token := range c.tokens {
		if !_isProtoInSlice(token.Protocol, protocols) {
			protocols = append(protocols, token.Protocol)
		}
	}
	return protocols
}

func (c *Client) Decimals() (decimals int) {
	return c.decimals
}

func (c *Client) TokensList() (tokensList []*types.TokenInfo) {
	return c.tokens
}

func (c *Client) MemPoolContent() (poolContent []*types.TransferInfo, err error) {
	pending, queued, err := c.GetTxPoolContent()
	if err != nil {
		return nil, err
	}
	if len(pending) == 0 && len(queued) == 0 {
		return nil, nil
	}
	for _, txBlocks := range pending {
		for _, tx := range txBlocks {
			t, err := c.transactionDecode(tx)
			if err != nil &&
				!errors.Is(err, ErrTransactionNotTransfer) &&
				!errors.Is(err, ErrUnsupportedTransactionType) &&
				!errors.Is(err, abi.ErrUnknownContract) &&
				!errors.Is(err, ErrUnsupportedTransactionFormat) {
				return nil, err
			} else if err == nil {
				t.InPool = true
				poolContent = append(poolContent, t)
			} else {
				err = nil
				log.Debug("Skipping transaction:", err)
			}
		}
	}
	for _, txBlocks := range queued {
		for _, tx := range txBlocks {
			t, err := c.transactionDecode(tx)
			if err != nil &&
				!errors.Is(err, ErrTransactionNotTransfer) &&
				!errors.Is(err, ErrUnsupportedTransactionType) &&
				!errors.Is(err, abi.ErrUnknownContract) &&
				!errors.Is(err, ErrUnsupportedTransactionFormat) {
				return nil, err
			} else if err == nil {
				t.InPool = true
				poolContent = append(poolContent, t)
			} else {
				log.Debug("Skipping transaction:", err)
			}
		}
	}
	return poolContent, nil
}

func (c *Client) BlockNum() (blockNum int64, err error) {
	return c.GetBlockNumber()
}

func (c *Client) BlockByNum(blockNum int64, fullInfo bool) (block *types.BlockInfo, err error) {
	block = new(types.BlockInfo)
	blockInternal, err := c.GetBlockByNumber(blockNum, fullInfo)
	if err != nil {
		return nil, err
	}
	block, err = c.blockDecode(blockInternal)
	return block, nil
}

func (c *Client) BlockByHash(blockHash string, fullInfo bool) (block *types.BlockInfo, err error) {
	//blockInternal
	_, err = c.GetBlockByHash(blockHash, fullInfo)
	if err != nil {
		return nil, err
	}
	panic("implement me")
}

func (c *Client) TransferInfoByHash(txHash string) (tx *types.TransferInfo, err error) {
	txInternal, err := c.GetTransactionByHash(txHash)
	if err != nil {
		return nil, err
	}
	tx, err = c.transactionDecode(txInternal)
	if err != nil {
		return nil, err
	}
	currentBlock, _ := c.GetBlockNumber()
	if txInternal.BlockNumber == 0 {
		tx.InPool = true
	} else {
		tx.Confirmations = int(currentBlock-txInternal.BlockNumber) + 1
		if currentBlock-txInternal.BlockNumber < int64(c.minConfirmations-1) {
			tx.Confirmed = false
		} else {
			tx.Confirmed = true
		}
	}

	return tx, nil
}

func (c *Client) TransferInfoByNum(blockNum int64, txIndex int) (tx *types.TransferInfo, err error) {
	txInternal, err := c.GetTransactionByBlockNumberAndIndex(blockNum, txIndex)
	if err != nil {
		return nil, err
	}
	tx, err = c.transactionDecode(txInternal)
	if err != nil {
		return nil, err
	}
	currentBlock, _ := c.GetBlockNumber()
	if txInternal.BlockNumber == 0 {
		tx.InPool = true
	}
	if currentBlock-txInternal.BlockNumber > int64(c.minConfirmations) {
		tx.Confirmed = true
	}
	tx.Confirmations = int(currentBlock - txInternal.BlockNumber)
	return tx, nil
}

func (c *Client) TransactionSendRaw(rawTx []byte) (txHash string, err error) {
	return c.SendRawTransaction(hexnum.BytesToHex(rawTx))
}

func (c *Client) TransferByPrivateKey(fromPrivateKey []byte, from, to string, amount *big.Int) (txHash string, err error) {
	fromAddress, _, err := c.addressCodec.PrivateKeyToAddress(fromPrivateKey)
	if err != nil {
		return "", err
	}
	if strings.ToUpper(from) != strings.ToUpper(fromAddress) {
		return "", address.ErrAddressPrivateKeyMismatch
	}
	_, err = c.addressCodec.DecodeAddressToBytes(to)
	if err != nil {
		return "", err
	}
	currentBalance, err := c.GetBalance(from)
	if err != nil {
		return "", err
	}
	log.Warning("Current Balance:", currentBalance)
	fee, gasPrice, gas, err := c.GetEstimatedFee(fromAddress, to, "", currentBalance)
	if err != nil {
		return "", err
	}
	amountWithFee := new(big.Int).Add(amount, fee)
	log.Warning("Estimated Fee:", fee)
	log.Warning("Amount with Fee:", amountWithFee)
	if amountWithFee.Cmp(currentBalance) > 0 {
		return "", ErrInsufficientFunds
	}
	return c.sendRawByPrivateKeyUnsafe(fromPrivateKey, from, to, amount, gasPrice, gas)
}

func (c *Client) TransferAllByPrivateKey(fromPrivateKey []byte, from, to string) (txHash string, err error) {
	fromAddress, _, err := c.addressCodec.PrivateKeyToAddress(fromPrivateKey)
	if err != nil {
		return "", err
	}
	if strings.ToUpper(from) != strings.ToUpper(fromAddress) {
		return "", address.ErrAddressPrivateKeyMismatch
	}
	_, err = c.addressCodec.DecodeAddressToBytes(to)
	if err != nil {
		return "", err
	}
	currentBalance, err := c.GetBalance(from)
	if err != nil {
		return "", err
	}
	log.Warning("Current Balance:", currentBalance)
	fee, gasPrice, gas, err := c.GetEstimatedFee(fromAddress, to, "", currentBalance)
	if err != nil {
		return "", err
	}
	amountToTransfer := new(big.Int).Sub(currentBalance, fee)
	log.Warning("Estimated Fee:", fee)
	log.Warning("Amount to transfer:", amountToTransfer)
	if amountToTransfer.Cmp(big.NewInt(0)) <= 0 {
		return "", ErrNothingToTransfer
	}
	return c.sendRawByPrivateKeyUnsafe(fromPrivateKey, from, to, amountToTransfer, gasPrice, gas)
}

func (c *Client) TransferGetEstimatedFee(from, to string, amount *big.Int) (fee *big.Int, err error) {
	fee, _, _, err = c.GetEstimatedFee(from, to, "", amount)
	if err != nil {
		return nil, err
	}
	return fee, nil
}

func (c *Client) TransferTokenByPrivateKey(fromPrivateKey []byte, from, to string, amount *big.Int, token string) (txHash string, err error) {
	//TODO implement me
	panic("implement me")
}

func (c *Client) TransferAllTokenByPrivateKey(fromPrivateKey []byte, from, to string, token string) (txHash string, err error) {
	//TODO implement me
	panic("implement me")
}

func (c *Client) TransferTokenGetEstimatedFee(from, to string, amount *big.Int, token string) (fee *big.Int, err error) {
	//TODO implement me
	panic("implement me")
}

func _isProtoInSlice(proto string, slice []string) bool {
	for _, p := range slice {
		if p == proto {
			return true
		}
	}
	return false
}
