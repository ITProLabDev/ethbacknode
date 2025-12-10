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

var (
	startComplete = make(chan bool)
)

func startTestIPCService() {
	_ = os.Remove("/tmp/pipe.ipc")
	serverSocketListener, err := net.Listen("unix", "/tmp/pipe.ipc")
	if err != nil {
		fmt.Printf("Server failed to start - %s\n", err)
		panic(err)
	}
	// Remove IPC file on exit
	defer serverSocketListener.Close()
	defer os.Remove("/tmp/pipe.ipc")
	startComplete <- true
	for {
		conn, err := serverSocketListener.Accept()
		if err != nil {
			fmt.Printf("Server failed to accept connection - %s\n", err)
			return
		}
		// read the data
		fmt.Println("Server accepted connection, read data...")
		for {
			rd := json.RawMessage{}
			err = json.NewDecoder(conn).Decode(&rd)
			if err != nil {
				fmt.Printf("Server failed to read data - %s\n", err)
				panic(err)
			}
			fmt.Println("Server read data:", len(rd), string(rd), "bytes, send it back...")
			conn.SetDeadline(time.Now().Add(1 * time.Second))
			err = json.NewEncoder(conn).Encode(rd)
			if err != nil {
				fmt.Printf("Server failed to write data - %s\n", err)
				panic(err)
			}
			fmt.Println("Server sent data back", "bytes")
		}
	}
}

/*
// account for null-terminator too
const (

	// On Linux, sun_path is 108 bytes in size
	// see http://man7.org/linux/man-pages/man7/unix.7.html
	maxPathSize = int(108)

)

	if len(endpoint)+1 > maxPathSize {
		log.Warn(fmt.Sprintf("The ipc endpoint is longer than %d characters. ", maxPathSize-1),
			"endpoint", endpoint)
	}
*/

func TestIPCClient(t *testing.T) {
	go startTestIPCService()
	<-startComplete
	client := &ipcClient{
		socketPath: "/tmp/pipe.ipc",
	}
	for i := 33; i < 1024*256; i++ {
		fmt.Println("Call socket...")
		msg := json.RawMessage(randBytes(i))
		var response json.RawMessage
		fmt.Println("Msg len:", len(msg), "Last byte:", msg[len(msg)-1])
		client.timeOut = 1 * time.Second
		err := client.Call(msg, &response)
		if err != nil {
			t.Log("Response len:", len(response))
			t.Error(err)
		}
	}
	fmt.Println("Test complete, exit")
}

const letterBytes = `0987654321abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`

func randBytes(n int) []byte {
	b := make([]byte, n+1)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	out := append([]byte{'"'}, b...)
	out[len(out)-1] = '"'
	return out
}
