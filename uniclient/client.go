package uniclient

import (
	"backnode/tools/log"
	"net/http"
)

type ClientOption func(client *Client)

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

func WithServiceId(serviceId int) ClientOption {
	return func(client *Client) {
		client.serviceId = serviceId
	}
}

func NewClient(options ...ClientOption) *Client {
	client := &Client{}
	for _, option := range options {
		option(client)
	}
	return client
}

type Client struct {
	debug     bool
	serviceId int
	rpcClient Transport
}

func (c *Client) SetDebug(debug bool) {
	c.debug = debug
}

func (c *Client) Call(request *Request, response interface{}) (err error) {
	return c.rpcClient.Call(request, response)
}

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
