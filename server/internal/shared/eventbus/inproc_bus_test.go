package eventbus

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestInprocBus_PublishSubscribe(t *testing.T) {
	bus := NewInprocBus(4)
	defer bus.Close()

	var mu sync.Mutex
	var received []Event
	bus.Subscribe("test.foo", func(ctx context.Context, e Event) error {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
		return nil
	})

	bus.Start()
	if err := bus.Publish(context.Background(), "test.foo", map[string]string{"k": "v"}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(received)
		mu.Unlock()
		if n == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 event received, got %d", len(received))
	}
	if received[0].Topic != "test.foo" {
		t.Fatalf("topic mismatch")
	}
}

func TestInprocBus_MultipleSubscribers(t *testing.T) {
	bus := NewInprocBus(4)
	defer bus.Close()

	var count1, count2 int
	var mu sync.Mutex
	bus.Subscribe("test.bar", func(ctx context.Context, e Event) error {
		mu.Lock()
		count1++
		mu.Unlock()
		return nil
	})
	bus.Subscribe("test.bar", func(ctx context.Context, e Event) error {
		mu.Lock()
		count2++
		mu.Unlock()
		return nil
	})
	bus.Start()
	_ = bus.Publish(context.Background(), "test.bar", nil)

	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if count1 != 1 || count2 != 1 {
		t.Fatalf("expected each subscriber to receive 1, got %d / %d", count1, count2)
	}
}
