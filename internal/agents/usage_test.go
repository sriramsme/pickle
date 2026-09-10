package agents

import (
	"testing"
	"time"
)

func TestParseCodexUsage(t *testing.T) {
	usage, err := parseCodexUsage([]byte(`{
		"rateLimits": {
			"planType": "plus",
			"primary": {"usedPercent": 31, "windowDurationMins": 300, "resetsAt": 1789067779},
			"secondary": {"usedPercent": 71, "windowDurationMins": 10080, "resetsAt": 1789472632}
		}
	}`))
	if err != nil {
		t.Fatalf("parse usage: %v", err)
	}
	if usage.Kind != "codex" || usage.Plan != "plus" || len(usage.Windows) != 2 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
	if usage.Windows[0].UsedPercent != 31 || usage.Windows[0].DurationMinutes != 300 {
		t.Fatalf("unexpected primary window: %+v", usage.Windows[0])
	}
	wantReset := time.Unix(1789472632, 0).UTC()
	if usage.Windows[1].ResetsAt == nil || !usage.Windows[1].ResetsAt.Equal(wantReset) {
		t.Fatalf("unexpected secondary reset: %+v", usage.Windows[1].ResetsAt)
	}
}

func TestReadUsageWithoutCodex(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	usage, err := ReadUsage(t.Context())
	if err != nil {
		t.Fatalf("read usage: %v", err)
	}
	if len(usage) != 0 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}
