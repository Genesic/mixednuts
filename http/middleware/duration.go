package middleware

import (
	"net/http"
	"time"

	"github.com/Genesic/mixednuts/logging"
)

func RequestDurationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		begin := time.Now()
		defer func() {
			logger := logging.FromContext(r.Context())

			// Here we assert the ResponseWriter should have type *apicommon.ResponseWriter,
			// otherwise we cannot get status code here
			rw, ok := w.(*ResponseWriter)
			if !ok {
				logger.Fatalw("ResponseMiddleware should be placed before RequestDurationMiddleware")
			}

			// Update request duration metrics aggregated by routing pattern and response status code
			duration := time.Since(begin)
			rw.WriteRequestDuration(duration)
		}()

		next.ServeHTTP(w, r)
	})
}
