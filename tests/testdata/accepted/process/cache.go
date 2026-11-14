package process

import (
	"context"
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

// Cache refreshes its entry each second.
type Cache struct {
	proc.Process
	version int
	entry   im.Owner[Entry]
}

// Run refreshes the entry each second until ctx ends.
func (c *Cache) Run(ctx context.Context) {
	for {
		c.version++
		c.entry.Store(&Entry{Key: "config", Version: c.version})
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

// Latest returns a copy of the current entry, or the zero Entry before the first refresh.
func (c *Cache) Latest() Entry {
	entry := c.entry.Load()
	if entry == nil {
		return Entry{}
	}
	return *entry
}
