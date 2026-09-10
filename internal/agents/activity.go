package agents

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxActivityEvents = 32

type ActivityContext struct {
	PaneID  string `json:"paneId"`
	Kind    string `json:"kind,omitempty"`
	Session string `json:"session"`
	Window  int    `json:"window"`
	Pane    int    `json:"pane"`
}

type Activity struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind,omitempty"`
	Session   string    `json:"session"`
	Window    int       `json:"window"`
	Pane      int       `json:"pane"`
	Message   string    `json:"message"`
	URL       string    `json:"url"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ActivityStore struct {
	mu     sync.RWMutex
	events map[string]Activity
}

func NewActivityStore() *ActivityStore {
	return &ActivityStore{events: make(map[string]Activity)}
}

func (s *ActivityStore) Record(context ActivityContext, message, url string) error {
	if err := ValidateActivityContext(context); err != nil {
		return err
	}
	event := Activity{
		ID:        context.PaneID,
		Kind:      context.Kind,
		Session:   context.Session,
		Window:    context.Window,
		Pane:      context.Pane,
		Message:   message,
		URL:       url,
		UpdatedAt: time.Now().UTC(),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.events[event.ID]; !exists && len(s.events) >= maxActivityEvents {
		var oldestID string
		var oldestTime time.Time
		for id, existing := range s.events {
			if oldestID == "" || existing.UpdatedAt.Before(oldestTime) {
				oldestID = id
				oldestTime = existing.UpdatedAt
			}
		}
		delete(s.events, oldestID)
	}
	s.events[event.ID] = event
	return nil
}

func (s *ActivityStore) List() []Activity {
	s.mu.RLock()
	activities := make([]Activity, 0, len(s.events))
	for _, event := range s.events {
		activities = append(activities, event)
	}
	s.mu.RUnlock()

	sort.Slice(activities, func(i, j int) bool {
		return activities[i].UpdatedAt.After(activities[j].UpdatedAt)
	})
	return activities
}

func (s *ActivityStore) Dismiss(id string) {
	s.mu.Lock()
	delete(s.events, id)
	s.mu.Unlock()
}

func ValidateActivityContext(context ActivityContext) error {
	if !validActivityPaneID(context.PaneID) || strings.TrimSpace(context.Session) == "" || len(context.Session) > 256 {
		return errors.New("invalid agent activity context")
	}
	if context.Window < 0 || context.Pane < 0 {
		return errors.New("invalid agent activity context")
	}
	switch context.Kind {
	case "", "codex", "claude", "opencode", "pi", "hermes":
		return nil
	default:
		return errors.New("invalid agent activity context")
	}
}

func validActivityPaneID(value string) bool {
	if !strings.HasPrefix(value, "%") || len(value) == 1 {
		return false
	}
	_, err := strconv.ParseUint(value[1:], 10, 64)
	return err == nil
}
