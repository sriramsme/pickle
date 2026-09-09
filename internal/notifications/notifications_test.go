package notifications

import (
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestStorePersistsSubscriptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pickle", "notifications.json")
	store, err := Load(path)
	if err != nil {
		t.Fatalf("load notifications: %v", err)
	}
	subscription := testSubscription("https://push.example/one")
	if err := store.Subscribe(subscription); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("reload notifications: %v", err)
	}
	if reloaded.PublicKey() != store.PublicKey() {
		t.Fatal("public key changed after reload")
	}
	if err := reloaded.Unsubscribe(subscription.Endpoint); err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}
	if _, err := reloaded.Send(t.Context(), testNotification()); !errors.Is(err, ErrNoSubscriptions) {
		t.Fatalf("unexpected send error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("inspect notification file: %v", err)
	}
	if permissions := info.Mode().Perm(); permissions != 0o600 {
		t.Fatalf("unexpected notification permissions: %o", permissions)
	}
}

func TestStoreReplacesMatchingSubscription(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notifications.json")
	store, err := Load(path)
	if err != nil {
		t.Fatalf("load notifications: %v", err)
	}
	subscription := testSubscription("https://push.example/device")
	if err := store.Subscribe(subscription); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	subscription.Keys.Auth = "new-auth"
	if err := store.Subscribe(subscription); err != nil {
		t.Fatalf("replace subscription: %v", err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("reload notifications: %v", err)
	}
	if len(reloaded.state.Subscriptions) != 1 || reloaded.state.Subscriptions[0].Keys.Auth != "new-auth" {
		t.Fatalf("unexpected subscriptions: %+v", reloaded.state.Subscriptions)
	}
}

func TestStoreRejectsInvalidSubscription(t *testing.T) {
	store, err := Load(filepath.Join(t.TempDir(), "notifications.json"))
	if err != nil {
		t.Fatalf("load notifications: %v", err)
	}
	if err := store.Subscribe(testSubscription("http://push.example/device")); err == nil {
		t.Fatal("expected insecure endpoint to be rejected")
	}
}

func TestStoreSendsEncryptedWebPushRequest(t *testing.T) {
	pushServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") != "aes128gcm" || r.Header.Get("Authorization") == "" {
			t.Error("missing Web Push headers")
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer pushServer.Close()

	store, err := Load(filepath.Join(t.TempDir(), "notifications.json"))
	if err != nil {
		t.Fatalf("load notifications: %v", err)
	}
	store.client = pushServer.Client()
	_, x, y, err := elliptic.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate subscription key: %v", err)
	}
	subscription := Subscription{
		Endpoint: pushServer.URL,
		Keys: Keys{
			Auth:   base64.RawURLEncoding.EncodeToString(make([]byte, 16)),
			P256dh: base64.RawURLEncoding.EncodeToString(elliptic.Marshal(elliptic.P256(), x, y)),
		},
	}
	if err := store.Subscribe(subscription); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if sent, err := store.Send(t.Context(), testNotification()); err != nil || sent != 1 {
		t.Fatalf("send notification: %v", err)
	}
}

func TestValidateRejectsExternalNotificationURL(t *testing.T) {
	notification := testNotification()
	notification.URL = "https://example.com"
	if err := Validate(notification); !errors.Is(err, ErrInvalidNotification) {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func testSubscription(endpoint string) Subscription {
	return Subscription{
		Endpoint: endpoint,
		Keys: Keys{
			Auth:   "auth",
			P256dh: "p256dh",
		},
	}
}

func testNotification() Notification {
	return Notification{
		Title:   "Pickle",
		Body:    "Finished",
		URL:     "/",
		Urgency: UrgencyNormal,
	}
}
