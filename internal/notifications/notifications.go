package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
)

var ErrNoSubscriptions = errors.New("no notification devices are enabled")

type Subscription struct {
	Endpoint string `json:"endpoint"`
	Keys     Keys   `json:"keys"`
}

type Keys struct {
	Auth   string `json:"auth"`
	P256dh string `json:"p256dh"`
}

type Notification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
	Tag   string `json:"tag,omitempty"`
}

type fileState struct {
	VAPIDPrivateKey string         `json:"vapidPrivateKey"`
	VAPIDPublicKey  string         `json:"vapidPublicKey"`
	Subscriptions   []Subscription `json:"subscriptions"`
}

type Store struct {
	mu     sync.RWMutex
	path   string
	state  fileState
	client webpush.HTTPClient
}

func Load(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
		if err != nil {
			return nil, fmt.Errorf("generate notification keys: %w", err)
		}
		store := &Store{
			path: path,
			state: fileState{
				VAPIDPrivateKey: privateKey,
				VAPIDPublicKey:  publicKey,
				Subscriptions:   []Subscription{},
			},
			client: &http.Client{Timeout: 10 * time.Second},
		}
		if err := writeFile(path, store.state); err != nil {
			return nil, err
		}
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read notifications: %w", err)
	}

	var state fileState
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&state); err != nil {
		return nil, fmt.Errorf("decode notifications: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return nil, fmt.Errorf("decode notifications: %w", err)
	}
	if state.VAPIDPrivateKey == "" || state.VAPIDPublicKey == "" {
		return nil, errors.New("notification keys are missing")
	}
	for _, subscription := range state.Subscriptions {
		if err := validateSubscription(subscription); err != nil {
			return nil, fmt.Errorf("invalid saved notification subscription: %w", err)
		}
	}

	return &Store{
		path:   path,
		state:  state,
		client: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (s *Store) PublicKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.VAPIDPublicKey
}

func (s *Store) Subscribe(subscription Subscription) error {
	if err := validateSubscription(subscription); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	next := append([]Subscription(nil), s.state.Subscriptions...)
	for index := range next {
		if next[index].Endpoint == subscription.Endpoint {
			next[index] = subscription
			return s.saveSubscriptions(next)
		}
	}
	if len(next) >= 32 {
		return errors.New("notification device limit reached")
	}
	next = append(next, subscription)
	return s.saveSubscriptions(next)
}

func (s *Store) Unsubscribe(endpoint string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := make([]Subscription, 0, len(s.state.Subscriptions))
	for _, subscription := range s.state.Subscriptions {
		if subscription.Endpoint != endpoint {
			next = append(next, subscription)
		}
	}
	if len(next) == len(s.state.Subscriptions) {
		return nil
	}
	return s.saveSubscriptions(next)
}

func (s *Store) Send(ctx context.Context, notification Notification) (int, error) {
	s.mu.RLock()
	subscriptions := append([]Subscription(nil), s.state.Subscriptions...)
	privateKey := s.state.VAPIDPrivateKey
	publicKey := s.state.VAPIDPublicKey
	s.mu.RUnlock()

	if len(subscriptions) == 0 {
		return 0, ErrNoSubscriptions
	}
	payload, err := json.Marshal(notification)
	if err != nil {
		return 0, fmt.Errorf("encode notification: %w", err)
	}

	sent := 0
	var sendErrors []error
	for _, subscription := range subscriptions {
		if err := s.send(ctx, subscription, payload, privateKey, publicKey); err != nil {
			sendErrors = append(sendErrors, err)
			continue
		}
		sent++
	}
	return sent, errors.Join(sendErrors...)
}

func (s *Store) send(ctx context.Context, subscription Subscription, payload []byte, privateKey, publicKey string) error {
	response, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
		Endpoint: subscription.Endpoint,
		Keys: webpush.Keys{
			Auth:   subscription.Keys.Auth,
			P256dh: subscription.Keys.P256dh,
		},
	}, &webpush.Options{
		HTTPClient:      s.client,
		Subscriber:      "https://github.com/sriramsme/pickle",
		VAPIDPrivateKey: privateKey,
		VAPIDPublicKey:  publicKey,
		TTL:             60,
		Urgency:         webpush.UrgencyNormal,
	})
	if err != nil {
		return errors.New("push request failed")
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone {
		_ = s.Unsubscribe(subscription.Endpoint)
		return errors.New("notification subscription expired")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("push service returned %s", response.Status)
	}
	return nil
}

func (s *Store) saveSubscriptions(subscriptions []Subscription) error {
	next := s.state
	next.Subscriptions = subscriptions
	if err := writeFile(s.path, next); err != nil {
		return err
	}
	s.state = next
	return nil
}

func validateSubscription(subscription Subscription) error {
	if len(subscription.Endpoint) > 4096 || len(subscription.Keys.Auth) > 512 || len(subscription.Keys.P256dh) > 512 {
		return errors.New("notification subscription is too large")
	}
	endpoint, err := url.Parse(subscription.Endpoint)
	if err != nil || endpoint.Scheme != "https" || endpoint.Host == "" {
		return errors.New("notification endpoint must be an HTTPS URL")
	}
	if subscription.Keys.Auth == "" || subscription.Keys.P256dh == "" {
		return errors.New("notification subscription keys are missing")
	}
	return nil
}

func writeFile(path string, state fileState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode notifications: %w", err)
	}
	data = append(data, '\n')

	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create notification directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".notifications-*.tmp")
	if err != nil {
		return fmt.Errorf("create notifications: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)

	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set notification permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write notifications: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close notifications: %w", err)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return fmt.Errorf("save notifications: %w", err)
	}
	return nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("unexpected data after notifications")
		}
		return err
	}
	return nil
}
