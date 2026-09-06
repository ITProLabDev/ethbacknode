package eventlog

import (
	"math/big"
	"strings"

	"github.com/ITProLabDev/ethbacknode/abi"
	"github.com/ITProLabDev/ethbacknode/tools/log"
)

// ethTransaction is the minimal transaction shape eventlog needs to decide
// and deliver a contractTransaction notification: enough to know who it was
// sent to and decode its call, not a general-purpose transaction type.
type ethTransaction struct {
	Hash  string
	From  string
	To    string // empty for a contract creation; never a subscribed contract
	Value *big.Int
	Gas   int64
	Input []byte // calldata
}

// TransactionSource returns every transaction in a block. A separate seam
// from LogSource, rather than folded into it, because it is only needed when
// a TransactionSink is wired: a deployment that only wants contractEvent
// never pays for it.
type TransactionSource interface {
	GetBlockTransactions(blockNum int64) ([]*ethTransaction, error)
}

// TransactionDecoder identifies the method a transaction's calldata invokes
// on a contract and decodes its inputs. Satisfied by an adapter over
// *abi.SmartContractsManager.DecodeCall (and by a fake in tests). Unlike
// Decoder, an unknown selector is not an error -- see abi.DecodeCall.
type TransactionDecoder interface {
	DecodeCall(contractAddress string, data []byte) (method string, inputs []abi.DecodedValue, err error)
}

// ContractTransaction is a transaction sent directly to a subscribed
// contract, decoded by method selector, plus the block/receipt context
// needed for delivery. It is the unit handed to a TransactionSink.
type ContractTransaction struct {
	ServiceID       string
	Contract        string
	BlockNumber     int64
	TransactionHash string
	From            string
	To              string
	Value           *big.Int
	Gas             int64
	GasUsed         int64
	Success         bool

	// Method is the decoded method name, the raw 4-byte selector as 0x-hex
	// when it matches no registered method, or "" when the transaction
	// carries no calldata (a plain value transfer). See abi.DecodeCall.
	Method string
	Inputs []abi.DecodedValue

	// Data is the raw calldata, always included regardless of whether Method
	// decoded: a subscriber that wants to decode an unknown selector itself
	// (or double-check a decoded one) needs the bytes either way.
	Data []byte
}

// TransactionSink receives contractTransaction notifications for delivery.
// May be called concurrently, like Sink.
type TransactionSink func(*ContractTransaction)

func WithTransactionSource(s TransactionSource) Option {
	return func(svc *Service) { svc.txSource = s }
}
func WithTransactionDecoder(d TransactionDecoder) Option {
	return func(svc *Service) { svc.txDecoder = d }
}
func WithTransactionSink(s TransactionSink) Option { return func(svc *Service) { svc.txSink = s } }

// collectAndDeliverTransactions fetches every transaction in the block and
// delivers a contractTransaction notification for each one sent directly to
// a subscribed contract. Only whole_contract subscriptions receive it in v1
// -- see todo/TASKS.md Milestone 8.
func (s *Service) collectAndDeliverTransactions(blockNum int64, addrs []string) {
	txs, err := s.txSource.GetBlockTransactions(blockNum)
	if err != nil {
		log.Error("eventlog: collect transactions for block", blockNum, "error:", err)
		return
	}
	wanted := make(map[string]bool, len(addrs))
	for _, a := range addrs {
		wanted[strings.ToLower(a)] = true
	}
	for _, tx := range txs {
		if tx.To == "" || !wanted[strings.ToLower(tx.To)] {
			continue
		}
		s.decodeAndDeliverTransaction(blockNum, tx)
	}
}

// decodeAndDeliverTransaction decodes one transaction's calldata and
// delivers it to every whole_contract subscription for its recipient.
func (s *Service) decodeAndDeliverTransaction(blockNum int64, tx *ethTransaction) {
	var targets []*Subscription
	for _, sub := range s.subs.forAddress(tx.To) {
		if sub.Scope == ScopeWholeContract {
			targets = append(targets, sub)
		}
	}
	if len(targets) == 0 {
		return
	}

	method, inputs, err := s.txDecoder.DecodeCall(tx.To, tx.Input)
	if err != nil {
		log.Error("eventlog: decode transaction", tx.Hash, "error:", err)
		return
	}

	receipt, err := s.source.GetTransactionReceipt(tx.Hash)
	if err != nil {
		log.Error("eventlog: receipt for transaction", tx.Hash, "error:", err)
		return
	}
	var gasUsed int64
	var success bool
	if receipt != nil {
		gasUsed = receipt.GasUsed
		success = receipt.Status
	}

	for _, sub := range targets {
		s.txSink(&ContractTransaction{
			ServiceID:       sub.ServiceID,
			Contract:        tx.To,
			BlockNumber:     blockNum,
			TransactionHash: tx.Hash,
			From:            tx.From,
			To:              tx.To,
			Value:           tx.Value,
			Gas:             tx.Gas,
			GasUsed:         gasUsed,
			Success:         success,
			Method:          method,
			Inputs:          inputs,
			Data:            tx.Input,
		})
	}
}
