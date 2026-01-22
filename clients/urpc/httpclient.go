package urpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// httpClient implements rpcTransport and restTransport over HTTP.
// Supports custom headers for authentication and identification.
type httpClient struct {
	additionalHeaders map[string]string // Custom HTTP headers
	httpUrl           string            // Base URL for requests
	http              *http.Client      // Underlying HTTP client
}

// Call sends a JSON-RPC request over HTTP POST.
// Encodes the request as JSON and decodes the response.
func (c *httpClient) Call(request interface{}, response interface{}) (err error) {
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

// Get performs an HTTP GET request to the specified URI with query parameters.
func (c *httpClient) Get(uri string, params map[string]interface{}, response interface{}) (err error) {
	urlParsed, err := url.Parse(c.httpUrl)
	if err != nil {
		return err
	}
	urlParsed.Path = uri
	for key, value := range params {
		urlParsed.Query().Add(key, fmt.Sprintf("%v", value))
	}
	httpRequest, err := http.NewRequest("GET", urlParsed.String(), nil)
	httpRequest.Header.Set("Content-Type", "application/json")
	for key, value := range c.additionalHeaders {
		httpRequest.Header.Set(key, value)
	}
	httpResponse, err := c.http.Do(httpRequest)
	if err != nil {
		return err
	}
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode != http.StatusOK {
		return err
	}
	return json.NewDecoder(httpResponse.Body).Decode(response)
}

// Post performs an HTTP POST request to the specified URI with JSON body.
func (c *httpClient) Post(uri string, request interface{}, response interface{}) (err error) {
	urlParsed, err := url.Parse(c.httpUrl)
	if err != nil {
		return err
	}
	urlParsed.Path = uri
	requestBuffer := new(bytes.Buffer)
	err = json.NewEncoder(requestBuffer).Encode(request)
	if err != nil {
		return err
	}
	httpRequest, err := http.NewRequest("POST", urlParsed.String(), requestBuffer)
	if err != nil {
		return err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	for key, value := range c.additionalHeaders {
		httpRequest.Header.Set(key, value)
	}
	httpResponse, err := c.http.Do(httpRequest)
	if err != nil {
		return err
	}
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode != http.StatusOK {
		err = errors.New("invalid server response: " + httpResponse.Status)
		return err
	}
	return json.NewDecoder(httpResponse.Body).Decode(response)
}

// AddHeader adds or updates a custom HTTP header.
// Headers are included in all subsequent requests.
func (c *httpClient) AddHeader(key, value string) {
	if c.additionalHeaders == nil {
		c.additionalHeaders = make(map[string]string)
	}
	c.additionalHeaders[key] = value
}
