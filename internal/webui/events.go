package webui

import (
	"encoding/json"
	"sync"
)

type ProgressEventKind string

const (
	ProgressKindStarted   ProgressEventKind = "import_started"
	ProgressKindProgress  ProgressEventKind = "import_progress"
	ProgressKindCompleted ProgressEventKind = "import_completed"
	ProgressKindFailed    ProgressEventKind = "import_failed"
)

type ProgressEvent struct {
	Kind        ProgressEventKind `json:"event"`
	Owner       string            `json:"owner"`
	CardUUID    string            `json:"card_uuid"`
	Total       int               `json:"total"`
	Imported    int               `json:"imported"`
	Skipped     int               `json:"skipped"`
	Failed      int               `json:"failed"`
	BytesCopied int64             `json:"bytes_copied"`
	Error       string            `json:"error,omitempty"`
}

func (e ProgressEvent) Marshal() ([]byte, error) { return json.Marshal(e) }

type EventBus struct {
	mu      sync.RWMutex
	clients map[chan ProgressEvent]struct{}
	bufSize int
}

func NewEventBus(bufSize int) *EventBus {
	return &EventBus{clients: make(map[chan ProgressEvent]struct{}), bufSize: bufSize}
}

func (b *EventBus) Subscribe() chan ProgressEvent {
	ch := make(chan ProgressEvent, b.bufSize)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *EventBus) Unsubscribe(ch chan ProgressEvent) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
	close(ch)
}

func (b *EventBus) Publish(e ProgressEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- e:
		default:
		}
	}
}
