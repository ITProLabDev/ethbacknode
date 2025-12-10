package ethclient

import (
	"backnode/common/hexnum"
	"backnode/tools/log"
	"backnode/types"
	"math/big"
	"time"
)

func (c *Client) transactionDecode(tx *Transaction) (txInfo *types.TransferInfo, err error) {
	fee := new(big.Int).Mul(tx.GasPrice, new(big.Int).SetUint64(uint64(tx.Gas)))
	txInfo = &types.TransferInfo{
		TxID:      tx.Hash,
		BlockNum:  int(tx.BlockNumber),
		Success:   true,
		Timestamp: time.Now().Unix(),
		From:      c._addressToNormal(tx.From),
		To:        c._addressToNormal(tx.To),
		Fee:       fee,
		//todo save chain specific data
		//ChainSpecificData:
	}
	if (tx.Input == "0x" || len(tx.Input) == 0) && tx.Value.Cmp(new(big.Int)) == 1 {
		txInfo.Transfer = true
		txInfo.Amount = tx.Value
		txInfo.NativeCoin = true
		txInfo.Symbol = c.chainSymbol
		txInfo.Decimals = c.decimals
	} else if tx.Input != "0x" {
		if tx.To == "" || tx.To == "0x" {
			return nil, ErrUnsupportedTransactionFormat
		}
		possibleContractAddress := c._addressToNormal(tx.To)
		if knownToken, yes := c.tokenGetIfExistByAddress(possibleContractAddress); yes {
			log.Debug("Known token:", knownToken.Name, "(", knownToken.Symbol, ")")
			callData, err := hexnum.ParseHexBytes(tx.Input)
			if err != nil {
				if c.config.Debug {
					log.Error("Can not parse call data from hex:", err)
				}
				return nil, ErrUnsupportedTransactionFormat
			}
			if c.abi.Erc20IsTransfer(callData) {
				txInfo.SmartContract = true
				txInfo.Token = knownToken.Name
				txInfo.TokenSymbol = knownToken.Symbol
				txInfo.Decimals = knownToken.Decimals
				toAddress, amountWei, err := c.abi.Erc20DecodeIfTransfer(callData)
				if err != nil {
					if c.config.Debug {
						log.Error("Can not Erc20 Decode Data:", err)
					}
					return nil, ErrUnsupportedTransactionType
				}
				if toAddress == "" || _isZeroBigInt(amountWei) {
					if c.config.Debug {
						log.Error("Invalid Erc20 Decoded Params:", toAddress == "", _isZeroBigInt(amountWei))
					}
					return nil, ErrUnsupportedTransactionType
				}
				txInfo.To = c._addressToNormal(toAddress)
				txInfo.Amount = amountWei
			} else {
				if c.config.Debug {
					log.Error("Unknown token call method:", tx.Input[:16])
				}
				return nil, ErrUnsupportedTransactionType
			}
		} else {
			if c.config.Debug {
				log.Error("Unknown token address:", possibleContractAddress)
			}
			return nil, ErrUnsupportedTransactionType
		}
	}
	return txInfo, nil
}

func (c *Client) _addressToNormal(address string) (normalAddress string) {
	ab, _ := c.addressCodec.DecodeAddressToBytes(address)
	normalAddress, _ = c.addressCodec.EncodeBytesToAddress(ab)
	return normalAddress
}

func _isZeroBigInt(b *big.Int) bool {
	return b.Cmp(new(big.Int)) == 0
}
