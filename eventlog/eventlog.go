package eventlog

import (
	"errors"

	"github.com/ITProLabDev/ethbacknode/abi"
	"github.com/ITProLabDev/ethbacknode/tools/log"
)

// errNoEvent is returned by a decoder when a log does not match a known event.
var errNoEvent = errors.New("no matching event")

// ethLog is the minimal log shape eventlog needs. It mirrors
// clients/ethclient.Log; the adapter in main.go converts between them.
type ethLog struct {
	Address          string
	Topics           []string
	Data             []byte
	BlockNumber      int64
	TransactionHash  string
	TransactionIndex int64
	LogIndex         int64
	// Removed is true if the log was reverted by a chain reorg. Carried through
	// so M6 delivery can distinguish reverted events from canonical ones.
	Removed bool
}

// topics32 converts the 0x-hex topics to [32]byte for the decoder.
func (l *ethLog) topics32() ([][32]byte, error) {
	out := make([][32]byte, len(l.Topics))
	for i, t := range l.Topics {
		b, err := parseTopic(t)
		if err != nil {
			return nil, err
		}
		out[i] = b
	}
	return out, nil
}

// ethReceipt is the minimal receipt shape eventlog needs.
type ethReceipt struct {
	Logs []*ethLog
}

// logFilter mirrors clients/ethclient.LogFilter for the LogSource interface.
type logFilter struct {
	FromBlock int64
	ToBlock   int64
	Addresses []string
	Topics    []string
}

// LogSource is the narrow chain-access surface eventlog needs. *ethclient.Client
// satisfies an adapter of this (wired in main.go); tests use a fake.
type LogSource interface {
	GetLogs(filter logFilter) ([]*ethLog, error)
	GetTransactionReceipt(txHash string) (*ethReceipt, error)
}

// Decoder decodes a single log into a DecodedEvent. Satisfied by an adapter
// over *abi.SmartContractsManager.DecodeLog (and by a fake in tests).
type Decoder interface {
	DecodeLog(contractAddress string, topics [][32]byte, data []byte) (*abi.DecodedEvent, error)
}

// Sink receives decoded, scope-filtered contract events for delivery (M6).
// It may be called concurrently from multiple goroutines (the watchdog
// dispatches block listeners in their own goroutines), so implementations
// must be safe for concurrent use.
type Sink func(*ContractEvent)

// Service collects, decodes, and scope-filters contract event logs per block.
type Service struct {
	source  LogSource
	decoder Decoder
	managed ManagedAddresses
	codec   AddressEncoder
	sink    Sink
	cfg     Config
	subs    *subscriptionSet

	// blockTxHashes returns the tx hashes in a block (Mode B). Wired in main.go
	// from the chain client; a test seam in unit tests.
	blockTxHashes func(blockNum int64) ([]string, error)
}

// Option configures a Service.
type Option func(*Service)

func WithLogSource(s LogSource) Option         { return func(svc *Service) { svc.source = s } }
func WithDecoder(d Decoder) Option             { return func(svc *Service) { svc.decoder = d } }
func WithManaged(m ManagedAddresses) Option    { return func(svc *Service) { svc.managed = m } }
func WithAddressCodec(c AddressEncoder) Option { return func(svc *Service) { svc.codec = c } }
func WithSink(s Sink) Option                   { return func(svc *Service) { svc.sink = s } }
func WithConfig(c Config) Option               { return func(svc *Service) { svc.cfg = c } }

// New builds a Service from options.
func New(opts ...Option) *Service {
	svc := &Service{cfg: DefaultConfig(), subs: newSubscriptionSet()}
	for _, o := range opts {
		o(svc)
	}
	return svc
}

// Subscribe registers a contract-event subscription.
func (s *Service) Subscribe(sub *Subscription) { s.subs.add(sub) }

// OnBlock is the watchdog block-listener entry point: collect the block's logs,
// decode them, and deliver scope-matched events to the sink.
func (s *Service) OnBlock(blockNum int64, blockID string) {
	addrs := s.subs.addresses()
	if len(addrs) == 0 {
		return // nothing subscribed; do no chain work
	}
	logs, err := s.collect(blockNum, addrs)
	if err != nil {
		log.Error("eventlog: collect block", blockNum, "error:", err)
		return
	}
	for _, lg := range logs {
		s.decodeAndDeliver(lg)
	}
}

// decodeAndDeliver decodes one log and delivers it to every matching
// subscription for its contract address.
func (s *Service) decodeAndDeliver(lg *ethLog) {
	subs := s.subs.forAddress(lg.Address)
	if len(subs) == 0 {
		return
	}
	topics, err := lg.topics32()
	if err != nil {
		return // malformed topic; skip
	}
	ev, err := s.decoder.DecodeLog(lg.Address, topics, lg.Data)
	if err != nil {
		return // unknown event / undecodable; skip
	}
	for _, sub := range subs {
		if eventMatchesScope(ev, sub.Scope, s.managed, s.codec) {
			s.sink(&ContractEvent{
				Event:            ev,
				ServiceID:        sub.ServiceID,
				BlockNumber:      lg.BlockNumber,
				TransactionHash:  lg.TransactionHash,
				TransactionIndex: lg.TransactionIndex,
				LogIndex:         lg.LogIndex,
				Removed:          lg.Removed,
			})
		}
	}
}
