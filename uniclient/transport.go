package uniclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

// Transport defines the interface for RPC communication.
type Transport interface {
	Call(request *Request, response interface{}) (err error)
}

// httpTransport implements Transport using HTTP POST requests.
type httpTransport struct {
	additionalHeaders map[string]string
	httpUrl           string
	http              *http.Client
}

// Call sends an HTTP POST request with JSON-encoded request body and decodes the response.
func (c *httpTransport) Call(request *Request, response interface{}) (err error) {
	requestBuffer := new(bytes.Buffer)
	err = json.NewEncoder(requestBuffer).Encode(request)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.httpUrl, requestBuffer)
	req.Header.Set("Content-Type", "application/json")
	for key, value := range c.additionalHeaders {
		req.Header.Set(key, value)
	}
	httpResponse, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode != http.StatusOK {
		err = errors.New("invalid server response: " + httpResponse.Status)
	}
	err = json.NewDecoder(httpResponse.Body).Decode(response)
	return err
}

// AddHeader adds a custom HTTP header to be sent with all requests.
func (c *httpTransport) AddHeader(key, value string) {
	if c.additionalHeaders == nil {
		c.additionalHeaders = make(map[string]string)
	}
	c.additionalHeaders[key] = value
}
