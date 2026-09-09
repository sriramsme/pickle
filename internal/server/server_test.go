package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sriramsme/pickle/internal/config"
	"github.com/sriramsme/pickle/internal/notifications"
)

type fakeNotificationService struct {
	publicKey    string
	subscription notifications.Subscription
	sentTo       string
}

func (f *fakeNotificationService) PublicKey() string {
	return f.publicKey
}

func (f *fakeNotificationService) Subscribe(subscription notifications.Subscription) error {
	f.subscription = subscription
	return nil
}

func (f *fakeNotificationService) Unsubscribe(endpoint string) error {
	if f.subscription.Endpoint == endpoint {
		f.subscription = notifications.Subscription{}
	}
	return nil
}

func (f *fakeNotificationService) Send(_ context.Context, _ notifications.Notification) (int, error) {
	f.sentTo = "all"
	return 1, nil
}

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

func TestNotificationHandlerSubscribesAndSendsTest(t *testing.T) {
	service := &fakeNotificationService{publicKey: "public-key"}
	handler := notificationHandler(service)
	subscription := `{"endpoint":"https://push.example/device","keys":{"auth":"auth","p256dh":"p256dh"}}`
	request := httptest.NewRequest(http.MethodPut, "/api/notifications", strings.NewReader(subscription))
	request.Header.Set("Origin", "http://example.com")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK || service.subscription.Endpoint != "https://push.example/device" {
		t.Fatalf("unexpected subscription response: %d, %+v", response.Code, service.subscription)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/notifications", nil)
	request.Header.Set("Origin", "http://example.com")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK || service.sentTo != "all" {
		t.Fatalf("unexpected test response: %d, %q", response.Code, service.sentTo)
	}
}

func TestNotificationHandlerRejectsCrossOriginMutation(t *testing.T) {
	request := httptest.NewRequest(http.MethodPut, "/api/notifications", strings.NewReader(`{}`))
	request.Header.Set("Origin", "https://other.example")
	response := httptest.NewRecorder()

	notificationHandler(&fakeNotificationService{}).ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("unexpected status: %d", response.Code)
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
