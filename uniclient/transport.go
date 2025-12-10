package uniclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

type Transport interface {
	Call(request *Request, response interface{}) (err error)
}

type httpTransport struct {
	additionalHeaders map[string]string
	httpUrl           string
	http              *http.Client
}

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

func (c *httpTransport) AddHeader(key, value string) {
	if c.additionalHeaders == nil {
		c.additionalHeaders = make(map[string]string)
	}
	c.additionalHeaders[key] = value
}
