package eventlog

import (
	"fmt"
	"math/big"

	"github.com/ITProLabDev/ethbacknode/abi"
)

// decoderFunc adapts a plain function to the Decoder interface, so main.go can
// wire abi.(*SmartContractsManager).DecodeLog without a named wrapper type.
type decoderFunc func(contractAddress string, topics [][32]byte, data []byte) (*abi.DecodedEvent, error)

func (f decoderFunc) DecodeLog(contractAddress string, topics [][32]byte, data []byte) (*abi.DecodedEvent, error) {
	return f(contractAddress, topics, data)
}

// NewDecoder wraps any DecodeLog-shaped function (e.g. the abi manager's
// DecodeLog method) as a Decoder.
func NewDecoder(fn func(contractAddress string, topics [][32]byte, data []byte) (*abi.DecodedEvent, error)) Decoder {
	return decoderFunc(fn)
}

// RawLog is the ethclient-shaped log the LogSource adapter receives. main.go
// converts clients/ethclient.Log values into RawLog so eventlog stays free of
// an ethclient import.
type RawLog struct {
	Address          string
	Topics           []string
	Data             []byte
	BlockNumber      int64
	TransactionHash  string
	TransactionIndex int64
	LogIndex         int64
	Removed          bool
}

// RawReceipt is the ethclient-shaped receipt the LogSource adapter receives:
// its logs, plus the outcome (Status/GasUsed) contractTransaction delivery
// needs and log collection does not.
type RawReceipt struct {
	Logs    []RawLog
	Status  bool
	GasUsed int64
}

// RawLogSource adapts ethclient-shaped functions to the eventlog LogSource
// interface. main.go provides the function fields backed by *ethclient.Client.
type RawLogSource struct {
	GetLogsFn    func(fromBlock, toBlock int64, addresses, topics []string) ([]RawLog, error)
	GetReceiptFn func(txHash string) (*RawReceipt, error)
}

func (r RawLogSource) GetLogs(filter logFilter) ([]*ethLog, error) {
	if r.GetLogsFn == nil {
		return nil, fmt.Errorf("eventlog: RawLogSource.GetLogsFn not configured")
	}
	raw, err := r.GetLogsFn(filter.FromBlock, filter.ToBlock, filter.Addresses, filter.Topics)
	if err != nil {
		return nil, err
	}
	return rawToEthLogs(raw), nil
}

func (r RawLogSource) GetTransactionReceipt(txHash string) (*ethReceipt, error) {
	if r.GetReceiptFn == nil {
		return nil, fmt.Errorf("eventlog: RawLogSource.GetReceiptFn not configured")
	}
	raw, err := r.GetReceiptFn(txHash)
	if err != nil {
		return nil, err
	}
	return &ethReceipt{Logs: rawToEthLogs(raw.Logs), Status: raw.Status, GasUsed: raw.GasUsed}, nil
}

// RawTransaction is the ethclient-shaped transaction the TransactionSource
// adapter receives. main.go converts clients/ethclient.Transaction values
// into RawTransaction so eventlog stays free of an ethclient import.
type RawTransaction struct {
	Hash  string
	From  string
	To    string
	Value *big.Int
	Gas   int64
	Input []byte // calldata, already hex-decoded
}

// RawTransactionSource adapts an ethclient-shaped function to the eventlog
// TransactionSource interface. main.go provides the function field backed by
// *ethclient.Client.
type RawTransactionSource struct {
	GetBlockTransactionsFn func(blockNum int64) ([]RawTransaction, error)
}

func (r RawTransactionSource) GetBlockTransactions(blockNum int64) ([]*ethTransaction, error) {
	if r.GetBlockTransactionsFn == nil {
		return nil, fmt.Errorf("eventlog: RawTransactionSource.GetBlockTransactionsFn not configured")
	}
	raw, err := r.GetBlockTransactionsFn(blockNum)
	if err != nil {
		return nil, err
	}
	out := make([]*ethTransaction, len(raw))
	for i, t := range raw {
		out[i] = &ethTransaction{Hash: t.Hash, From: t.From, To: t.To, Value: t.Value, Gas: t.Gas, Input: t.Input}
	}
	return out, nil
}

// transactionDecoderFunc adapts a plain function to the TransactionDecoder
// interface, so main.go can wire abi.(*SmartContractsManager).DecodeCall
// without a named wrapper type.
type transactionDecoderFunc func(contractAddress string, data []byte) (string, []abi.DecodedValue, error)

func (f transactionDecoderFunc) DecodeCall(contractAddress string, data []byte) (string, []abi.DecodedValue, error) {
	return f(contractAddress, data)
}

// NewTransactionDecoder wraps any DecodeCall-shaped function (e.g. the abi
// manager's DecodeCall method) as a TransactionDecoder.
func NewTransactionDecoder(fn func(contractAddress string, data []byte) (string, []abi.DecodedValue, error)) TransactionDecoder {
	return transactionDecoderFunc(fn)
}

func rawToEthLogs(raw []RawLog) []*ethLog {
	out := make([]*ethLog, len(raw))
	for i, l := range raw {
		out[i] = &ethLog{
			Address:          l.Address,
			Topics:           l.Topics,
			Data:             l.Data,
			BlockNumber:      l.BlockNumber,
			TransactionHash:  l.TransactionHash,
			TransactionIndex: l.TransactionIndex,
			LogIndex:         l.LogIndex,
			Removed:          l.Removed,
		}
	}
	return out
}
