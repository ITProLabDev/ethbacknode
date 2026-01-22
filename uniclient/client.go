// Package uniclient provides a unified JSON-RPC client for interacting with EthBackNode services.
// It supports address management, balance queries, transactions, and service operations.
package uniclient

import (
	"net/http"

	"github.com/ITProLabDev/ethbacknode/tools/log"
)

// ClientOption is a function that configures a Client.
type ClientOption func(client *Client)

// WithHttpTransport configures the client to use HTTP transport with the given endpoint.
func WithHttpTransport(endpointUrl string, headers map[string]string) ClientOption {
	return func(client *Client) {
		transport := &httpTransport{
			httpUrl: endpointUrl,
			http:    &http.Client{},
		}
		if headers != nil {
			for key, value := range headers {
				transport.AddHeader(key, value)
			}
		}
		client.rpcClient = transport
	}
}

// WithServiceId sets the service ID for subscription-based operations.
func WithServiceId(serviceId int) ClientOption {
	return func(client *Client) {
		client.serviceId = serviceId
	}
}

// NewClient creates a new unified RPC client with the given options.
func NewClient(options ...ClientOption) *Client {
	client := &Client{}
	for _, option := range options {
		option(client)
	}
	return client
}

// Client is a JSON-RPC client for EthBackNode API.
type Client struct {
	debug     bool
	serviceId int
	rpcClient Transport
}

// SetDebug enables or disables debug logging of requests and responses.
func (c *Client) SetDebug(debug bool) {
	c.debug = debug
}

// Call executes a raw RPC request and populates the response.
func (c *Client) Call(request *Request, response interface{}) (err error) {
	return c.rpcClient.Call(request, response)
}

// rpcCall executes an RPC request and parses the result into the given struct.
func (c *Client) rpcCall(request *Request, result interface{}) (err error) {
	rpcResponse := NewResponse()
	if c.debug {
		log.Dump(request)
	}
	err = c.rpcClient.Call(request, rpcResponse)
	if err != nil {
		return err
	}
	if c.debug {
		log.Dump(rpcResponse)
	}
	if rpcResponse.HasError() {
		return rpcResponse.Error
	}
	return rpcResponse.ParseResult(result)
}
