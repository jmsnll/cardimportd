package notify

import (
	"strings"
	"testing"
	"time"
)

func TestFormatEvent(t *testing.T) {
	stats := &ImportStats{Imported: 42, Skipped: 3, Failed: 1, Duration: 90 * time.Second}

	cases := []struct {
		name        string
		event       Event
		wantTitle   string
		wantMsgPart string
	}{
		{
			name:        "new card pending",
			event:       Event{Kind: KindNewCardPending, CardUUID: "AABB-1234"},
			wantTitle:   "cardimportd: new card",
			wantMsgPart: "AABB-1234",
		},
		{
			name:        "import started",
			event:       Event{Kind: KindImportStarted, Owner: "James"},
			wantTitle:   "cardimportd: import started",
			wantMsgPart: "James",
		},
		{
			name:        "import completed with stats",
			event:       Event{Kind: KindImportCompleted, Owner: "Sophie", Stats: stats},
			wantTitle:   "Sophie imported",
			wantMsgPart: "42",
		},
		{
			name:        "import completed without stats",
			event:       Event{Kind: KindImportCompleted},
			wantTitle:   "cardimportd: import complete",
			wantMsgPart: "Import finished",
		},
		{
			name:        "import failed",
			event:       Event{Kind: KindImportFailed, Owner: "James", Detail: "disk full"},
			wantTitle:   "FAILED",
			wantMsgPart: "disk full",
		},
		{
			name:        "unknown kind",
			event:       Event{Kind: "mystery_event"},
			wantTitle:   "cardimportd",
			wantMsgPart: "mystery_event",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			title, msg := formatEvent(tc.event)
			if !strings.Contains(title, tc.wantTitle) {
				t.Errorf("title = %q, want to contain %q", title, tc.wantTitle)
			}
			if !strings.Contains(title+msg, tc.wantMsgPart) {
				t.Errorf("title+msg = %q, want to contain %q", title+msg, tc.wantMsgPart)
			}
		})
	}
}

func TestPriorityFor(t *testing.T) {
	cases := []struct {
		kind EventKind
		want int
	}{
		{KindImportFailed, 1},
		{KindImportStarted, -1},
		{KindImportCompleted, 0},
		{KindNewCardPending, 0},
	}
	for _, tc := range cases {
		if got := priorityFor(tc.kind); got != tc.want {
			t.Errorf("priorityFor(%q) = %d, want %d", tc.kind, got, tc.want)
		}
	}
}
