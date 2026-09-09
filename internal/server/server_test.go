package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServiceHandlerRejectsUnsupportedMethod(t *testing.T) {
	request := httptest.NewRequest(http.MethodPut, "/api/services", nil)
	response := httptest.NewRecorder()

	serviceHandler(t.TempDir()).ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: %d", response.Code)
	}
	if allow := response.Header().Get("Allow"); allow != "GET, POST" {
		t.Fatalf("unexpected Allow header: %q", allow)
	}
}

func TestServiceHandlerRejectsInvalidAction(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/services", strings.NewReader(`{"action":"end"}`))
	response := httptest.NewRecorder()

	serviceHandler(t.TempDir()).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", response.Code)
	}
}
