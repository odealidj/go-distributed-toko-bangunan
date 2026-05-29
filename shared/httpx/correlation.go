package httpx

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/odealidj/go-distributed-toko-bangunan/shared/observability"
)

const (
	HeaderRequestID      = observability.HeaderRequestID
	HeaderRequestIDAlt   = observability.HeaderRequestIDAlt
	HeaderCorrelationID  = observability.HeaderCorrelationID
	HeaderCorrelationAlt = observability.HeaderCorrelationAlt
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

			next.ServeHTTP(w, r.WithContext(observability.WithRequestScope(r.Context(), requestID, correlationID)))
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
