package history_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jmsnll/cardimportd/internal/history"
)

func TestAppendRecent(t *testing.T) {
	log := history.New(filepath.Join(t.TempDir(), "h.jsonl"))
	e1 := history.Entry{UUID: "A", Owner: "James", StartedAt: time.Now(), CompletedAt: time.Now()}
	e2 := history.Entry{UUID: "B", Owner: "Sophie", StartedAt: time.Now(), CompletedAt: time.Now()}
	log.Append(e1)
	log.Append(e2)
	entries, err := log.Recent(0)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d, want 2", len(entries))
	}
	if entries[0].UUID != "B" {
		t.Errorf("want most-recent first, got %q", entries[0].UUID)
	}
}

func TestRecent_Limit(t *testing.T) {
	log := history.New(filepath.Join(t.TempDir(), "h.jsonl"))
	for i := 0; i < 10; i++ {
		log.Append(history.Entry{UUID: "X", StartedAt: time.Now(), CompletedAt: time.Now()})
	}
	entries, _ := log.Recent(3)
	if len(entries) != 3 {
		t.Errorf("got %d, want 3", len(entries))
	}
}

func TestAppend_TrimsAtLimit(t *testing.T) {
	log := history.New(filepath.Join(t.TempDir(), "h.jsonl"))
	for i := 0; i < 10005; i++ {
		if err := log.Append(history.Entry{UUID: "X", StartedAt: time.Now(), CompletedAt: time.Now()}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}
	entries, err := log.Recent(0)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(entries) > 10000 {
		t.Errorf("got %d entries, want <= 10000", len(entries))
	}
}

func TestRecent_Missing(t *testing.T) {
	log := history.New(filepath.Join(t.TempDir(), "missing.jsonl"))
	entries, err := log.Recent(10)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d, want 0", len(entries))
	}
}
