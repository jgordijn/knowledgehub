package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func makeTestRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func executeHandler(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}
