package health

import (
	"context"
	"sync"
	"time"
)

// Check is one dependency probe.
type Check struct {
	Name     string
	Critical bool // true → if fails, readyz returns 503
	Check    func(ctx context.Context) error
}

// Report is the aggregated state returned by /healthz handler.
type Report struct {
	Live    bool                  `json:"live"`
	Ready   bool                  `json:"ready"`
	Details map[string]CheckState `json:"details"`
}

type CheckState struct {
	OK  bool   `json:"ok"`
	Err string `json:"error,omitempty"`
	MS  int64  `json:"latency_ms"`
}

type Registry struct {
	mu     sync.RWMutex
	checks []Check
}

func NewRegistry() *Registry { return &Registry{} }

func (r *Registry) Register(c Check) {
	r.mu.Lock()
	r.checks = append(r.checks, c)
	r.mu.Unlock()
}

// Report runs every registered check sequentially (M1 simplicity).
// M2 may parallelize if cumulative latency exceeds 100ms.
func (r *Registry) Report(ctx context.Context) Report {
	r.mu.RLock()
	checks := append([]Check(nil), r.checks...)
	r.mu.RUnlock()

	rep := Report{Live: true, Ready: true, Details: make(map[string]CheckState)}
	for _, c := range checks {
		start := time.Now()
		err := c.Check(ctx)
		state := CheckState{OK: err == nil, MS: time.Since(start).Milliseconds()}
		if err != nil {
			state.Err = err.Error()
			if c.Critical {
				rep.Ready = false
			}
		}
		rep.Details[c.Name] = state
	}
	return rep
}
