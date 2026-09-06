package watchdog

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ITProLabDev/ethbacknode/types"
)

// memPoolErrClient always fails MemPoolContent and signals every call on a
// channel, so a test can tell whether runLoop keeps calling it.
type memPoolErrClient struct {
	types.ChainClient
	calls chan struct{}
}

func (f *memPoolErrClient) GetChainName() string { return "test" }

func (f *memPoolErrClient) MemPoolContent() ([]*types.TransferInfo, error) {
	f.calls <- struct{}{}
	return nil, errors.New("mempool unavailable")
}

// runLoop took w.mux.Lock() at the top of its loop and, on a MemPoolContent
// error, slept and `continue`d without unlocking it first. The next iteration
// then blocked forever trying to re-acquire a lock it already held itself,
// silently stopping all chain monitoring after the first transient RPC error.
func TestRunLoopDoesNotDeadlockAfterMemPoolError(t *testing.T) {
	calls := make(chan struct{}, 4)
	w := &Service{
		client:        &memPoolErrClient{calls: calls},
		config:        &Config{},
		state:         &lastState{},
		events:        make(chan *event, 10),
		checkInterval: 0,
	}
	go w.runLoop()

	select {
	case <-calls:
	case <-time.After(2 * time.Second):
		t.Fatal("runLoop never called MemPoolContent")
	}
	// A second call proves the loop returned to the top of its for-loop after
	// the error, instead of self-deadlocking on w.mux.Lock().
	select {
	case <-calls:
	case <-time.After(2 * time.Second):
		t.Fatal("runLoop deadlocked after a MemPoolContent error (mux never unlocked before continue)")
	}
}

// catchUpClient reports a fixed chain head of 3 with an empty mempool, and
// fails BlockByNum for one configured block number exactly once (a transient
// RPC blip), succeeding on every other call including retries.
type catchUpClient struct {
	types.ChainClient
	mu           sync.Mutex
	failBlockNum int64
	failedOnce   bool
}

func (f *catchUpClient) GetChainName() string                           { return "test" }
func (f *catchUpClient) MemPoolContent() ([]*types.TransferInfo, error) { return nil, nil }
func (f *catchUpClient) BlockNum() (int64, error)                       { return 3, nil }
func (f *catchUpClient) BlockByNum(blockNum int64, fullInfo bool) (*types.BlockInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if blockNum == f.failBlockNum && !f.failedOnce {
		f.failedOnce = true
		return nil, errors.New("transient rpc blip")
	}
	return &types.BlockInfo{BlockID: fmt.Sprintf("0x%d", blockNum)}, nil
}

// getState reads w.state under w.mux, the same lock runLoop holds around
// every UpdateState call in its tick: lastState has no locking of its own, so
// a concurrent reader must synchronize through the lock runLoop already uses.
func getState(w *Service) int64 {
	w.mux.Lock()
	defer w.mux.Unlock()
	return w.state.GetState()
}

func waitForState(t *testing.T, w *Service, want int64, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if getState(w) == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("state never reached %d within %s (got %d)", want, timeout, getState(w))
}

// A block failing partway through a multi-block catch-up must not be marked
// processed. The old code unlocked w.mux inside the inner catch-up loop's
// error branch and `continue`d it (skipping straight to the next block
// instead of stopping), then unconditionally recorded state.UpdateState(currentBlock)
// at the bottom of the tick regardless of the failure -- silently marking the
// whole range, including the block that failed, as done. Falling through to
// that bottom Unlock also double-unlocks w.mux, which panics the process.
func TestRunLoopDoesNotLoseAFailedBlockInCatchUp(t *testing.T) {
	w := &Service{
		client:        &catchUpClient{failBlockNum: 2},
		config:        &Config{},
		state:         &lastState{},
		events:        make(chan *event, 10),
		checkInterval: 1,
	}
	go w.runLoop()

	// First tick: blocks 1 and 2 are attempted, 2 fails. Progress must stop at
	// 1, not silently advance to 3 (the chain head) as if block 2 (and the
	// block after it) had been processed.
	waitForState(t, w, 1, 2*time.Second)
	time.Sleep(200 * time.Millisecond) // give a buggy build time to overshoot to 3
	if got := getState(w); got != 1 {
		t.Fatalf("state.LastBlockNum = %d right after the failure, want 1 (block 2 must not be marked processed)", got)
	}

	// Second tick retries from block 2 onward and succeeds this time, proving
	// the loop is still alive (no crash, no self-deadlock) and makes progress.
	waitForState(t, w, 3, 3*time.Second)
}
