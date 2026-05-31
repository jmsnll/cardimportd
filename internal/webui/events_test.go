package webui

import (
	"testing"
	"time"
)

func TestEventBus_PublishReceive(t *testing.T) {
	bus := NewEventBus(4)
	ch := bus.Subscribe()
	defer bus.Unsubscribe(ch)
	bus.Publish(ProgressEvent{Kind: ProgressKindStarted, Owner: "James"})
	select {
	case got := <-ch:
		if got.Owner != "James" {
			t.Errorf("Owner = %q", got.Owner)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestEventBus_MultiSubscribers(t *testing.T) {
	bus := NewEventBus(4)
	ch1, ch2 := bus.Subscribe(), bus.Subscribe()
	defer bus.Unsubscribe(ch1)
	defer bus.Unsubscribe(ch2)
	bus.Publish(ProgressEvent{Kind: ProgressKindCompleted, Owner: "Alice"})
	for _, ch := range []chan ProgressEvent{ch1, ch2} {
		select {
		case e := <-ch:
			if e.Owner != "Alice" {
				t.Errorf("Owner = %q", e.Owner)
			}
		case <-time.After(time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestEventBus_SlowSubNotBlocked(t *testing.T) {
	bus := NewEventBus(1)
	ch := bus.Subscribe()
	defer bus.Unsubscribe(ch)
	for i := 0; i < 20; i++ {
		bus.Publish(ProgressEvent{Kind: ProgressKindProgress})
	}
}

func TestProgressEvent_Marshal(t *testing.T) {
	data, err := ProgressEvent{Kind: ProgressKindCompleted, Owner: "Bob"}.Marshal()
	if err != nil || len(data) == 0 {
		t.Errorf("marshal failed: %v", err)
	}
}
