package process

import (
	"context"
	"sync"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// Config is one immutable configuration version.
type Config struct {
	im.Scoped
	Version int
	Limits  im.FrozenMap[string, int]
}

// LoaderArgs are the arguments of a ConfigLoader.
type LoaderArgs struct {
	im.Shared
	Path  string
	Every time.Duration
}

// ConfigLoader polls for configuration and publishes each version.
type ConfigLoader struct {
	proc.Process[LoaderArgs]
	mu      sync.Mutex
	version int
	current im.Owner[Config]
}

// Run publishes a new configuration at each interval until ctx ends.
func (l *ConfigLoader) Run(ctx context.Context) {
	ticker := time.NewTicker(l.Args().Every)
	defer ticker.Stop()
	for {
		l.publish()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// publish builds and stores the next configuration.
func (l *ConfigLoader) publish() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.version++
	limits := map[string]int{l.Args().Path: 100 * l.version}
	frozen := im.FreezeMap(limits)
	l.current.Store(&Config{Version: l.version, Limits: frozen})
}

// Version returns the last published version.
func (l *ConfigLoader) Version() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.version
}

// Limit returns one limit of the current configuration.
func (l *ConfigLoader) Limit(name string) int {
	config := l.current.Load()
	if config == nil {
		return 0
	}
	limit, _ := config.Limits.Get(name)
	return limit
}
