package endpoint_test

// End-to-end test for the contract-transaction delivery path (todo/TASKS.md
// Milestone 8). It wires the REAL components the way main.go does —
// abi.SmartContractsManager (decode by method selector), eventlog.Service
// (collect block transactions + scope filter), and
// endpoint.NewContractTransactionSink — then replays a block's transaction
// through the real stack and asserts the delivered contractTransaction wire
// JSON: the exact payload a client receives over its webhook.
//
// Lives in package endpoint_test (not endpoint), matching
// e2e_contract_event_test.go, and reuses its memBin/hexOf/leftPad32 helpers
// from that file (same package, same test binary).

import (
	"encoding/json"
	"math/big"
	"sync"
	"testing"

	"github.com/ITProLabDev/ethbacknode/abi"
	"github.com/ITProLabDev/ethbacknode/clients/ethclient"
	"github.com/ITProLabDev/ethbacknode/endpoint"
	"github.com/ITProLabDev/ethbacknode/eventlog"
)

const transferABI = `[{"type":"function","name":"transfer","stateMutability":"nonpayable",` +
	`"inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],` +
	`"outputs":[{"name":"","type":"bool"}]}]`

// capturedTx is one delivered contractTransaction, captured at the notifier
// as the (serviceID, subject, payload) the subscriptions layer would send.
type capturedTx struct {
	serviceID int64
	subject   string
	payload   interface{}
}

// buildTransactionService wires a real eventlog.Service with both the log and
// transaction data paths, exactly as main.go does, capturing every delivered
// contractTransaction.
func buildTransactionService(abiManager *abi.SmartContractsManager, txs []eventlog.RawTransaction, receipts map[string]eventlog.RawReceipt) (*eventlog.Service, *[]capturedTx, *sync.Mutex) {
	var mu sync.Mutex
	var captured []capturedTx
	sink := endpoint.NewContractTransactionSink(endpoint.NotifierFunc(
		func(serviceID int64, subject string, payload interface{}) {
			mu.Lock()
			captured = append(captured, capturedTx{serviceID, subject, payload})
			mu.Unlock()
		}))

	svc := eventlog.New(
		eventlog.WithLogSource(eventlog.RawLogSource{
			GetLogsFn: func(from, to int64, addresses, topics []string) ([]eventlog.RawLog, error) {
				return nil, nil
			},
			GetReceiptFn: func(txHash string) (*eventlog.RawReceipt, error) {
				r := receipts[txHash]
				return &r, nil
			},
		}),
		eventlog.WithDecoder(eventlog.NewDecoder(abiManager.DecodeLog)),
		eventlog.WithTransactionSource(eventlog.RawTransactionSource{
			GetBlockTransactionsFn: func(blockNum int64) ([]eventlog.RawTransaction, error) {
				return txs, nil
			},
		}),
		eventlog.WithTransactionDecoder(eventlog.NewTransactionDecoder(abiManager.DecodeCall)),
		eventlog.WithTransactionSink(sink),
		eventlog.WithSink(func(*eventlog.ContractEvent) {}),
	)
	return svc, &captured, &mu
}

func TestE2E_ContractTransaction_DeliveredEndToEnd(t *testing.T) {
	abiManager := abi.NewManager(abi.WithStorage(&memBin{}), abi.WithAddressCodec(ethclient.GetAddressCodec()))
	if err := abiManager.Init(); err != nil {
		t.Fatalf("abi Init: %v", err)
	}
	const contractAddr = "0xCONTRACT00000000000000000000000000000009"
	if err := abiManager.AddContractFromABI("Token", "TOK", contractAddr, []byte(transferABI)); err != nil {
		t.Fatalf("register contract: %v", err)
	}

	contract, err := abiManager.GetSmartContractByAddress(contractAddr)
	if err != nil {
		t.Fatalf("lookup contract: %v", err)
	}
	entry, err := contract.Abi.GetMethodByName("transfer")
	if err != nil {
		t.Fatalf("lookup method: %v", err)
	}
	to := make([]byte, 20)
	to[19] = 0x42
	calldata, err := entry.EncodeInputsTyped(to, big.NewInt(1000))
	if err != nil {
		t.Fatalf("EncodeInputsTyped: %v", err)
	}

	tx := eventlog.RawTransaction{
		Hash:  "0xtx1",
		From:  "0xfrom0000000000000000000000000000000001",
		To:    contractAddr,
		Value: big.NewInt(0),
		Gas:   100000,
		Input: calldata,
	}
	receipts := map[string]eventlog.RawReceipt{
		"0xtx1": {Status: true, GasUsed: 51423},
	}

	t.Run("whole_contract delivers the decoded call with safe JSON", func(t *testing.T) {
		svc, captured, mu := buildTransactionService(abiManager, []eventlog.RawTransaction{tx}, receipts)
		if err := svc.SubscribeAndSaveStrings("42", contractAddr, "whole_contract"); err != nil {
			t.Fatalf("subscribe: %v", err)
		}
		svc.OnBlock(20123456, "0xblock")

		mu.Lock()
		got := append([]capturedTx{}, (*captured)...)
		mu.Unlock()
		if len(got) != 1 {
			t.Fatalf("delivered %d contractTransactions, want 1", len(got))
		}
		ct := got[0]
		if ct.serviceID != 42 || ct.subject != "contractTransaction" {
			t.Fatalf("serviceID=%d subject=%q", ct.serviceID, ct.subject)
		}

		b, err := json.Marshal(ct.payload)
		if err != nil {
			t.Fatal(err)
		}
		var wire struct {
			Contract string `json:"contract"`
			BlockNum int64  `json:"blockNum"`
			TxHash   string `json:"txHash"`
			From     string `json:"from"`
			To       string `json:"to"`
			Value    string `json:"value"`
			Gas      int64  `json:"gas"`
			GasUsed  int64  `json:"gasUsed"`
			Success  bool   `json:"success"`
			Method   string `json:"method"`
			Inputs   []struct {
				Name  string          `json:"name"`
				Type  string          `json:"type"`
				Value json.RawMessage `json:"value"`
			} `json:"inputs"`
			Data string `json:"data"`
		}
		if err := json.Unmarshal(b, &wire); err != nil {
			t.Fatalf("payload not valid JSON: %v\n%s", err, b)
		}
		if wire.Contract != contractAddr || wire.BlockNum != 20123456 || wire.TxHash != "0xtx1" {
			t.Fatalf("context wrong: %+v", wire)
		}
		if wire.From != tx.From || wire.To != contractAddr || wire.Value != "0" || wire.Gas != 100000 {
			t.Fatalf("transaction fields wrong: %+v", wire)
		}
		if wire.GasUsed != 51423 || !wire.Success {
			t.Fatalf("receipt fields wrong: %+v", wire)
		}
		if wire.Method != "transfer" || len(wire.Inputs) != 2 {
			t.Fatalf("decoded call wrong: %+v", wire)
		}
		if wire.Data != "0x"+hexOf(calldata) {
			t.Fatalf("data=%s want raw calldata as 0x-hex", wire.Data)
		}
	})

	t.Run("unknown selector still delivers, with the raw hex selector and calldata", func(t *testing.T) {
		unknown := append([]byte{0xde, 0xad, 0xbe, 0xef}, make([]byte, 32)...)
		badTx := eventlog.RawTransaction{Hash: "0xtx2", From: tx.From, To: contractAddr, Value: big.NewInt(0), Gas: 21000, Input: unknown}
		svc, captured, mu := buildTransactionService(abiManager, []eventlog.RawTransaction{badTx},
			map[string]eventlog.RawReceipt{"0xtx2": {Status: true}})
		if err := svc.SubscribeAndSaveStrings("42", contractAddr, "whole_contract"); err != nil {
			t.Fatalf("subscribe: %v", err)
		}
		svc.OnBlock(1, "0xblock")

		mu.Lock()
		got := append([]capturedTx{}, (*captured)...)
		mu.Unlock()
		if len(got) != 1 {
			t.Fatalf("delivered %d contractTransactions, want 1 (never skipped for being undecodable)", len(got))
		}
		b, err := json.Marshal(got[0].payload)
		if err != nil {
			t.Fatal(err)
		}
		var wire struct {
			Method string          `json:"method"`
			Inputs json.RawMessage `json:"inputs"`
			Data   string          `json:"data"`
		}
		if err := json.Unmarshal(b, &wire); err != nil {
			t.Fatalf("payload not valid JSON: %v\n%s", err, b)
		}
		if wire.Method != "0xdeadbeef" {
			t.Fatalf("method=%q want the raw hex selector", wire.Method)
		}
		if string(wire.Inputs) != "[]" && string(wire.Inputs) != "null" {
			t.Fatalf("inputs=%s want empty", wire.Inputs)
		}
		if wire.Data != "0x"+hexOf(unknown) {
			t.Fatalf("data=%s want the raw calldata", wire.Data)
		}
	})

	t.Run("managed_only subscription does not receive contractTransaction", func(t *testing.T) {
		svc, captured, mu := buildTransactionService(abiManager, []eventlog.RawTransaction{tx}, receipts)
		if err := svc.SubscribeAndSaveStrings("43", contractAddr, "managed_only"); err != nil {
			t.Fatalf("subscribe: %v", err)
		}
		svc.OnBlock(20123456, "0xblock")

		mu.Lock()
		defer mu.Unlock()
		if len(*captured) != 0 {
			t.Fatalf("managed_only must not receive contractTransaction today, got %d", len(*captured))
		}
	})
}
