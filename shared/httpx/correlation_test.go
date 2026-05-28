package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCorrelationGeneratesMissingHeaders(t *testing.T) {
	handler := Correlation()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(HeaderRequestID) == "" {
			t.Fatal("expected request id on request")
		}
		if r.Header.Get(HeaderCorrelationID) == "" {
			t.Fatal("expected correlation id on request")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	handler.ServeHTTP(recorder, request)

	if recorder.Header().Get(HeaderRequestID) == "" {
		t.Fatal("expected request id on response")
	}
	if recorder.Header().Get(HeaderCorrelationID) == "" {
		t.Fatal("expected correlation id on response")
	}
}

func TestCorrelationKeepsExistingHeaders(t *testing.T) {
	handler := Correlation()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(HeaderRequestID); got != "req-1" {
			t.Fatalf("expected request id req-1, got %s", got)
		}
		if got := r.Header.Get(HeaderCorrelationID); got != "corr-1" {
			t.Fatalf("expected correlation id corr-1, got %s", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set(HeaderRequestID, "req-1")
	request.Header.Set(HeaderCorrelationID, "corr-1")

	handler.ServeHTTP(recorder, request)

	if got := recorder.Header().Get(HeaderRequestID); got != "req-1" {
		t.Fatalf("expected response request id req-1, got %s", got)
	}
	if got := recorder.Header().Get(HeaderCorrelationID); got != "corr-1" {
		t.Fatalf("expected response correlation id corr-1, got %s", got)
	}
}
