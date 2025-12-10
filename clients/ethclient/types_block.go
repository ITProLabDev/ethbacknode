package ethclient

import (
	"encoding/json"
	"github.com/ITProLabDev/ethbacknode/common/hexnum"
	"math/big"
)

type Block struct {
	// FullTransactions set to true if "returns the full transaction"
	// parameter set
	FullTransactions bool `json:"hashesOnly"`

	// Number The block number. null when its pending block.
	Number int64 `json:"number"`

	// Hash 32 Bytes - hash of the block. null when its pending block.
	Hash string `json:"hash"`

	// ParentHash 32 Bytes - hash of the parent block.
	ParentHash string `json:"parentHash"`

	// ParentBeaconBlockRoot 32 Bytes - hash of the parent beacon block.
	ParentBeaconBlockRoot string `json:"parentBeaconBlockRoot"`

	// Nonce 8 Bytes - hash of the generated proof-of-work. null when its pending block.
	Nonce uint64 `json:"nonce"`

	// Sha3Uncles 32 Bytes - SHA3 of the uncles data in the block.
	Sha3Uncles string `json:"sha3Uncles"`

	// LogsBloom 256 Bytes - the bloom filter for the logs of the block.
	// null when its pending block.
	LogsBloom string `json:"logsBloom"`

	// TransactionsRoot 32 Bytes - the root of the transaction
	// trie of the block.
	TransactionsRoot string `json:"transactionsRoot"`

	// StateRoot 32 Bytes - the root of the final state trie of the block.
	StateRoot string `json:"stateRoot"`

	// ReceiptsRoot 32 Bytes - the root of the receipts trie of the block.
	ReceiptsRoot string `json:"receiptsRoot"`

	// Miner DATA, 20 Bytes - the address of the beneficiary to whom the mining
	Miner string `json:"miner"`

	// BaseFeePerGas A string of the base fee encoded in hexadecimal format.
	// Please note that this response field will not be included in a block
	// requested before the EIP-1559 upgrade
	BaseFeePerGas *big.Int `json:"baseFeePerGas"`

	// Difficulty The integer of the difficulty for this block
	// encoded as a hexadecimal
	Difficulty int64 `json:"difficulty"`

	// TotalDifficulty integer of the difficulty for this block
	// encoded as a hexadecimal
	TotalDifficulty string `json:"totalDifficulty"`

	// ExtraData The "extra data" field of this block
	ExtraData string `json:"extraData"`

	// Size integer the size of this block in bytes.
	Size int64 `json:"size"`

	// GasLimit the maximum gas allowed in this block
	GasLimit int64 `json:"gasLimit"`

	// GasUsed the total used gas by all transactions in this block.
	GasUsed int64 `json:"gasUsed"`

	// Timestamp the unix timestamp for when the block was collated.
	Timestamp int64 `json:"timestamp"`

	// BlobGasUsed
	BlobGasUsed   string `json:"blobGasUsed"`
	ExcessBlobGas string `json:"excessBlobGas"`

	MixHash string `json:"mixHash"`

	// Transactions Array of transaction objects, or 32 Bytes
	// transaction hashes depending on the last given parameter.
	Transactions json.RawMessage `json:"transactions"`
	//so now we can prepare hidden fields for decoded transactions/hashes
	transactionsFullDecoded   []*Transaction
	transactionsHashesDecoded []string
	// Uncles Array of uncle hashes.
	Uncles []string `json:"uncles"`

	Withdrawals     json.RawMessage `json:"withdrawals"`
	WithdrawalsRoot string          `json:"withdrawalsRoot"`
}

// GetTransactions returns the transactions from the block.
// If the block was requested without full transactions, it will return error
func (b *Block) GetTransactions() (txs []*Transaction, err error) {
	if !b.FullTransactions {
		return nil, ErrHashesOnlyBlockHash
	}
	return b.transactionsFullDecoded, nil
}

// GetTransactionsHashes returns the transaction hashes from the block.
func (b *Block) GetTransactionsHashes() (txs []string) {
	return b.transactionsHashesDecoded
}

// WalkTransactions walks through the transactions of the block and calls
// the view function for each transaction. If the view function returns true,
// the walking is stopped. If the block was requested without full transactions,
// it will return error
func (b *Block) WalkTransactions(view func(tx *Transaction) (stop bool)) error {
	if !b.FullTransactions {
		return ErrHashesOnlyBlockHash
	}
	for _, tx := range b.transactionsFullDecoded {
		if view(tx) {
			return nil
		}
	}
	return nil
}

// WalkTransactionsHashes walks through the transaction hashes of the block and calls
// the view function for each transaction. If the view function returns true,
// the walking is stopped.
func (b *Block) WalkTransactionsHashes(view func(tx string) (stop bool)) {
	for _, tx := range b.transactionsHashesDecoded {
		if view(tx) {
			return
		}
	}
	return
}

// UnmarshalJSON Since Ethereum uses non-standard encoding of integer data
// (0x prefixed hex) in its RPC responses, we need to implement full
// decoding of transactions and blocks through a special method.
func (b *Block) UnmarshalJSON(data []byte) (err error) {
	// We need to use a proxy map to decode the data first, because we need to
	// decode fields by it type (string, int, etc) and then assign them to the
	// Block struct. If we decode directly into the Block struct, we will get
	// an error because the types will not match (used 0x prefixed hex vs int64, etc)
	proxy := make(map[string]json.RawMessage)
	err = json.Unmarshal(data, &proxy)
	if err != nil {
		return err
	}

	if number, found := proxy[`number`]; found {
		var numberStr string
		err = json.Unmarshal(number, &numberStr)
		if err != nil {
			return err
		}
		b.Number, err = hexnum.ParseHexInt64(numberStr)
		if err != nil {
			return err
		}
	}
	if hash, found := proxy[`hash`]; found {
		err = json.Unmarshal(hash, &b.Hash)
		if err != nil {
			return err
		}
	}
	if parentHash, found := proxy[`parentHash`]; found {
		err = json.Unmarshal(parentHash, &b.ParentHash)
		if err != nil {
			return err
		}
	}
	if parentBeaconBlockRoot, found := proxy[`parentBeaconBlockRoot`]; found {
		err = json.Unmarshal(parentBeaconBlockRoot, &b.ParentBeaconBlockRoot)
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
		b.Nonce, err = hexnum.ParseHexUint64(nonceStr)
		if err != nil {
			return err
		}
	}
	if sha3Uncles, found := proxy[`sha3Uncles`]; found {
		err = json.Unmarshal(sha3Uncles, &b.Sha3Uncles)
		if err != nil {
			return err
		}
	}
	if logsBloom, found := proxy[`logsBloom`]; found {
		err = json.Unmarshal(logsBloom, &b.LogsBloom)
		if err != nil {
			return err
		}
	}
	if transactionsRoot, found := proxy[`transactionsRoot`]; found {
		err = json.Unmarshal(transactionsRoot, &b.TransactionsRoot)
		if err != nil {
			return err
		}
	}
	if stateRoot, found := proxy[`stateRoot`]; found {
		err = json.Unmarshal(stateRoot, &b.StateRoot)
		if err != nil {
			return err
		}
	}
	if receiptsRoot, found := proxy[`receiptsRoot`]; found {
		err = json.Unmarshal(receiptsRoot, &b.ReceiptsRoot)
		if err != nil {
			return err
		}
	}
	if miner, found := proxy[`miner`]; found {
		err = json.Unmarshal(miner, &b.Miner)
		if err != nil {
			return err
		}
	}
	if baseFeePerGas, found := proxy[`baseFeePerGas`]; found {
		var baseFeePerGasStr string
		err = json.Unmarshal(baseFeePerGas, &baseFeePerGasStr)
		if err != nil {
			return err
		}
		b.BaseFeePerGas, err = hexnum.ParseBigInt(baseFeePerGasStr)
		if err != nil {
			return err
		}
	}
	if difficulty, found := proxy[`difficulty`]; found {
		var difficultyStr string
		err = json.Unmarshal(difficulty, &difficultyStr)
		if err != nil {
			return err
		}
		b.Difficulty, err = hexnum.ParseHexInt64(difficultyStr)
		if err != nil {
			return err
		}
	}
	if totalDifficulty, found := proxy[`totalDifficulty`]; found {
		err = json.Unmarshal(totalDifficulty, &b.TotalDifficulty)
		if err != nil {
			return err
		}
	}
	if extraData, found := proxy[`extraData`]; found {
		err = json.Unmarshal(extraData, &b.ExtraData)
		if err != nil {
			return err
		}
	}
	if size, found := proxy[`size`]; found {
		var sizeStr string
		err = json.Unmarshal(size, &sizeStr)
		if err != nil {
			return err
		}
		b.Size, err = hexnum.ParseHexInt64(sizeStr)
		if err != nil {
			return err
		}
	}
	if gasLimit, found := proxy[`gasLimit`]; found {
		var gasLimitStr string
		err = json.Unmarshal(gasLimit, &gasLimitStr)
		if err != nil {
			return err
		}
		b.GasLimit, err = hexnum.ParseHexInt64(gasLimitStr)
		if err != nil {
			return err
		}
	}
	if gasUsed, found := proxy[`gasUsed`]; found {
		var gasUsedStr string
		err = json.Unmarshal(gasUsed, &gasUsedStr)
		if err != nil {
			return err
		}
		b.GasUsed, err = hexnum.ParseHexInt64(gasUsedStr)
		if err != nil {
			return err
		}
	}
	if timestamp, found := proxy[`timestamp`]; found {
		var timestampStr string
		err = json.Unmarshal(timestamp, &timestampStr)
		if err != nil {
			return err
		}
		b.Timestamp, err = hexnum.ParseHexInt64(timestampStr)
		if err != nil {
			return err
		}
	}
	if blobGasUsed, found := proxy[`blobGasUsed`]; found {
		err = json.Unmarshal(blobGasUsed, &b.BlobGasUsed)
		if err != nil {
			return err
		}
	}
	if excessBlobGas, found := proxy[`excessBlobGas`]; found {
		err = json.Unmarshal(excessBlobGas, &b.ExcessBlobGas)
		if err != nil {
			return err
		}
	}
	if mixHash, found := proxy[`mixHash`]; found {
		err = json.Unmarshal(mixHash, &b.MixHash)
		if err != nil {
			return err
		}
	}

	// Since we don't know in advance whether the block was requested
	// with full transactions or just headers (hashes), we cannot decode
	// them correctly at this stage. So let's leave the []json.RawMessage
	// array for processing later
	if transactions, found := proxy[`transactions`]; found {
		err = json.Unmarshal(transactions, &b.Transactions)
		if err != nil {
			return err
		}
		if b.FullTransactions {
			// If we have full transactions, we can decode them right away into private properties
			err = json.Unmarshal(transactions, &b.transactionsFullDecoded)
			if err != nil {
				return err
			}
			b.transactionsHashesDecoded = make([]string, len(b.transactionsFullDecoded))
			for i, tx := range b.transactionsFullDecoded {
				b.transactionsHashesDecoded[i] = tx.Hash
			}
		} else {
			// otherwise we can only decode hashes
			err = json.Unmarshal(transactions, &b.transactionsHashesDecoded)
			if err != nil {
				return err
			}
		}
	}
	if uncles, found := proxy[`uncles`]; found {
		err = json.Unmarshal(uncles, &b.Uncles)
		if err != nil {
			return err
		}
	}
	if withdrawals, found := proxy[`withdrawals`]; found {
		err = json.Unmarshal(withdrawals, &b.Withdrawals)
		if err != nil {
			return err
		}
	}
	if withdrawalsRoot, found := proxy[`withdrawalsRoot`]; found {
		err = json.Unmarshal(withdrawalsRoot, &b.WithdrawalsRoot)
		if err != nil {
			return err
		}
	}
	return nil
}
