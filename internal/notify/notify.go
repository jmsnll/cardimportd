// Package notify defines the Notifier interface and concrete implementations.
// Future push adapters (Pushover, ntfy, Slack) implement Notifier and compose
// via NewMultiNotifier without changing any other package.
package notify

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// EventKind classifies the lifecycle event being reported.
type EventKind string

const (
	KindNewCardPending  EventKind = "new_card_pending"
	KindImportStarted   EventKind = "import_started"
	KindImportCompleted EventKind = "import_completed"
	KindImportFailed    EventKind = "import_failed"
)

// ImportStats carries aggregate counters for a completed or failed import run.
type ImportStats struct {
	Total       int
	Imported    int
	Skipped     int
	Failed      int
	BytesCopied int64
	Duration    time.Duration
}

// Event describes a single lifecycle occurrence within the daemon.
type Event struct {
	Kind      EventKind
	CardUUID  string
	Owner     string
	CardLabel string
	MountPath string
	Time      time.Time
	Detail    string
	Stats     *ImportStats
}

// Notifier is the delivery contract for daemon lifecycle events.
type Notifier interface {
	Notify(ctx context.Context, event Event) error
}

// -- LogNotifier -------------------------------------------------------------

type logNotifier struct{ logger *slog.Logger }

// NewLogNotifier returns a Notifier that records events as structured slog
// records at Info level. Uses LogAttrs so the caller controls the logger.
func NewLogNotifier(logger *slog.Logger) Notifier {
	return &logNotifier{logger: logger}
}

func (n *logNotifier) Notify(ctx context.Context, event Event) error {
	attrs := []slog.Attr{
		slog.String("kind", string(event.Kind)),
		slog.String("card_uuid", event.CardUUID),
		slog.String("owner", event.Owner),
		slog.String("mount_path", event.MountPath),
		slog.String("detail", event.Detail),
	}
	if s := event.Stats; s != nil {
		attrs = append(attrs,
			slog.Int("total", s.Total),
			slog.Int("imported", s.Imported),
			slog.Int("skipped", s.Skipped),
			slog.Int("failed", s.Failed),
			slog.Int64("bytes_copied", s.BytesCopied),
			slog.Int64("duration_ms", s.Duration.Milliseconds()),
		)
	}
	n.logger.LogAttrs(ctx, slog.LevelInfo, string(event.Kind), attrs...)
	return nil
}

// -- MultiNotifier -----------------------------------------------------------

type multiNotifier struct{ notifiers []Notifier }

// NewMultiNotifier fans out each event to every inner Notifier in order.
// All notifiers run even when earlier ones fail; errors are joined.
func NewMultiNotifier(notifiers ...Notifier) Notifier {
	cp := make([]Notifier, len(notifiers))
	copy(cp, notifiers)
	return &multiNotifier{notifiers: cp}
}

func (m *multiNotifier) Notify(ctx context.Context, event Event) error {
	var errs []error
	for _, n := range m.notifiers {
		if err := n.Notify(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
