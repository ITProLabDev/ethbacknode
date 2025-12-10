package ethclient

import (
	"errors"
	"module github.com/ITProLabDev/ethbacknode/abi"
	"module github.com/ITProLabDev/ethbacknode/tools/log"
	"module github.com/ITProLabDev/ethbacknode/types"
)

func (c *Client) blockDecode(block *Block) (blockInfo *types.BlockInfo, err error) {
	blockDecoded := &types.BlockInfo{
		BlockID:    block.Hash,
		Number:     int(block.Number),
		ParentHash: block.ParentHash,
		Timestamp:  block.Timestamp,
	}
	if block.FullTransactions {
		for _, txInternal := range block.transactionsFullDecoded {
			txDecoded, err := c.transactionDecode(txInternal)
			if err != nil &&
				!errors.Is(err, ErrTransactionNotTransfer) &&
				!errors.Is(err, ErrUnsupportedTransactionType) &&
				!errors.Is(err, abi.ErrUnknownContract) &&
				!errors.Is(err, ErrUnsupportedTransactionFormat) {
				return nil, err
			} else if err == nil {
				txDecoded.InPool = false
				txDecoded.Timestamp = block.Timestamp
				txDecoded.BlockNum = blockDecoded.Number
				blockDecoded.Transactions = append(blockDecoded.Transactions, txDecoded)
			} else {
				log.Debug("Skipping transaction:", err)
			}
		}
	} else {
		blockDecoded.Transactions = make([]*types.TransferInfo, len(block.Transactions))
		for i, txHash := range block.transactionsHashesDecoded {
			blockDecoded.Transactions[i] = &types.TransferInfo{
				TxID: txHash,
			}
		}
	}
	return blockDecoded, nil
}
