package eventbus

import (
	"context"
	"sync"
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
	"go.uber.org/zap"
)

// InprocBus is a goroutine-pool backed bus. M1 default.
type InprocBus struct {
	workers     int
	queue       chan Event
	subscribers map[string][]Handler
	mu          sync.RWMutex
	wg          sync.WaitGroup
	closed      bool
	logger      *zap.Logger
}

// NewInprocBus creates a bus with `workers` parallel handler goroutines.
// Queue size is workers*16; tune later if backpressure shows up.
func NewInprocBus(workers int) *InprocBus {
	if workers <= 0 {
		workers = 4
	}
	return &InprocBus{
		workers:     workers,
		queue:       make(chan Event, workers*16),
		subscribers: make(map[string][]Handler),
		logger:      zap.NewNop(),
	}
}

// WithLogger swaps in a real logger (used during app wiring).
func (b *InprocBus) WithLogger(l *zap.Logger) *InprocBus {
	b.logger = l
	return b
}

func (b *InprocBus) Publish(ctx context.Context, topic string, data any) error {
	b.mu.RLock()
	closed := b.closed
	b.mu.RUnlock()
	if closed {
		return nil
	}
	tenantID, _ := kernel.TenantIDFromContext(ctx)
	actor := ""
	if mid, ok := kernel.MemberIDFromContext(ctx); ok {
		actor = string(mid)
	}
	e := Event{
		ID:         kernel.NewID(),
		Topic:      topic,
		OccurredAt: time.Now().UTC(),
		TenantID:   tenantID,
		Actor:      actor,
		TraceID:    kernel.TraceIDFromContext(ctx),
		Data:       data,
	}
	select {
	case b.queue <- e:
		return nil
	default:
		b.logger.Warn("eventbus queue full, dropping event",
			zap.String("topic", topic), zap.String("event_id", string(e.ID)))
		return nil
	}
}

func (b *InprocBus) Subscribe(topic string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[topic] = append(b.subscribers[topic], h)
}

func (b *InprocBus) Start() {
	for i := 0; i < b.workers; i++ {
		b.wg.Add(1)
		go b.worker()
	}
}

func (b *InprocBus) worker() {
	defer b.wg.Done()
	for e := range b.queue {
		b.dispatch(e)
	}
}

func (b *InprocBus) dispatch(e Event) {
	b.mu.RLock()
	handlers := b.subscribers[e.Topic]
	b.mu.RUnlock()
	for _, h := range handlers {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		ctx = kernel.WithTraceID(ctx, e.TraceID)
		if e.TenantID != "" {
			ctx = kernel.WithTenantID(ctx, e.TenantID)
		}
		if err := h(ctx, e); err != nil {
			b.logger.Error("eventbus handler error",
				zap.String("topic", e.Topic),
				zap.String("event_id", string(e.ID)),
				zap.Error(err))
		}
		cancel()
	}
}

func (b *InprocBus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.mu.Unlock()
	close(b.queue)
	done := make(chan struct{})
	go func() { b.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		b.logger.Warn("eventbus drain timeout, some events may be lost")
	}
	return nil
}
