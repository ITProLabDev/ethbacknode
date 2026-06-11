package ethclient

import (
	"fmt"
	"time"

	"github.com/ITProLabDev/ethbacknode/abi"
	"github.com/ITProLabDev/ethbacknode/clients/urpc"
	"github.com/ITProLabDev/ethbacknode/storage"
)

func WithRpcClient(nodeAddress, nodePort string, useSSL bool, headers map[string]string) Option {
	urlMaks := "http://%s:%s"
	if useSSL {
		urlMaks = "https://%s:%s"
	}
	endpointUrl := fmt.Sprintf(urlMaks, nodeAddress, nodePort)
	return func(client *Client) {
		rpcClient := urpc.NewClient(urpc.WithHTTPRpc(endpointUrl, headers))
		client.rpcClient = rpcClient
	}
}

func WithIPCClient(ipcPath string) Option {
	return func(client *Client) {
		rpcClient := urpc.NewClient(urpc.WithRpcIPCSocket(ipcPath))
		client.rpcClient = rpcClient
	}
}

// WithIPCClientPool configures the client to talk to the node over a pool of
// up to poolSize IPC connections, so independent RPC calls can run in parallel
// instead of being serialized on a single socket. A poolSize <= 1 is
// equivalent to WithIPCClient. timeout bounds each call (0 = no deadline).
func WithIPCClientPool(ipcPath string, poolSize int, timeout time.Duration) Option {
	return func(client *Client) {
		rpcClient := urpc.NewClient(urpc.WithRpcIPCSocketPool(ipcPath, poolSize, timeout))
		client.rpcClient = rpcClient
	}
}

func WithAbiManager(abiManager *abi.SmartContractsManager) Option {
	return func(client *Client) {
		client.abi = abiManager
	}
}

func WithConfigStorage(storage storage.BinStorage) Option {
	return func(client *Client) {
		client.config.storage = storage
	}
}
