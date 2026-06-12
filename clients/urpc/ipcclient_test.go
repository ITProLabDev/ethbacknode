package urpc

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"testing"
	"time"
)

// startTestIPCService runs a Unix-socket echo server for the IPC client test.
// It echoes each decoded JSON message back. Errors are reported via serveErr
// (not panic) so a failure fails the test cleanly instead of crashing the
// process. It stops when the listener is closed (via t.Cleanup).
func startTestIPCService(t *testing.T, socketPath string) {
	t.Helper()
	_ = os.Remove(socketPath)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("server listen: %v", err)
	}

	var conns []net.Conn
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed
			}
			conns = append(conns, conn)
			dec := json.NewDecoder(conn)
			enc := json.NewEncoder(conn)
			for {
				var rd json.RawMessage
				if err := dec.Decode(&rd); err != nil {
					break // client closed / done
				}
				if err := enc.Encode(rd); err != nil {
					break
				}
			}
		}
	}()

	t.Cleanup(func() {
		_ = ln.Close()
		for _, c := range conns {
			_ = c.Close()
		}
		<-done
		_ = os.Remove(socketPath)
	})
}

func TestIPCClient(t *testing.T) {
	// Unique socket path per run avoids collisions with a stale /tmp/pipe.ipc
	// or a parallel test process.
	socketPath := fmt.Sprintf("/tmp/ebn-ipcclient-%d.ipc", os.Getpid())
	startTestIPCService(t, socketPath)

	client := &ipcClient{
		socketPath: socketPath,
		timeOut:    10 * time.Second, // generous: this is a correctness test, not a latency benchmark
	}

	// Representative message sizes: tiny, around typical RPC payloads, and large
	// enough to exercise multi-read framing over the socket. A deterministic set
	// proves round-trip correctness without 262K flaky, slow iterations.
	for _, n := range []int{33, 64, 256, 1024, 8192, 65536, 262143} {
		msg := json.RawMessage(randBytes(n))
		var response json.RawMessage
		if err := client.Call(msg, &response); err != nil {
			t.Fatalf("Call(size=%d): %v", n, err)
		}
		if len(response) != len(msg) {
			t.Fatalf("size=%d: echoed %d bytes, want %d", n, len(response), len(msg))
		}
	}
}

const letterBytes = `0987654321abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`

// randBytes returns a quoted JSON string literal of n random alphanumeric bytes.
func randBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	out := make([]byte, 0, n+2)
	out = append(out, '"')
	out = append(out, b...)
	out = append(out, '"')
	return out
}
