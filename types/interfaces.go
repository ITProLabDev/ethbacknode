// Package types defines core interfaces and data structures for blockchain interaction.
// It provides abstractions for chain clients, transactions, blocks, and tokens.
package types

import (
	"github.com/ITProLabDev/ethbacknode/address"
	"math/big"
)

// ChainClientInfo provides blockchain metadata and configuration.
// Implementations should return chain-specific information.
type ChainClientInfo interface {
	GetChainId() (chainId string)
	GetChainName() (chainName string)
	GetChainSymbol() (chainSymbol string)
	GetAddressCodec() address.AddressCodec
	Decimals() (decimals int)
	TokensList() (tokensList []*TokenInfo)
	MinConfirmations() (confirmations int)
	TokenProtocols() []string
}

// ChainClientBlocks provides block query operations.
// Supports querying by block number or hash.
type ChainClientBlocks interface {
	// BlockNum returns the current (latest) block number.
	BlockNum() (blockNum int64, err error)
	// BlockByNum retrieves a block by its number. If fullInfo is true, includes transactions.
	BlockByNum(blockNum int64, fullInfo bool) (block *BlockInfo, err error)
	// BlockByHash retrieves a block by its hash. If fullInfo is true, includes transactions.
	BlockByHash(blockHash string, fullInfo bool) (block *BlockInfo, err error)
}

// ChainClientMemPool provides access to pending transactions in the memory pool.
type ChainClientMemPool interface {
	// MemPoolContent returns all pending transactions in the mempool.
	MemPoolContent() (txs []*TransferInfo, err error)
}

// ChainClientTransactions provides transaction query operations.
type ChainClientTransactions interface {
	// TransferInfoByHash retrieves a transaction by its hash.
	TransferInfoByHash(txHash string) (tx *TransferInfo, err error)
	// TransferInfoByNum retrieves a transaction by block number and transaction index.
	TransferInfoByNum(blockNum int64, txIndex int) (tx *TransferInfo, err error)
}

// ChainClientBalances provides balance query operations for addresses.
type ChainClientBalances interface {
	// BalanceOf returns the native coin balance of an address.
	BalanceOf(address string) (balance *big.Int, err error)
	// TokensBalanceOf returns the token balance of an address for a specific token.
	TokensBalanceOf(address string, token string) (balance *big.Int, err error)
}

// ChainClientCoinTransfer provides native coin transfer operations.
type ChainClientCoinTransfer interface {
	// TransferByPrivateKey sends native coins from one address to another.
	TransferByPrivateKey(fromPrivateKey []byte, from, to string, amount *big.Int) (txHash string, err error)
	// TransferGetEstimatedFee estimates the fee for a native coin transfer.
	TransferGetEstimatedFee(from, to string, amount *big.Int) (fee *big.Int, err error)
	// TransferAllByPrivateKey sends the entire balance (minus fees) to another address.
	TransferAllByPrivateKey(fromPrivateKey []byte, from, to string) (txHash string, err error)
}

// ChainClientTokenTransfer provides ERC-20 token transfer operations.
type ChainClientTokenTransfer interface {
	// TransferTokenByPrivateKey sends tokens from one address to another.
	TransferTokenByPrivateKey(fromPrivateKey []byte, from, to string, amount *big.Int, token string) (txHash string, err error)
	// TransferAllTokenByPrivateKey sends the entire token balance to another address.
	TransferAllTokenByPrivateKey(fromPrivateKey []byte, from, to string, token string) (txHash string, err error)
	// TransferTokenGetEstimatedFee estimates the fee for a token transfer.
	TransferTokenGetEstimatedFee(from, to string, amount *big.Int, token string) (fee *big.Int, err error)
}

// ChainClient is the main interface for blockchain interaction.
// It aggregates all chain client capabilities into a single interface.
type ChainClient interface {
	ChainClientInfo
	ChainClientMemPool
	ChainClientBlocks
	ChainClientTransactions
	ChainClientBalances
	ChainClientCoinTransfer
	ChainClientTokenTransfer
}

// TxCache provides cached transaction lookups.
// Used for fast retrieval of recently seen transactions.
type TxCache interface {
	// GetTransferInfo retrieves a cached transaction by its hash.
	GetTransferInfo(txHash string) (tx *TransferInfo, err error)
	// GetTransfersByAddress retrieves all cached transactions for an address.
	GetTransfersByAddress(address string) (txs []*TransferInfo, err error)
}
