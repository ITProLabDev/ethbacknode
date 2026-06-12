package endpoint

import (
	"strconv"

	"github.com/ITProLabDev/ethbacknode/abi"
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

// NotifierFunc adapts a function to ContractEventNotifier.
type NotifierFunc func(serviceID int64, subject string, payload interface{})

func (f NotifierFunc) NotifyContractEvent(serviceID int64, subject string, payload interface{}) {
	f(serviceID, subject, payload)
}
