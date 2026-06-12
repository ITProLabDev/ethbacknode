package eventlog

import (
	"fmt"

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
	Address         string
	Topics          []string
	Data            []byte
	BlockNumber     int64
	TransactionHash string
	LogIndex        int64
}

// RawLogSource adapts ethclient-shaped functions to the eventlog LogSource
// interface. main.go provides the function fields backed by *ethclient.Client.
type RawLogSource struct {
	GetLogsFn    func(fromBlock, toBlock int64, addresses, topics []string) ([]RawLog, error)
	GetReceiptFn func(txHash string) (logs []RawLog, err error)
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
	return &ethReceipt{Logs: rawToEthLogs(raw)}, nil
}

func rawToEthLogs(raw []RawLog) []*ethLog {
	out := make([]*ethLog, len(raw))
	for i, l := range raw {
		out[i] = &ethLog{
			Address:         l.Address,
			Topics:          l.Topics,
			Data:            l.Data,
			BlockNumber:     l.BlockNumber,
			TransactionHash: l.TransactionHash,
			LogIndex:        l.LogIndex,
		}
	}
	return out
}
