package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sriramsme/pickle/internal/agents"
	"github.com/sriramsme/pickle/internal/config"
	"github.com/sriramsme/pickle/internal/notifications"
)

type fakeNotificationService struct {
	publicKey    string
	subscription notifications.Subscription
	sentTo       string
	notification notifications.Notification
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

func (f *fakeNotificationService) Send(_ context.Context, notification notifications.Notification) (int, error) {
	f.sentTo = "all"
	f.notification = notification
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
	activityStore := agents.NewActivityStore()
	handler := notificationHandler(service, activityStore)
	subscription := `{"endpoint":"https://push.example/device","keys":{"auth":"auth","p256dh":"p256dh"}}`
	request := httptest.NewRequest(http.MethodPut, "/api/notifications", strings.NewReader(subscription))
	request.Header.Set("Origin", "http://example.com")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK || service.subscription.Endpoint != "https://push.example/device" {
		t.Fatalf("unexpected subscription response: %d, %+v", response.Code, service.subscription)
	}

	request = httptest.NewRequest(
		http.MethodPost,
		"/api/notifications",
		strings.NewReader(`{"title":"Codex","body":"Finished","url":"/tmux/pickle","tag":"codex-pickle","urgency":"high","context":{"paneId":"%7","kind":"codex","session":"pickle","window":1,"pane":2}}`),
	)
	request.Header.Set("Origin", "http://example.com")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK || service.sentTo != "all" || service.notification.Urgency != notifications.UrgencyHigh {
		t.Fatalf("unexpected test response: %d, %q", response.Code, service.sentTo)
	}
	activities := activityStore.List()
	if len(activities) != 1 || activities[0].Message != "Finished" || activities[0].Session != "pickle" {
		t.Fatalf("unexpected activities: %+v", activities)
	}
}

func TestNotificationHandlerRejectsExternalDestination(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/notifications",
		strings.NewReader(`{"title":"Pickle","body":"Finished","url":"https://example.com","urgency":"normal"}`),
	)
	response := httptest.NewRecorder()

	notificationHandler(&fakeNotificationService{}, agents.NewActivityStore()).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", response.Code)
	}
}

func TestNotificationHandlerRejectsCrossOriginMutation(t *testing.T) {
	request := httptest.NewRequest(http.MethodPut, "/api/notifications", strings.NewReader(`{}`))
	request.Header.Set("Origin", "https://other.example")
	response := httptest.NewRecorder()

	notificationHandler(&fakeNotificationService{}, agents.NewActivityStore()).ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("unexpected status: %d", response.Code)
	}
}

func TestAgentActivityHandlerListsAndDismisses(t *testing.T) {
	store := agents.NewActivityStore()
	context := agents.ActivityContext{PaneID: "%7", Kind: "codex", Session: "pickle", Window: 1, Pane: 2}
	if err := store.Record(context, "Finished", "/tmux/pickle"); err != nil {
		t.Fatalf("record activity: %v", err)
	}
	handler := agentActivityHandler(store)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/agent-activity", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"message":"Finished"`) {
		t.Fatalf("unexpected list response: %d, %s", response.Code, response.Body.String())
	}

	request := httptest.NewRequest(http.MethodDelete, "/api/agent-activity?id=%257", nil)
	request.Header.Set("Origin", "http://example.com")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || len(store.List()) != 0 {
		t.Fatalf("unexpected dismiss response: %d, %s", response.Code, response.Body.String())
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
