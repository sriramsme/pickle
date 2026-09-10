package agents

import "testing"

func TestActivityStoreReplacesAndDismissesPaneEvent(t *testing.T) {
	store := NewActivityStore()
	context := ActivityContext{PaneID: "%7", Kind: "codex", Session: "work", Window: 1, Pane: 2}
	if err := store.Record(context, "First", "/tmux/work"); err != nil {
		t.Fatalf("record first activity: %v", err)
	}
	if err := store.Record(context, "Ready", "/tmux/work"); err != nil {
		t.Fatalf("replace activity: %v", err)
	}

	activities := store.List()
	if len(activities) != 1 || activities[0].Message != "Ready" || activities[0].Kind != "codex" {
		t.Fatalf("unexpected activities: %+v", activities)
	}
	store.Dismiss("%7")
	if activities := store.List(); len(activities) != 0 {
		t.Fatalf("activity was not dismissed: %+v", activities)
	}
}

func TestActivityStoreRejectsInvalidContext(t *testing.T) {
	store := NewActivityStore()
	context := ActivityContext{PaneID: "7", Kind: "codex", Session: "work", Window: 1, Pane: 2}
	if err := store.Record(context, "Ready", "/tmux/work"); err == nil {
		t.Fatal("expected an error")
	}
}
