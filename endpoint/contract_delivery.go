package endpoint

import (
	"strconv"

	"github.com/ITProLabDev/ethbacknode/abi"
	"github.com/ITProLabDev/ethbacknode/common/hexnum"
	"github.com/ITProLabDev/ethbacknode/eventlog"
	"github.com/ITProLabDev/ethbacknode/tools/log"
)

// ContractEventNotifier delivers a contract-event payload to a subscriber.
// Satisfied by an adapter over subscriptions.Manager (wired in main.go).
type ContractEventNotifier interface {
	NotifyContractEvent(serviceID int64, subject string, payload interface{})
}

// contractEventPayload is the JSON-RPC notification body for a contractEvent.
type contractEventPayload struct {
	Event    string             `json:"event"`             // event name
	Contract string             `json:"contract"`          // contract address
	BlockNum int64              `json:"blockNum"`          // block number
	TxHash   string             `json:"txHash"`            // transaction hash
	TxIndex  int64              `json:"txIndex"`           // tx index in block
	LogIndex int64              `json:"logIndex"`          // log index in block
	Removed  bool               `json:"removed,omitempty"` // true if reverted by reorg
	Inputs   []abi.DecodedValue `json:"inputs"`            // decoded event parameters
}

// NewContractEventSink builds an eventlog.Sink that routes each decoded
// contract event to its subscriber via the notifier. It is concurrency-safe as
// long as the notifier is (subscriptions.Manager is).
func NewContractEventSink(notifier ContractEventNotifier) eventlog.Sink {
	return func(ce *eventlog.ContractEvent) {
		serviceID, err := strconv.ParseInt(ce.ServiceID, 10, 64)
		if err != nil {
			log.Error("contractEvent: bad serviceID", ce.ServiceID, ":", err)
			return
		}
		payload := &contractEventPayload{
			Event:    ce.Event.Name,
			Contract: ce.Event.Contract,
			BlockNum: ce.BlockNumber,
			TxHash:   ce.TransactionHash,
			TxIndex:  ce.TransactionIndex,
			LogIndex: ce.LogIndex,
			Removed:  ce.Removed,
			Inputs:   ce.Event.Inputs,
		}
		notifier.NotifyContractEvent(serviceID, "contractEvent", payload)
	}
}

// contractTransactionPayload is the JSON-RPC notification body for a
// contractTransaction (todo/TASKS.md Milestone 8): a transaction sent
// directly to a subscribed contract, decoded by method selector.
type contractTransactionPayload struct {
	Contract string `json:"contract"`
	BlockNum int64  `json:"blockNum"`
	TxHash   string `json:"txHash"`
	From     string `json:"from"`
	To       string `json:"to"`
	// Value is a decimal string, not a JSON number -- wei amounts routinely
	// exceed 2^53 and would lose precision through JS's JSON.parse, the same
	// reason abi.DecodedValue's big integers are strings (see M7 in
	// todo/TASKS.md).
	Value   string `json:"value"`
	Gas     int64  `json:"gas"`
	GasUsed int64  `json:"gasUsed"`
	Success bool   `json:"success"`
	// Method is the decoded method name, the raw 4-byte selector as 0x-hex
	// when it matches no registered method, or "" for a plain value transfer
	// with no calldata. See abi.DecodeCall.
	Method string             `json:"method"`
	Inputs []abi.DecodedValue `json:"inputs"`
	// Data is the raw calldata as 0x-hex, always included regardless of
	// whether Method decoded -- a subscriber that wants to decode an unknown
	// selector itself (or double-check a decoded one) needs the bytes either
	// way.
	Data string `json:"data"`
}

// NewContractTransactionSink builds an eventlog.TransactionSink that routes
// each contractTransaction to its subscriber via the notifier.
func NewContractTransactionSink(notifier ContractEventNotifier) eventlog.TransactionSink {
	return func(ct *eventlog.ContractTransaction) {
		serviceID, err := strconv.ParseInt(ct.ServiceID, 10, 64)
		if err != nil {
			log.Error("contractTransaction: bad serviceID", ct.ServiceID, ":", err)
			return
		}
		value := "0"
		if ct.Value != nil {
			value = ct.Value.String()
		}
		payload := &contractTransactionPayload{
			Contract: ct.Contract,
			BlockNum: ct.BlockNumber,
			TxHash:   ct.TransactionHash,
			From:     ct.From,
			To:       ct.To,
			Value:    value,
			Gas:      ct.Gas,
			GasUsed:  ct.GasUsed,
			Success:  ct.Success,
			Method:   ct.Method,
			Inputs:   ct.Inputs,
			Data:     hexnum.BytesToHex(ct.Data),
		}
		notifier.NotifyContractEvent(serviceID, "contractTransaction", payload)
	}
}

// NotifierFunc adapts a function to ContractEventNotifier.
type NotifierFunc func(serviceID int64, subject string, payload interface{})

func (f NotifierFunc) NotifyContractEvent(serviceID int64, subject string, payload interface{}) {
	f(serviceID, subject, payload)
}
