package notify_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jmsnll/cardimportd/internal/notify"
)

type funcNotifier func(ctx context.Context, event notify.Event) error

func (f funcNotifier) Notify(ctx context.Context, event notify.Event) error { return f(ctx, event) }

func TestLogNotifier_CoreFields(t *testing.T) {
	var buf bytes.Buffer
	n := notify.NewLogNotifier(slog.New(slog.NewJSONHandler(&buf, nil)))
	evt := notify.Event{
		Kind:      notify.KindImportStarted,
		CardUUID:  "1A2B-3C4D",
		Owner:     "James",
		MountPath: "/volumeUSB1/usbshare1",
		Detail:    "starting",
		Time:      time.Now(),
	}
	if err := n.Notify(context.Background(), evt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		`"kind":"import_started"`,
		`"card_uuid":"1A2B-3C4D"`,
		`"owner":"James"`,
		`"mount_path":"/volumeUSB1/usbshare1"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot: %s", want, out)
		}
	}
}

func TestLogNotifier_StatsFields(t *testing.T) {
	var buf bytes.Buffer
	n := notify.NewLogNotifier(slog.New(slog.NewJSONHandler(&buf, nil)))
	evt := notify.Event{
		Kind: notify.KindImportCompleted,
		Stats: &notify.ImportStats{
			Total: 10, Imported: 9, Skipped: 1, BytesCopied: 1024, Duration: 5 * time.Second,
		},
	}
	n.Notify(context.Background(), evt)
	out := buf.String()
	for _, want := range []string{`"total":10`, `"imported":9`, `"duration_ms":5000`} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot: %s", want, out)
		}
	}
}

func TestLogNotifier_NilStats_NoStatsFields(t *testing.T) {
	var buf bytes.Buffer
	n := notify.NewLogNotifier(slog.New(slog.NewJSONHandler(&buf, nil)))
	n.Notify(context.Background(), notify.Event{Kind: notify.KindNewCardPending})
	out := buf.String()
	for _, absent := range []string{`"total"`, `"imported"`, `"bytes_copied"`} {
		if strings.Contains(out, absent) {
			t.Errorf("output should not contain %q when Stats is nil\ngot: %s", absent, out)
		}
	}
}

func TestMultiNotifier_CallsAll(t *testing.T) {
	called := make([]bool, 3)
	ns := make([]notify.Notifier, 3)
	for i := range ns {
		i := i
		ns[i] = funcNotifier(func(_ context.Context, _ notify.Event) error { called[i] = true; return nil })
	}
	notify.NewMultiNotifier(ns...).Notify(context.Background(), notify.Event{})
	for i, c := range called {
		if !c {
			t.Errorf("notifier[%d] not called", i)
		}
	}
}

func TestMultiNotifier_ErrorAggregation(t *testing.T) {
	errA, errB := errors.New("A"), errors.New("B")
	called := make([]bool, 3)
	m := notify.NewMultiNotifier(
		funcNotifier(func(_ context.Context, _ notify.Event) error { called[0] = true; return errA }),
		funcNotifier(func(_ context.Context, _ notify.Event) error { called[1] = true; return errB }),
		funcNotifier(func(_ context.Context, _ notify.Event) error { called[2] = true; return nil }),
	)
	err := m.Notify(context.Background(), notify.Event{})
	if !errors.Is(err, errA) || !errors.Is(err, errB) {
		t.Errorf("expected both errors in result, got: %v", err)
	}
	for i, c := range called {
		if !c {
			t.Errorf("notifier[%d] not called", i)
		}
	}
}

func TestMultiNotifier_Empty(t *testing.T) {
	if err := notify.NewMultiNotifier().Notify(context.Background(), notify.Event{}); err != nil {
		t.Fatalf("expected nil for empty MultiNotifier, got: %v", err)
	}
}

func TestEvent_CardLabelZeroValue(t *testing.T) {
	// An Event constructed without setting CardLabel must have the zero value "".
	evt := notify.Event{
		Kind:     notify.KindImportStarted,
		CardUUID: "AABB-CCDD",
		Owner:    "James",
	}
	if evt.CardLabel != "" {
		t.Errorf("CardLabel = %q, want empty string when not set", evt.CardLabel)
	}
}

func TestEvent_CardLabelRoundTrip(t *testing.T) {
	label := "Fujifilm X-T5"
	var received notify.Event
	n := funcNotifier(func(_ context.Context, e notify.Event) error {
		received = e
		return nil
	})
	evt := notify.Event{
		Kind:      notify.KindImportStarted,
		CardUUID:  "AABB-CCDD",
		Owner:     "James",
		CardLabel: label,
	}
	if err := n.Notify(context.Background(), evt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received.CardLabel != label {
		t.Errorf("CardLabel = %q, want %q", received.CardLabel, label)
	}
}
