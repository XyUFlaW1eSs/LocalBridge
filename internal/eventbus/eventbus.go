package eventbus

import (
	"context"
	"sync"
	"time"
)

const (
	ClipboardChanged     = "clipboard.changed"
	DeviceConnected      = "device.connected"
	DeviceDisconnected   = "device.disconnected"
	FileSent             = "file.sent"
	FileReceived         = "file.received"
	ImageReceived        = "image.received"
	NotificationReceived = "notification.received"
)

type Event struct {
	Type string    `json:"type"`
	Time time.Time `json:"time"`
	Data any       `json:"data,omitempty"`
}

type subscription struct {
	id uint64
	ch chan Event
}

type Bus struct {
	mu     sync.RWMutex
	nextID uint64
	subs   map[string]map[uint64]subscription
}

func New() *Bus { return &Bus{subs: make(map[string]map[uint64]subscription)} }

func (b *Bus) Publish(event Event) {
	if event.Time.IsZero() {
		event.Time = time.Now().UTC()
	}
	b.mu.RLock()
	for _, sub := range b.subs[event.Type] {
		select {
		case sub.ch <- event:
		default:
			// A slow subscriber must not block the application event path.
		}
	}
	b.mu.RUnlock()
}

func (b *Bus) Subscribe(ctx context.Context, eventType string, buffer int) <-chan Event {
	if buffer < 1 {
		buffer = 1
	}
	b.mu.Lock()
	b.nextID++
	id := b.nextID
	if b.subs[eventType] == nil {
		b.subs[eventType] = make(map[uint64]subscription)
	}
	ch := make(chan Event, buffer)
	b.subs[eventType][id] = subscription{id: id, ch: ch}
	b.mu.Unlock()
	go func() {
		<-ctx.Done()
		b.mu.Lock()
		if subs := b.subs[eventType]; subs != nil {
			delete(subs, id)
			if len(subs) == 0 {
				delete(b.subs, eventType)
			}
		}
		close(ch)
		b.mu.Unlock()
	}()
	return ch
}
