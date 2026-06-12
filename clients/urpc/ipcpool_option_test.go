package urpc

import (
	"encoding/json"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// jsonrpcEchoServer answers each JSON-RPC request with a result echoing the
// request id, while tracking peak concurrent connections.
type jsonrpcEchoServer struct {
	ln       net.Listener
	delay    time.Duration
	curConns int32
	maxConns int32
	wg       sync.WaitGroup

	connMu sync.Mutex
	conns  []net.Conn
}

func newJSONRPCEchoServer(t *testing.T, path string, delay time.Duration) *jsonrpcEchoServer {
	t.Helper()
	_ = os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := &jsonrpcEchoServer{ln: ln, delay: delay}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			s.connMu.Lock()
			s.conns = append(s.conns, conn)
			s.connMu.Unlock()
			s.wg.Add(1)
			go s.serve(conn)
		}
	}()
	// Cleanup must close the listener AND every accepted connection: the pool
	// keeps connections open for reuse, so serve()'s blocking Decode only
	// returns once its conn is closed. Without this, wg.Wait() deadlocks.
	t.Cleanup(func() {
		_ = ln.Close()
		s.connMu.Lock()
		for _, c := range s.conns {
			_ = c.Close()
		}
		s.connMu.Unlock()
		s.wg.Wait()
	})
	return s
}

func (s *jsonrpcEchoServer) serve(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()
	n := atomic.AddInt32(&s.curConns, 1)
	for {
		m := atomic.LoadInt32(&s.maxConns)
		if n <= m || atomic.CompareAndSwapInt32(&s.maxConns, m, n) {
			break
		}
	}
	defer atomic.AddInt32(&s.curConns, -1)

	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)
	for {
		var req struct {
			Id json.RawMessage `json:"id"`
		}
		if err := dec.Decode(&req); err != nil {
			return
		}
		if s.delay > 0 {
			time.Sleep(s.delay)
		}
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      json.RawMessage(req.Id),
			"result":  json.RawMessage(`"ok"`),
		}
		if err := enc.Encode(resp); err != nil {
			return
		}
	}
}

func (s *jsonrpcEchoServer) MaxConns() int { return int(atomic.LoadInt32(&s.maxConns)) }

func TestWithRpcIPCSocketPool_ParallelCalls(t *testing.T) {
	const size = 4
	path := tempSock(t)
	srv := newJSONRPCEchoServer(t, path, 80*time.Millisecond)

	client := NewClient(WithRpcIPCSocketPool(path, size, 5*time.Second))

	var wg sync.WaitGroup
	var failures int32
	for i := 0; i < size; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Call(NewRequest("eth_blockNumber"))
			if err != nil || resp == nil {
				atomic.AddInt32(&failures, 1)
			}
		}()
	}
	wg.Wait()

	if failures != 0 {
		t.Fatalf("%d calls failed", failures)
	}
	if srv.MaxConns() != size {
		t.Fatalf("pool did not parallelize: max concurrent conns = %d, want %d", srv.MaxConns(), size)
	}
}
