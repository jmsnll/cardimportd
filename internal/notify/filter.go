package notify

import "context"

type filteredNotifier struct {
	inner Notifier
	allow map[EventKind]bool
}

// NewFilteredNotifier wraps n so that only events whose Kind is in the allow
// list are forwarded. If events is empty, n is returned unwrapped so there is
// no overhead when filtering is not configured.
func NewFilteredNotifier(n Notifier, events []string) Notifier {
	if len(events) == 0 {
		return n
	}
	allow := make(map[EventKind]bool, len(events))
	for _, e := range events {
		allow[EventKind(e)] = true
	}
	return &filteredNotifier{inner: n, allow: allow}
}

func (f *filteredNotifier) Notify(ctx context.Context, event Event) error {
	if !f.allow[event.Kind] {
		return nil
	}
	return f.inner.Notify(ctx, event)
}
