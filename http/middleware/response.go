package middleware

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"time"
)

type ResponseWriter struct {
	writer          http.ResponseWriter
	statusCode      int
	requestDuration time.Duration
	clientID        string
	body            []byte
	err             error
}

func (r *ResponseWriter) Header() http.Header {
	return r.writer.Header()
}

func (r *ResponseWriter) Write(body []byte) (int, error) {
	r.body = body
	return r.writer.Write(body)
}

func (r *ResponseWriter) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.writer.WriteHeader(statusCode)
}

func (r *ResponseWriter) WriteRequestDuration(duration time.Duration) {
	r.requestDuration = duration
}

func (r *ResponseWriter) WriteClientID(clientID string) {
	r.clientID = clientID
}

func (r *ResponseWriter) WriteError(err error) {
	r.err = err
}

func (r *ResponseWriter) GetStatusCode() int {
	return r.statusCode
}

func (r *ResponseWriter) GetRequestDuration() time.Duration {
	return r.requestDuration
}

func (r *ResponseWriter) GetClientID() string {
	return r.clientID
}

func (r *ResponseWriter) GetBody() []byte {
	return r.body
}

func (r *ResponseWriter) GetError() error {
	return r.err
}

func (r *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := r.writer.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijack not supported")
	}
	return h.Hijack()
}

func ResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&ResponseWriter{
			writer:     w,
			statusCode: http.StatusOK,
		}, r)
	})
}
