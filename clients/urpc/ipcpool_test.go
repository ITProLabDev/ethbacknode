package urpc

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// --- concurrent echo server test double -----------------------------------

type echoServer struct {
	ln            net.Listener
	delay         time.Duration
	curConns      int32
	maxConns      int32
	totalAccepted int32
	connWg        sync.WaitGroup
}

func newEchoServer(t *testing.T, path string, delay time.Duration) *echoServer {
	t.Helper()
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatalf("listen %s: %v", path, err)
	}
	s := &echoServer{ln: ln, delay: delay}
	go s.acceptLoop()
	t.Cleanup(func() { _ = s.ln.Close(); s.connWg.Wait() })
	return s
}

func (s *echoServer) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		s.connWg.Add(1)
		go s.handle(conn)
	}
}

func (s *echoServer) handle(conn net.Conn) {
	defer s.connWg.Done()
	defer conn.Close()
	atomic.AddInt32(&s.totalAccepted, 1)
	n := atomic.AddInt32(&s.curConns, 1)
	s.observeMax(n)
	defer atomic.AddInt32(&s.curConns, -1)

	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)
	for {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return
		}
		if s.delay > 0 {
			time.Sleep(s.delay)
		}
		if err := enc.Encode(raw); err != nil {
			return
		}
	}
}

func (s *echoServer) observeMax(n int32) {
	for {
		m := atomic.LoadInt32(&s.maxConns)
		if n <= m || atomic.CompareAndSwapInt32(&s.maxConns, m, n) {
			return
		}
	}
}

func (s *echoServer) MaxConns() int      { return int(atomic.LoadInt32(&s.maxConns)) }
func (s *echoServer) TotalAccepted() int { return int(atomic.LoadInt32(&s.totalAccepted)) }

// --- helpers --------------------------------------------------------------

var sockCounter int32

func tempSock(t *testing.T) string {
	t.Helper()
	p := fmt.Sprintf("/tmp/ebn-ipcpool-%d-%d.ipc", os.Getpid(), atomic.AddInt32(&sockCounter, 1))
	_ = os.Remove(p)
	t.Cleanup(func() { _ = os.Remove(p) })
	return p
}

// --- tests ----------------------------------------------------------------

func TestIPCPool_RoundTrip(t *testing.T) {
	path := tempSock(t)
	newEchoServer(t, path, 0)
	pool := newIPCPool(path, 4, 2*time.Second)
	t.Cleanup(func() { pool.Close() })

	var got int
	if err := pool.Call(42, &got); err != nil {
		t.Fatalf("Call: %v", err)
	}
	if got != 42 {
		t.Fatalf("got %d, want 42", got)
	}
}

func TestIPCPool_ConcurrentCallsRunInParallel(t *testing.T) {
	const size = 5
	path := tempSock(t)
	srv := newEchoServer(t, path, 100*time.Millisecond) // hold each request open
	pool := newIPCPool(path, size, 5*time.Second)
	t.Cleanup(func() { pool.Close() })

	var wg sync.WaitGroup
	errs := make([]error, size)
	for i := 0; i < size; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			var got int
			if err := pool.Call(idx, &got); err != nil {
				errs[idx] = err
				return
			}
			if got != idx {
				errs[idx] = fmt.Errorf("got %d want %d", got, idx)
			}
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	// If the pool serialized calls onto one socket, the server would only ever
	// see a single concurrent connection.
	if srv.MaxConns() != size {
		t.Fatalf("max concurrent connections = %d, want %d (no real parallelism)", srv.MaxConns(), size)
	}
}

func TestIPCPool_NeverExceedsPoolSize(t *testing.T) {
	const size = 3
	const calls = 12
	path := tempSock(t)
	srv := newEchoServer(t, path, 30*time.Millisecond)
	pool := newIPCPool(path, size, 5*time.Second)
	t.Cleanup(func() { pool.Close() })

	var wg sync.WaitGroup
	var failures int32
	for i := 0; i < calls; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			var got int
			if err := pool.Call(idx, &got); err != nil || got != idx {
				atomic.AddInt32(&failures, 1)
			}
		}(i)
	}
	wg.Wait()
	if failures != 0 {
		t.Fatalf("%d calls failed", failures)
	}
	if srv.MaxConns() > size {
		t.Fatalf("opened %d simultaneous connections, must not exceed pool size %d", srv.MaxConns(), size)
	}
}

func TestIPCPool_ReusesConnections(t *testing.T) {
	path := tempSock(t)
	srv := newEchoServer(t, path, 0)
	pool := newIPCPool(path, 4, 2*time.Second)
	t.Cleanup(func() { pool.Close() })

	for i := 0; i < 10; i++ {
		var got int
		if err := pool.Call(i, &got); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if got != i {
			t.Fatalf("call %d: got %d", i, got)
		}
	}
	// Sequential calls must reuse a single connection, not open ten.
	if srv.TotalAccepted() != 1 {
		t.Fatalf("opened %d connections for sequential calls, want 1", srv.TotalAccepted())
	}
	if srv.MaxConns() != 1 {
		t.Fatalf("max concurrent = %d, want 1", srv.MaxConns())
	}
}

func TestIPCPool_NoCrossTalk(t *testing.T) {
	const calls = 50
	path := tempSock(t)
	newEchoServer(t, path, 10*time.Millisecond)
	pool := newIPCPool(path, 8, 5*time.Second)
	t.Cleanup(func() { pool.Close() })

	var wg sync.WaitGroup
	mismatches := make([]string, calls)
	for i := 0; i < calls; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			var got int
			if err := pool.Call(idx, &got); err != nil {
				mismatches[idx] = fmt.Sprintf("call %d error: %v", idx, err)
				return
			}
			if got != idx {
				mismatches[idx] = fmt.Sprintf("call %d got %d (cross-talk)", idx, got)
			}
		}(i)
	}
	wg.Wait()
	for _, m := range mismatches {
		if m != "" {
			t.Fatal(m)
		}
	}
}

func TestIPCPool_DialErrorThenRecovers(t *testing.T) {
	path := tempSock(t)
	pool := newIPCPool(path, 2, 2*time.Second)
	t.Cleanup(func() { pool.Close() })

	// No server yet: the dial must fail and not leak a connection slot.
	var got int
	if err := pool.Call(1, &got); err == nil {
		t.Fatal("expected dial error when no server is listening")
	}

	// Bring the server up; the pool must recover and serve normally.
	newEchoServer(t, path, 0)
	for i := 0; i < 5; i++ {
		if err := pool.Call(7, &got); err != nil {
			t.Fatalf("recovery call %d: %v", i, err)
		}
		if got != 7 {
			t.Fatalf("recovery call %d: got %d, want 7", i, got)
		}
	}
}

func TestIPCPool_NonPositiveSizeStillWorks(t *testing.T) {
	path := tempSock(t)
	srv := newEchoServer(t, path, 0)
	pool := newIPCPool(path, 0, 2*time.Second) // coerced to a single connection
	t.Cleanup(func() { pool.Close() })

	for i := 0; i < 3; i++ {
		var got int
		if err := pool.Call(i, &got); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if got != i {
			t.Fatalf("call %d: got %d", i, got)
		}
	}
	if srv.MaxConns() != 1 {
		t.Fatalf("max concurrent = %d, want 1", srv.MaxConns())
	}
}

func TestIPCPool_ImplementsTransport(t *testing.T) {
	var _ rpcTransport = newIPCPool("/tmp/whatever.ipc", 2, time.Second)
}
