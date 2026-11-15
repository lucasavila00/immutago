//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// FileArgs selects a file and its refresh interval.
type FileArgs struct {
	im.Frozen
	Path  string
	Every time.Duration
}

// FileCache serves the last file contents.
type FileCache struct {
	supervise.Service[FileArgs]
	mu   sync.RWMutex
	text string
}

// Init loads the first contents before calls begin.
func (c *FileCache) Init() error {
	if c.Args().Every <= 0 {
		return fmt.Errorf("refresh interval must be positive")
	}
	return c.read()
}

// Run refreshes the file until ctx ends.
func (c *FileCache) Run(ctx context.Context) {
	ticker := time.NewTicker(c.Args().Every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.read(); err != nil {
				panic(err)
			}
		}
	}
}

// read replaces the contents after a successful read.
func (c *FileCache) read() error {
	path := c.Args().Path
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.text = string(data)
	c.mu.Unlock()
	return nil
}

// Text returns the last contents.
func (c *FileCache) Text() im.StringProjection {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return im.ProjectString(c.text)
}

// main starts the cache, reads it, and stops its group.
func main() {
	root := supervise.NewRoot(nil)
	cache, err := supervise.Start[FileCache](root, FileArgs{
		Path: "settings.txt", Every: time.Second,
	})
	if err != nil {
		panic(err)
	}
	text := cache.Borrow().Text()
	fmt.Print(text.Clone())
	<-root.Stop()
}
