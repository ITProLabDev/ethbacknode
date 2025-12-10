package urpc

import (
	"encoding/json"
	"net"
	"sync"
	"time"
)

type ipcClient struct {
	mux        sync.Mutex
	socketPath string
	socketConn net.Conn
	timeOut    time.Duration
}

func (i *ipcClient) connect() error {
	conn, err := newSocketConnection(i.socketPath)
	if err != nil {
		return err
	}
	i.socketConn = conn
	return nil
}

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
	//Send request to server
	err = json.NewEncoder(i.socketConn).Encode(request)
	if err != nil {
		return err
	}
	if i.timeOut > 0 {
		_ = i.socketConn.SetReadDeadline(time.Now().Add(i.timeOut))
	}
	//Receive response from server
	return json.NewDecoder(i.socketConn).Decode(response)
}

// newSocketConnection will connect to a Unix socket on the given endpoint.
func newSocketConnection(socketPath string) (net.Conn, error) {
	return new(net.Dialer).Dial("unix", socketPath)
}
