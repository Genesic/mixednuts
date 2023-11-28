package middleware

import (
	"bytes"
	"fmt"
	"github.com/Genesic/mixednuts/logging"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"
)

const (
	requestHeaderPrefix  = "req-"
	responseHeaderPrefix = "resp-"

	requestLogMsg = "req-log"
)

var includeHeader = map[string]struct{}{
	"content-type": {},
}

type logMiddlewareHandler struct {
	next http.Handler
}

type httpRequest struct {
	Method    string `json:"requestMethod"`
	URL       string `json:"requestUrl"`
	ReqSize   int64  `json:"requestSize"`
	Status    int    `json:"status"`
	UserAgent string `json:"userAgent"`
	RespSize  int    `json:"responseSize"`
	Latency   string `json:"latency"`
	IP        string `json:"IP"`
	ClientID  string `json:"clientID,omitempty"`
}

func (h *logMiddlewareHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logging.FromContext(ctx)
	r = r.WithContext(logging.WithLogger(ctx, logger))

	// Here we assert the ResponseWriter should have type *apicommon.ResponseWriter,
	// otherwise we cannot get status code here
	rw, ok := w.(*ResponseWriter)
	if !ok {
		logger.Fatalw("ResponseMiddleware should be placed before LogMiddleware")
	}

	var requestField httpRequest

	fields := populateRequestHeaderFields(r, &requestField)

	// This must be put before ServeHTTP to intercept request body.
	reqBuf := &bytes.Buffer{}
	r.Body = io.NopCloser(io.TeeReader(r.Body, reqBuf))

	h.next.ServeHTTP(w, r)

	// fields = append(fields, zap.String(requestBodyKey, reqBuf.String()))
	// fields = append(fields, zap.String(responseBodyKey, string(rw.GetBody())))
	fields = append(fields, populateResponseHeaderFields(rw, &requestField)...)
	fields = append(fields, zap.Any("httpRequest", requestField))

	if rw.GetError() != nil {
		logger.Desugar().Error(requestLogMsg, fields...)
	} else {
		logger.Desugar().Info(requestLogMsg, fields...)
	}
}

func populateRequestHeaderFields(r *http.Request, requestField *httpRequest) []zap.Field {
	requestField.Method = r.Method
	requestField.URL = r.URL.String()
	requestField.ReqSize = r.ContentLength
	requestField.IP = r.Header.Get("x-real-ip")

	return generateHeaderLogFields(requestHeaderPrefix, r.Header)

}

func populateResponseHeaderFields(rw *ResponseWriter, requestField *httpRequest) []zap.Field {
	requestField.Latency = strings.TrimRight(strings.TrimRight(
		fmt.Sprintf("%.4f", rw.GetRequestDuration().Seconds()),
		"0"), ".") + "s"
	requestField.RespSize = len(rw.GetBody())
	requestField.Status = rw.GetStatusCode()
	requestField.ClientID = rw.GetClientID()

	fields := generateHeaderLogFields(responseHeaderPrefix, rw.Header())
	return fields
}

func generateHeaderLogFields(prefix string, headers map[string][]string) []zap.Field {
	var fields []zap.Field
	for hdr, values := range headers {
		hdr = strings.ToLower(hdr)
		if _, ok := includeHeader[hdr]; ok {
			value := strings.Join(values, ",")
			fields = append(fields, zap.String(prefix+hdr, value))
		}

	}

	return fields
}

func LogMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return &logMiddlewareHandler{
			next: next,
		}
	}
}
