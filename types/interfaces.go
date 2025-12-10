package types

import (
	"backnode/address"
	"math/big"
)

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

type ChainClientBlocks interface {
	BlockNum() (blockNum int64, err error)
	BlockByNum(blockNum int64, fullInfo bool) (block *BlockInfo, err error)
	BlockByHash(blockHash string, fullInfo bool) (block *BlockInfo, err error)
}
type ChainClientMemPool interface {
	MemPoolContent() (txs []*TransferInfo, err error)
}
type ChainClientTransactions interface {
	TransferInfoByHash(txHash string) (tx *TransferInfo, err error)
	TransferInfoByNum(blockNum int64, txIndex int) (tx *TransferInfo, err error)
}

type ChainClientBalances interface {
	BalanceOf(address string) (balance *big.Int, err error)
	TokensBalanceOf(address string, token string) (balance *big.Int, err error)
}

type ChainClientCoinTransfer interface {
	TransferByPrivateKey(fromPrivateKey []byte, from, to string, amount *big.Int) (txHash string, err error)
	TransferGetEstimatedFee(from, to string, amount *big.Int) (fee *big.Int, err error)
	TransferAllByPrivateKey(fromPrivateKey []byte, from, to string) (txHash string, err error)
}

type ChainClientTokenTransfer interface {
	TransferTokenByPrivateKey(fromPrivateKey []byte, from, to string, amount *big.Int, token string) (txHash string, err error)
	TransferAllTokenByPrivateKey(fromPrivateKey []byte, from, to string, token string) (txHash string, err error)
	TransferTokenGetEstimatedFee(from, to string, amount *big.Int, token string) (fee *big.Int, err error)
}

type ChainClient interface {
	ChainClientInfo
	ChainClientMemPool
	ChainClientBlocks
	ChainClientTransactions
	ChainClientBalances
	ChainClientCoinTransfer
	ChainClientTokenTransfer
}

type TxCache interface {
	GetTransferInfo(txHash string) (tx *TransferInfo, err error)
	GetTransfersByAddress(address string) (txs []*TransferInfo, err error)
}
