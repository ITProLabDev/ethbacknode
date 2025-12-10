package urpc

import (
	"errors"
	"net/http"
)

const (
	rpcMode = iota
	restMode
)

type ClientOption func(client *Client)

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

type Client struct {
	rpcClient  rpcTransport
	restClient restTransport
}

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

func (c *Client) Get(method string, params map[string]interface{}, response interface{}) (err error) {
	return c.restClient.Get(method, params, response)
}

func (c *Client) Post(method string, params interface{}, response interface{}) (err error) {
	return c.restClient.Post(method, params, response)
}

type rpcTransport interface {
	Call(request interface{}, response interface{}) (err error)
}

type restTransport interface {
	Get(uri string, params map[string]interface{}, response interface{}) (err error)
	Post(uri string, request interface{}, response interface{}) (err error)
}
