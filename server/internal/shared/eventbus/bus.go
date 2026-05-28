package eventbus

import "context"

// Handler is called once per matching event. Errors are logged but do not propagate
// to the publisher (fire-and-forget semantics).
type Handler func(ctx context.Context, e Event) error

// Bus is the publish-subscribe abstraction. v1 implementation is in-process channels.
// Future swap target: NATS (when真实削峰信号 appears).
type Bus interface {
	// Publish enqueues an event for asynchronous fan-out. Returns immediately.
	Publish(ctx context.Context, topic string, data any) error

	// Subscribe registers a handler for a topic. Must be called before Start().
	Subscribe(topic string, h Handler)

	// Start launches worker goroutines. Call once during app boot.
	Start()

	// Close drains pending events with a max timeout, then stops workers.
	Close() error
}
