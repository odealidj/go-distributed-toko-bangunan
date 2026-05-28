package httpx

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

const (
	HeaderRequestID      = "X-Request-Id"
	HeaderRequestIDAlt   = "X-Request-ID"
	HeaderCorrelationID  = "X-Correlation-Id"
	HeaderCorrelationAlt = "X-Correlation-ID"
)

func Correlation() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := firstHeader(r, HeaderRequestID, HeaderRequestIDAlt)
			if requestID == "" {
				requestID = newID()
				r.Header.Set(HeaderRequestID, requestID)
			}

			correlationID := firstHeader(r, HeaderCorrelationID, HeaderCorrelationAlt)
			if correlationID == "" {
				correlationID = requestID
				r.Header.Set(HeaderCorrelationID, correlationID)
			}

			w.Header().Set(HeaderRequestID, requestID)
			w.Header().Set(HeaderCorrelationID, correlationID)

			next.ServeHTTP(w, r)
		})
	}
}

func firstHeader(r *http.Request, keys ...string) string {
	for _, key := range keys {
		if value := r.Header.Get(key); value != "" {
			return value
		}
	}
	return ""
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "local-request"
	}
	return hex.EncodeToString(b[:])
}
