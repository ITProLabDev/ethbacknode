package urpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

type httpClient struct {
	additionalHeaders map[string]string
	httpUrl           string
	http              *http.Client
}

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

func (c *httpClient) AddHeader(key, value string) {
	if c.additionalHeaders == nil {
		c.additionalHeaders = make(map[string]string)
	}
	c.additionalHeaders[key] = value
}
