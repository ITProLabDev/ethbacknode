package urpc

import (
	"encoding/json"
	"net"
	"sync"
	"time"
)

// ipcPool is a concurrency-safe pool of IPC (Unix-domain) connections to a
// node socket. Unlike ipcClient — which serializes every request through a
// single mutex-guarded connection — the pool keeps up to a fixed number of
// live connections, allowing independent RPC calls to proceed in parallel
// while still bounding the load placed on the node.
//
// Design: a counting semaphore caps in-flight calls at the pool size; a LIFO
// stack of idle connections hands the most-recently-used (already-dialed)
// connection back first, so sequential calls reuse one connection instead of
// fanning out to new sockets. Each connection is owned exclusively between
// acquire and release, which prevents request/response interleaving on a
// single socket.
type ipcPool struct {
	socketPath string
	timeout    time.Duration

	sem chan struct{} // tokens; len == pool size, bounds concurrency

	mu     sync.Mutex
	idle   []*pooledConn
	closed bool
}

// pooledConn pairs a connection with its long-lived JSON codec.
type pooledConn struct {
	conn net.Conn
	enc  *json.Encoder
	dec  *json.Decoder
}

// newIPCPool creates a pool with at most size live connections. A size <= 0 is
// coerced to a single connection (equivalent to the serialized ipcClient).
func newIPCPool(socketPath string, size int, timeout time.Duration) *ipcPool {
	if size <= 0 {
		size = 1
	}
	return &ipcPool{
		socketPath: socketPath,
		timeout:    timeout,
		sem:        make(chan struct{}, size),
	}
}

// Call sends a JSON-RPC request over one pooled connection and decodes the
// response into the provided value. It blocks until a connection slot is free,
// bounding concurrency to the pool size.
func (p *ipcPool) Call(request interface{}, response interface{}) error {
	// Acquire a slot — this is what bounds concurrency to the pool size.
	p.sem <- struct{}{}
	defer func() { <-p.sem }()

	pc, err := p.acquire()
	if err != nil {
		return err
	}

	if p.timeout > 0 {
		_ = pc.conn.SetWriteDeadline(time.Now().Add(p.timeout))
	}
	if err := pc.enc.Encode(request); err != nil {
		p.discard(pc)
		return err
	}
	if p.timeout > 0 {
		_ = pc.conn.SetReadDeadline(time.Now().Add(p.timeout))
	}
	if err := pc.dec.Decode(response); err != nil {
		p.discard(pc)
		return err
	}

	p.release(pc)
	return nil
}

// acquire returns a live connection, reusing an idle one (LIFO) when available
// or dialing a fresh connection otherwise.
func (p *ipcPool) acquire() (*pooledConn, error) {
	p.mu.Lock()
	if n := len(p.idle); n > 0 {
		pc := p.idle[n-1]
		p.idle = p.idle[:n-1]
		p.mu.Unlock()
		return pc, nil
	}
	p.mu.Unlock()

	conn, err := new(net.Dialer).Dial("unix", p.socketPath)
	if err != nil {
		return nil, err
	}
	return &pooledConn{
		conn: conn,
		enc:  json.NewEncoder(conn),
		dec:  json.NewDecoder(conn),
	}, nil
}

// release returns a healthy connection to the idle stack for reuse. If the
// pool has been closed, the connection is closed instead.
func (p *ipcPool) release(pc *pooledConn) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		_ = pc.conn.Close()
		return
	}
	p.idle = append(p.idle, pc)
	p.mu.Unlock()
}

// discard closes a connection that errored so the next acquire dials a fresh one.
func (p *ipcPool) discard(pc *pooledConn) {
	_ = pc.conn.Close()
}

// Close shuts the pool and closes all idle connections. Connections currently
// in use are closed when their Call returns.
func (p *ipcPool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	idle := p.idle
	p.idle = nil
	p.mu.Unlock()
	for _, pc := range idle {
		_ = pc.conn.Close()
	}
}
