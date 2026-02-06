// Package urpc provides a universal RPC client supporting JSON-RPC 2.0 over HTTP and IPC.
// It abstracts the transport layer, allowing the same client API for different protocols.
package urpc

import (
	"errors"
	"net/http"
)

// Transport mode constants.
const (
	rpcMode = iota  // JSON-RPC mode
	restMode        // REST API mode
)

// ClientOption is a function that configures a Client.
type ClientOption func(client *Client)

// NewClient creates a new universal RPC client with the specified options.
// Options determine the transport type (HTTP-RPC, IPC socket, or REST).
func NewClient(options ...ClientOption) *Client {
	client := &Client{}
	for _, option := range options {
		option(client)
	}
	return client
}

// WithHTTPRpc sets the URL of the RPC server for http network requests
func WithHTTPRpc(url string, headers map[string]string) ClientOption {
	return func(client *Client) {
		httpClientInstance := &httpClient{
			httpUrl:           url,
			http:              &http.Client{},
			additionalHeaders: make(map[string]string),
		}
		if headers != nil {
			for key, value := range headers {
				httpClientInstance.AddHeader(key, value)
			}
		}
		client.rpcClient = httpClientInstance
	}
}

// WithRpcIPCSocket sets the RPC client for use unix socket interactions
// Please note, may not work on Windows (differences between Unix
// sockets and Windows named pipes)
func WithRpcIPCSocket(socketPath string) ClientOption {
	return func(client *Client) {
		client.rpcClient = &ipcClient{
			socketPath: socketPath,
		}
	}
}

// WithHTTPRest sets the URL of the REST server for http network requests
// This is used for RESTful like API requests for TRON and similar networks
func WithHTTPRest(url string, headers map[string]string) ClientOption {
	return func(client *Client) {
		httpClientInstance := &httpClient{
			httpUrl:           url,
			http:              &http.Client{},
			additionalHeaders: make(map[string]string),
		}
		if headers != nil {
			for key, value := range headers {
				httpClientInstance.AddHeader(key, value)
			}
		}
		client.rpcClient = httpClientInstance
		client.restClient = httpClientInstance
	}
}

// Client is the universal RPC client that supports multiple transports.
// It can use HTTP-RPC, IPC sockets, or REST depending on configuration.
type Client struct {
	rpcClient  rpcTransport  // Transport for JSON-RPC calls
	restClient restTransport // Transport for REST API calls
}

// Call executes a JSON-RPC request and returns the response.
// Returns an error if the request fails or if the response contains an error.
func (c *Client) Call(request *Request) (response *Response, err error) {
	response = NewResponse()
	err = c.rpcClient.Call(request, response)
	if err != nil {
		return nil, err
	}
	if response.Error != nil {
		return response, errors.New(response.Error.Message)
	}
	return response, nil
}

// Get performs a REST GET request to the specified method/endpoint.
func (c *Client) Get(method string, params map[string]interface{}, response interface{}) (err error) {
	return c.restClient.Get(method, params, response)
}

// Post performs a REST POST request to the specified method/endpoint.
func (c *Client) Post(method string, params interface{}, response interface{}) (err error) {
	return c.restClient.Post(method, params, response)
}

// rpcTransport defines the interface for JSON-RPC transports.
type rpcTransport interface {
	// Call sends a request and populates the response.
	Call(request interface{}, response interface{}) (err error)
}

// restTransport defines the interface for REST API transports.
type restTransport interface {
	// Get performs an HTTP GET request.
	Get(uri string, params map[string]interface{}, response interface{}) (err error)
	// Post performs an HTTP POST request.
	Post(uri string, request interface{}, response interface{}) (err error)
}
