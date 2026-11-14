package process

import (
	"context"
	"sync"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// Decision is the result of one rate limit check.
type Decision int

// Decisions of the rate limiter.
const (
	Allowed Decision = iota
	Limited
)

// Usage is a copy of the limiter counters.
type Usage struct {
	im.Scoped
	Used  int
	Limit int
}

// Limiter counts requests against a fixed limit.
type Limiter struct {
	proc.Process
	mu   sync.Mutex
	used int
}

// Run waits until ctx ends.
func (l *Limiter) Run(ctx context.Context) { <-ctx.Done() }

// Take records one request and returns whether it is allowed.
func (l *Limiter) Take(limit int) Decision {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.used >= limit {
		return Limited
	}
	l.used++
	return Allowed
}

// Usage returns a copy of the counters.
func (l *Limiter) Usage(limit int) Usage {
	l.mu.Lock()
	defer l.mu.Unlock()
	return Usage{Used: l.used, Limit: limit}
}
