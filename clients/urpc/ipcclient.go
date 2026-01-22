package urpc

import (
	"encoding/json"
	"net"
	"sync"
	"time"
)

// ipcClient implements rpcTransport using Unix domain sockets (IPC).
// Provides lower latency compared to HTTP for local node connections.
// Note: May not work on Windows due to differences between Unix sockets and named pipes.
type ipcClient struct {
	mux        sync.Mutex    // Mutex for thread-safe socket access
	socketPath string        // Path to Unix socket file
	socketConn net.Conn      // Active socket connection
	timeOut    time.Duration // Read/write timeout (0 = no timeout)
}

// connect establishes a connection to the Unix socket.
func (i *ipcClient) connect() error {
	conn, err := newSocketConnection(i.socketPath)
	if err != nil {
		return err
	}
	i.socketConn = conn
	return nil
}

// Call sends a JSON-RPC request over the Unix socket.
// Thread-safe: uses mutex to synchronize socket access.
// Lazily connects on first call.
func (i *ipcClient) Call(request interface{}, response interface{}) (err error) {
	i.mux.Lock()
	defer i.mux.Unlock()
	if i.socketConn == nil {
		err = i.connect()
		if err != nil {
			return err
		}
	}
	if i.timeOut > 0 {
		_ = i.socketConn.SetWriteDeadline(time.Now().Add(i.timeOut))
	}
	// Send request to server
	err = json.NewEncoder(i.socketConn).Encode(request)
	if err != nil {
		return err
	}
	if i.timeOut > 0 {
		_ = i.socketConn.SetReadDeadline(time.Now().Add(i.timeOut))
	}
	// Receive response from server
	return json.NewDecoder(i.socketConn).Decode(response)
}

// newSocketConnection establishes a new Unix socket connection.
func newSocketConnection(socketPath string) (net.Conn, error) {
	return new(net.Dialer).Dial("unix", socketPath)
}
