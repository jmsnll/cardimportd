package notify

import (
	"context"
	"testing"
)

type recordingNotifier struct{ kinds []EventKind }

func (r *recordingNotifier) Notify(_ context.Context, e Event) error {
	r.kinds = append(r.kinds, e.Kind)
	return nil
}

func TestFilteredNotifier_AllowsMatchingKinds(t *testing.T) {
	rec := &recordingNotifier{}
	f := NewFilteredNotifier(rec, []string{string(KindImportCompleted), string(KindImportFailed)})

	for _, kind := range []EventKind{
		KindNewCardPending,
		KindImportStarted,
		KindImportCompleted,
		KindImportFailed,
	} {
		f.Notify(context.Background(), Event{Kind: kind})
	}

	if len(rec.kinds) != 2 {
		t.Fatalf("expected 2 events delivered, got %d: %v", len(rec.kinds), rec.kinds)
	}
	if rec.kinds[0] != KindImportCompleted || rec.kinds[1] != KindImportFailed {
		t.Errorf("unexpected kinds: %v", rec.kinds)
	}
}

func TestFilteredNotifier_EmptyEventsPassesAll(t *testing.T) {
	rec := &recordingNotifier{}
	f := NewFilteredNotifier(rec, nil)

	// When events is empty, NewFilteredNotifier returns the inner notifier
	// unwrapped — verify it is the exact same pointer.
	if f != Notifier(rec) {
		t.Error("expected unwrapped notifier when events list is empty")
	}
}
