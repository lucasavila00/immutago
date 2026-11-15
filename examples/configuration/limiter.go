package configuration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Decision reports whether one request fits in the current window.
type Decision int

// Request decisions.
const (
	Allowed Decision = iota
	Limited
)

// LimiterArgs selects a configuration key and a fixed window duration.
type LimiterArgs struct {
	Loader supervise.Ref[ConfigLoader]
	Key    string
	Window time.Duration
}

// Usage is a copy of the current counters and limit.
type Usage struct {
	im.Snapshot
	Used   int
	Limit  int
	Window int
}

// Limiter counts requests and resets the count once per window.
type Limiter struct {
	supervise.Service[LimiterArgs]
	mu     sync.Mutex
	used   int
	window int
}

// Init rejects a window that cannot drive a timer.
func (l *Limiter) Init() error {
	if l.Args().Window <= 0 {
		return fmt.Errorf("window duration must be positive")
	}
	return nil
}

// Run resets the quota at each window boundary until shutdown.
func (l *Limiter) Run(ctx context.Context) {
	ticker := time.NewTicker(l.Args().Window)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.mu.Lock()
			l.used = 0
			l.window++
			l.mu.Unlock()
		}
	}
}

// Take counts an allowed request using the latest published limit.
func (l *Limiter) Take() Decision {
	limit := l.Args().Loader.Current().Limit(l.Args().Key)
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.used >= limit {
		return Limited
	}
	l.used++
	return Allowed
}

// Usage returns a value copy that callers can keep after the next window starts.
func (l *Limiter) Usage() Usage {
	limit := l.Args().Loader.Current().Limit(l.Args().Key)
	l.mu.Lock()
	defer l.mu.Unlock()
	return Usage{Used: l.used, Limit: limit, Window: l.window}
}
