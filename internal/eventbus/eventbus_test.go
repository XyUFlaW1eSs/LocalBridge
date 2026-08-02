package eventbus

import (
	"context"
	"testing"
	"time"
)

func TestPublishAndUnsubscribe(t *testing.T) {
	bus := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := bus.Subscribe(ctx, ClipboardChanged, 1)
	bus.Publish(Event{Type: ClipboardChanged, Data: "hello"})
	select {
	case event := <-ch:
		if event.Data != "hello" || event.Time.IsZero() {
			t.Fatalf("unexpected event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("event was not delivered")
	}
	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("subscription channel should be closed")
		}
	case <-time.After(time.Second):
		t.Fatal("subscription was not removed")
	}
}
