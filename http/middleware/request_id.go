package middleware

import (
	"context"
	"net/http"

	"github.com/Genesic/mixednuts/logging"
	"github.com/Genesic/mixednuts/utils"
)

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		requestID := r.Header.Get(logging.RequestIdHeader)
		if requestID == "" {
			requestID = utils.GenRequestID()
		}

		ctx = context.WithValue(ctx, logging.RequestIDKey, requestID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
