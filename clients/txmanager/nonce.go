// Package txmanager owns the life of a transaction this node broadcasts,
// starting with which nonce it is signed with. clients/ethclient can sign and
// send, but its knowledge ends when a call returns; a node that sends more
// than one transaction per address at a time needs something that remembers
// across calls, and works from that record rather than from the node's
// momentary view. See docs/CHAIN_LAYER_TODO.md (T1.1) and
// docs/PROJECT_STATUS.md for why this exists.
package txmanager

import (
	"sync"

	"github.com/ITProLabDev/ethbacknode/tools/log"
)

// Option configures a NonceManager.
type Option func(*NonceManager)

// WithChain sets the chain the allocator seeds nonces from. Required.
func WithChain(chain ChainNonces) Option {
	return func(m *NonceManager) { m.chain = chain }
}

// WithStore sets where issued nonces are persisted, so a restart does not
// re-issue one. Required.
func WithStore(store Store) Option {
	return func(m *NonceManager) { m.recs = newRecords(store) }
}

// NewNonceManager builds a NonceManager from options. WithChain and WithStore
// are both required for Allocate to work; a NonceManager missing either
// returns ErrChainNotSet / ErrStoreNotSet rather than panicking, so a caller
// that forgets one at wiring time gets a clear error at first use.
func NewNonceManager(opts ...Option) *NonceManager {
	m := &NonceManager{state: make(map[string]*addressNonces)}
	for _, o := range opts {
		o(m)
	}
	return m
}

// NonceManager hands out nonces for the addresses this node signs from, and
// implements ethclient.NonceSource (Allocate/Sent/Release).
//
// The node's pending count cannot do this job alone: it only advances once
// the node holds a transaction, so two sends prepared back to back are told
// the same number, and one is then rejected as a duplicate or silently
// replaces the other. This asks the node once per address and counts from
// there, persisting every nonce it issues so a restart does not re-issue one.
//
// State is per address and guarded per address, so seeding one address --
// which costs a round trip to the node -- does not hold up another.
type NonceManager struct {
	chain ChainNonces
	recs  *records

	mu    sync.Mutex
	state map[string]*addressNonces
}

// addressNonces is one address's cursor.
type addressNonces struct {
	mu sync.Mutex

	// next is the nonce to hand out when there is no hole to fill.
	next int64

	// seeded says whether next has been established from the chain and the
	// records.
	seeded bool

	// freed holds nonces that were issued and never reached the node, lowest
	// first. They are handed out before next advances any further: the chain
	// will not process a nonce above a gap, so a hole left unfilled stalls
	// every later transaction from this address.
	freed []int64
}

// Allocate returns the nonce to sign an address's next transaction with.
func (m *NonceManager) Allocate(from string) (int64, error) {
	if m.chain == nil {
		return 0, ErrChainNotSet
	}
	if m.recs == nil {
		return 0, ErrStoreNotSet
	}

	e := m.entry(from)
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.seeded {
		if err := m.seed(from, e); err != nil {
			return 0, err
		}
	}
	if len(e.freed) > 0 {
		nonce := e.freed[0]
		e.freed = e.freed[1:]
		return nonce, nil
	}
	nonce := e.next
	e.next++
	return nonce, nil
}

// Sent records a transaction the node accepted, under the nonce it was
// allocated. This is what makes the allocator survive a restart: the cursor
// is in memory, and the records are the only account of which nonces have
// been issued. A failure to save is logged loudly rather than swallowed --
// the transaction is already broadcast, so the consequence is not a lost send
// but a nonce this node will not remember issuing.
func (m *NonceManager) Sent(from, txHash string, nonce int64) {
	if m.recs == nil {
		return
	}
	if err := m.recs.save(from, txHash, nonce); err != nil {
		log.Error("txmanager: a broadcast transaction was not recorded:", "from:", from, "nonce:", nonce, "hash:", txHash, "error:", err)
	}
}

// Release takes back a nonce that was issued and never reached the node. The
// chain is still waiting for it, so handing the next transaction a higher
// number instead would leave a gap: nothing above a gap is processed until
// the gap is filled.
func (m *NonceManager) Release(from string, nonce int64) {
	e := m.entry(from)
	e.mu.Lock()
	defer e.mu.Unlock()

	switch {
	case !e.seeded:
		// Nothing was ever issued for this address, so there is nothing to
		// take back.
	case nonce+1 == e.next:
		// It was the most recent one out: step the cursor back over it.
		// Releasing several in descending order cascades through this branch
		// and leaves no holes behind.
		e.next = nonce
	case nonce < e.next:
		e.freeHole(nonce)
	default:
		// Above the cursor, so it was never issued here.
	}
}

// freeHole remembers a released nonce below the cursor, lowest first and
// without duplicates. Called with e.mu held.
func (e *addressNonces) freeHole(nonce int64) {
	at := 0
	for at < len(e.freed) && e.freed[at] < nonce {
		at++
	}
	if at < len(e.freed) && e.freed[at] == nonce {
		return
	}
	e.freed = append(e.freed, 0)
	copy(e.freed[at+1:], e.freed[at:])
	e.freed[at] = nonce
}

// entry returns an address's cursor, creating it on first use. Addresses are
// normalized so a checksummed and a lowercase spelling of one address share
// one cursor.
func (m *NonceManager) entry(from string) *addressNonces {
	key := normalizeAddress(from)
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.state[key]
	if !ok {
		e = new(addressNonces)
		m.state[key] = e
	}
	return e
}

// seed establishes an address's cursor. Called with e.mu held.
//
// It takes the higher of what the chain reports and one past the highest
// nonce on record, and needs both. The chain alone is not enough: a
// transaction still sitting in the pool leaves the pending count where it was
// after a restart, so seeding from the chain alone would re-issue a nonce
// already in flight. The records alone are not enough either: an address may
// have been used by something other than this node, and its history here
// would start below the chain's.
func (m *NonceManager) seed(from string, e *addressNonces) error {
	fromChain, err := m.chain.PendingNonceAt(from)
	if err != nil {
		return err
	}
	next := fromChain

	highest, found, err := m.recs.highestNonceFor(from)
	if err != nil {
		return err
	}
	if found && highest+1 > next {
		next = highest + 1
	}

	e.next = next
	e.seeded = true
	log.Debug("txmanager: nonce cursor seeded for", from, "- chain:", fromChain, "next:", next)
	return nil
}
