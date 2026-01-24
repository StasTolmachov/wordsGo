package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"wordsGo/slogger"
)

func TestLoggerMiddleware(t *testing.T) {
	slogger.MakeLogger(true)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggerMiddleware(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	requestID := rr.Header().Get("X-Request-ID")
	assert.NotEmpty(t, requestID, "X-Request-ID header should be present")
}
