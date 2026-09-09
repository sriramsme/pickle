package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sriramsme/pickle/internal/config"
)

func TestServiceHandlerRejectsUnsupportedMethod(t *testing.T) {
	request := httptest.NewRequest(http.MethodPut, "/api/services", nil)
	response := httptest.NewRecorder()

	serviceHandler(testSettings(t)).ServeHTTP(response, request)

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

	serviceHandler(testSettings(t)).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", response.Code)
	}
}

func TestSettingsHandlerUpdatesProjectsDirectory(t *testing.T) {
	settings := testSettings(t)
	projectsDirectory := t.TempDir()
	request := httptest.NewRequest(
		http.MethodPut,
		"/api/settings",
		strings.NewReader(`{"projectsDirectory":`+strconv.Quote(projectsDirectory)+`}`),
	)
	response := httptest.NewRecorder()

	settingsHandler(settings).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d: %s", response.Code, response.Body.String())
	}
	if current := settings.Current(); !current.Configured || current.ProjectsDirectory != projectsDirectory {
		t.Fatalf("unexpected settings: %+v", current)
	}
}

func testSettings(t *testing.T) *config.Store {
	t.Helper()
	store, err := config.Load(filepath.Join(t.TempDir(), "config.json"), t.TempDir(), "")
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	return store
}
