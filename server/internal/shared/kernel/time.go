package kernel

import (
	"sync"
	"time"
)

// Clock abstracts time. Injected into services so tests can use FakeClock.
type Clock interface {
	Now() time.Time
}

// RealClock is the production implementation. Use in app.go DI.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now().UTC() }

// FakeClock is for tests. Time advances only when Advance() is called.
type FakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func NewFakeClock(t time.Time) *FakeClock { return &FakeClock{now: t} }

func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
