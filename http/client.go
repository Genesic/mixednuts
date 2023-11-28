package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	*http.Client

	headers map[string]string
	baseURL string
}

func NewClient(baseURL string) *Client {
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	return &Client{
		Client: &http.Client{
			Timeout: 5 * time.Second,
		},
		headers: make(map[string]string),
		baseURL: baseURL,
	}
}

func (c *Client) WithBaseURL(baseURL string) *Client {
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	c.baseURL = baseURL
	return c
}

func (c *Client) WithTimeout(timeout time.Duration) *Client {
	c.Timeout = timeout
	return c
}

func (c *Client) WithHeaders(key string, value string) *Client {
	c.headers[key] = value
	return c
}

func (c *Client) MakeJSONRequest(method, path string, headers map[string]string, input interface{}) (*http.Request, error) {
	var body io.Reader
	if input != nil {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(input); err != nil {
			return nil, err
		}
		body = &buf
	}

	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c *Client) CommonDoWithJSON(method, path string, headers map[string]string, input, output interface{}) (int, error) {
	req, err := c.MakeJSONRequest(method, path, headers, input)
	if err != nil {
		return -1, err
	}

	return c.CommonDoFromRequest(req, output)
}

func (c *Client) CommonDoWithForm(method, path string, headers map[string]string, input url.Values) (int, []byte, error) {
	req, err := http.NewRequest(method, c.baseURL+path, strings.NewReader(input.Encode()))
	if err != nil {
		return -1, nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.Client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()

	statusCode := res.StatusCode
	bs, err := io.ReadAll(res.Body)
	if err != nil {
		return statusCode, nil, errors.New("cannot read service error in response body")
	}
	bs = bytes.TrimSpace(bs)

	if statusCode >= 300 {
		return statusCode, nil, errors.New(string(bs))
	}

	return statusCode, bs, nil
}

func (c *Client) CommonDoFromRequest(req *http.Request, output interface{}) (int, error) {
	res, err := c.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	statusCode := res.StatusCode
	if statusCode >= 300 {
		var bs []byte
		bs, err = io.ReadAll(res.Body)
		if err != nil {
			return statusCode, fmt.Errorf("cannot read service error in response body: %w", err)
		}
		bs = bytes.TrimSpace(bs)

		return statusCode, errors.New(string(bs))
	}

	if output != nil {
		if err = json.NewDecoder(res.Body).Decode(output); err != nil {
			return statusCode, fmt.Errorf("failed to parse service response: %w", err)
		}
	}

	return statusCode, nil
}
