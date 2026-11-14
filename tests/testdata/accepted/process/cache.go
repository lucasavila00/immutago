package process

import (
	"context"
	"sync"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// Entry is one immutable cache value.
type Entry struct {
	im.Scoped
	Key     string
	Version int
}

// Cache refreshes its entry when the configuration changes.
type Cache struct {
	proc.Process
	mu       sync.Mutex
	config   proc.PID[ConfigLoader]
	watching bool
	seen     int
	entry    im.Owner[Entry]
}

// Run refreshes the entry each second until ctx ends.
func (c *Cache) Run(ctx context.Context) {
	for {
		c.refresh()
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

// refresh replaces the entry when the configuration version changed.
func (c *Cache) refresh() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.watching {
		return
	}
	if version := c.config.Proc().Version(); version != c.seen {
		c.seen = version
		c.entry.Store(&Entry{Key: "config", Version: version})
	}
}

// SetConfig sets the configuration loader whose version the cache follows.
func (c *Cache) SetConfig(config proc.PID[ConfigLoader]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config = config
	c.watching = true
}

// Latest returns a copy of the current entry, or the zero Entry before the first refresh.
func (c *Cache) Latest() Entry {
	entry := c.entry.Load()
	if entry == nil {
		return Entry{}
	}
	return *entry
}
