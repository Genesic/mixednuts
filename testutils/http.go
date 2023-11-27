package testutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Genesic/mixednuts/errors"
	httpApp "github.com/Genesic/mixednuts/http"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type TestHttpServer struct {
	Server *httptest.Server
	client *http.Client

	cookie  string
	headers map[string]string
}

func NewTestHttpServer(controller httpApp.Controller) TestHttpServer {
	router := mux.NewRouter()
	controller.RegisterHandlers(router)
	httpServer := httptest.NewServer(router)
	client := httpServer.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return TestHttpServer{
		Server: httpServer,
		client: client,
	}
}

func (c *TestHttpServer) WithHeader(key string, value string) *TestHttpServer {
	c.headers[key] = value
	return c
}

func (c *TestHttpServer) WithCookie(value string) *TestHttpServer {
	c.cookie = value
	return c
}

type Response struct {
	Cookies []*http.Cookie
	Headers http.Header
}

func (c *TestHttpServer) MustDoJSON(t *testing.T, method string, url string, reqBody interface{}, respBody interface{}, statusCode int) *Response {
	// Encode request
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(reqBody)
	if err != nil {
		t.Fatal(err)
	}

	// Prepare request
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", c.Server.URL, url), &buf)
	if err != nil {
		t.Fatal(err)
	}

	return c.mustDo(t, req, respBody, statusCode)
}

func (c *TestHttpServer) MustDoForm(t *testing.T, method string, url string, reqBody url.Values, respBody interface{}, statusCode int) *Response {
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", c.Server.URL, url), strings.NewReader(reqBody.Encode()))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return c.mustDo(t, req, respBody, statusCode)
}

func (c *TestHttpServer) mustDo(t *testing.T, req *http.Request, respBody interface{}, statusCode int) *Response {
	// Set header
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	// Set cookie
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}

	// Do request
	resp, err := c.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Parse response body
	if respBody != nil {
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		if err != nil {
			body, _ := io.ReadAll(resp.Body)
			t.Fatal(fmt.Errorf("status_code: %d, body: %s, err: %s", statusCode, string(body), err))
		}
		So(resp.Header.Get("Content-Type"), ShouldEqual, "application/json")
	}

	So(resp.StatusCode, ShouldEqual, statusCode)

	return &Response{
		Cookies: resp.Cookies(),
		Headers: resp.Header,
	}
}

func (c *TestHttpServer) MustFailedJSON(t *testing.T, method string, path string, reqBody interface{}, err errors.HttpError) {
	respBody := new(interface{})
	c.MustDoJSON(t, method, path, reqBody, respBody, err.GetCode())
	result, _ := json.Marshal(respBody)
	So(string(result), ShouldEqual, err.GetMessage())
}

func (c *TestHttpServer) MustFailedForm(t *testing.T, method string, path string, reqBody url.Values, err errors.HttpError) {
	respBody := new(interface{})
	c.MustDoForm(t, method, path, reqBody, respBody, err.GetCode())
	result, _ := json.Marshal(respBody)
	So(string(result), ShouldEqual, err.GetMessage())
}
