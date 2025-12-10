package ethclient

import (
	"backnode/common/hexnum"
	"encoding/json"
	"math/big"
)

type Transaction struct {
	BlockHash        string          `json:"blockHash"`
	BlockNumber      int64           `json:"blockNumber"`
	Hash             string          `json:"hash"`
	From             string          `json:"from"`
	To               string          `json:"to"`
	Gas              int64           `json:"gas"`
	GasPrice         *big.Int        `json:"gasPrice"`
	Value            *big.Int        `json:"value"`
	Input            string          `json:"input"`
	Nonce            int64           `json:"nonce"`
	TransactionIndex int64           `json:"transactionIndex"`
	Type             int64           `json:"type"`
	ChainId          int64           `json:"chainId"`
	AccessList       json.RawMessage `json:"accessList"`
	V                string          `json:"v"`
	R                string          `json:"r"`
	S                string          `json:"s"`
}

// UnmarshalJSON Since Ethereum uses non-standard encoding of integer data
// (0x prefixed hex) in its RPC responses, we need to implement full
// decoding of transactions and blocks through a special method.
func (t *Transaction) UnmarshalJSON(data []byte) (err error) {
	proxy := make(map[string]json.RawMessage)
	err = json.Unmarshal(data, &proxy)
	if blockHash, found := proxy[`blockHash`]; found {
		err = json.Unmarshal(blockHash, &t.BlockHash)
		if err != nil {
			return err
		}
	}
	if blockNumber, found := proxy[`blockNumber`]; found {
		var blockNumberStr string
		err = json.Unmarshal(blockNumber, &blockNumberStr)
		if err != nil {
			return err
		}
		if blockNumberStr != "" {
			t.BlockNumber, err = hexnum.ParseHexInt64(blockNumberStr)
			if err != nil {
				return err
			}
		}
	}
	if from, found := proxy[`from`]; found {
		err = json.Unmarshal(from, &t.From)
		if err != nil {
			return err
		}
	}
	if gas, found := proxy[`gas`]; found {
		var gasStr string
		err = json.Unmarshal(gas, &gasStr)
		if err != nil {
			return err
		}
		if gasStr != "" && gasStr != "0x" {
			t.Gas, err = hexnum.ParseHexInt64(gasStr)
			if err != nil {
				return err
			}
		}
	}
	if gasPrice, found := proxy[`gasPrice`]; found {
		var gasPriceStr string
		err = json.Unmarshal(gasPrice, &gasPriceStr)
		if err != nil {
			return err
		}
		t.GasPrice, err = hexnum.ParseBigInt(gasPriceStr)
		if err != nil {
			return err
		}
	}
	if hash, found := proxy[`hash`]; found {
		err = json.Unmarshal(hash, &t.Hash)
		if err != nil {
			return err
		}
	}
	if input, found := proxy[`input`]; found {
		err = json.Unmarshal(input, &t.Input)
		if err != nil {
			return err
		}
	}
	if nonce, found := proxy[`nonce`]; found {
		var nonceStr string
		err = json.Unmarshal(nonce, &nonceStr)
		if err != nil {
			return err
		}
		if nonceStr != "" && nonceStr != "0x" {
			t.Nonce, err = hexnum.ParseHexInt64(nonceStr)
			if err != nil {
				return err
			}
		}
	}
	if to, found := proxy[`to`]; found {
		err = json.Unmarshal(to, &t.To)
		if err != nil {
			return err
		}
	}
	if transactionIndex, found := proxy[`transactionIndex`]; found {
		var transactionIndexStr string
		err = json.Unmarshal(transactionIndex, &transactionIndexStr)
		if err != nil {
			return err
		}
		if transactionIndexStr != "" && transactionIndexStr != "0x" {
			t.TransactionIndex, err = hexnum.ParseHexInt64(transactionIndexStr)
			if err != nil {
				return err
			}
		}
	}
	if value, found := proxy[`value`]; found {
		var valueStr string
		err = json.Unmarshal(value, &valueStr)
		if err != nil {
			return err
		}
		t.Value, err = hexnum.ParseBigInt(valueStr)
		if err != nil {
			return err
		}
	}
	if txType, found := proxy[`type`]; found {
		var typeStr string
		err = json.Unmarshal(txType, &typeStr)
		if err != nil {
			return err
		}
		t.Type, err = hexnum.ParseHexInt64(typeStr)
		if err != nil {
			return err
		}
	}
	if chainId, found := proxy[`chainId`]; found {
		var chainIdStr string
		err = json.Unmarshal(chainId, &chainIdStr)
		if err != nil {
			return err
		}
		t.ChainId, err = hexnum.ParseHexInt64(chainIdStr)
		if err != nil {
			return err
		}
	}
	if accessList, found := proxy[`accessList`]; found {
		t.AccessList = accessList

	}
	if v, found := proxy[`v`]; found {
		err = json.Unmarshal(v, &t.V)
		if err != nil {
			return err
		}
	}
	if r, found := proxy[`r`]; found {
		err = json.Unmarshal(r, &t.R)
		if err != nil {
			return err
		}
	}
	if s, found := proxy[`s`]; found {
		err = json.Unmarshal(s, &t.S)
		if err != nil {
			return err
		}
	}

	return nil
}

// TODO we need implement ABI decoding fo Input (Data) field
