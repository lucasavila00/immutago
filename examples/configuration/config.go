package configuration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Config is one immutable version of the request limits.
type Config struct {
	im.Scoped
	Version int
	Limits  im.FrozenMap[im.FrozenString, int]
}

// LoaderArgs supplies the text cache and the interval between checks.
type LoaderArgs struct {
	Cache supervise.Ref[FileCache]
	Every time.Duration
}

// ConfigLoader publishes valid JSON limits and keeps the last valid version on an error.
type ConfigLoader struct {
	supervise.Service[LoaderArgs]
	current im.Owner[Config]
	last    string
	mu      sync.RWMutex
	problem string
}

// Init parses the initial contents before callers can read limits.
func (l *ConfigLoader) Init() error {
	if l.Args().Every <= 0 {
		return fmt.Errorf("configuration interval must be positive")
	}
	return l.load()
}

// Run checks for changed text and records parse errors without discarding valid limits.
func (l *ConfigLoader) Run(ctx context.Context) {
	ticker := time.NewTicker(l.Args().Every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := l.load()
			l.mu.Lock()
			l.problem = ""
			if err != nil {
				l.problem = err.Error()
			}
			l.mu.Unlock()
		}
	}
}

// load copies decoded limits into a fresh builder and publishes one immutable version.
func (l *ConfigLoader) load() error {
	text := l.Args().Cache.Current().Text().Clone()
	version := l.Version()
	if version > 0 && text == l.last {
		return nil
	}
	var decoded map[string]int
	if err := json.Unmarshal([]byte(text), &decoded); err != nil {
		return err
	}
	if decoded == nil {
		return fmt.Errorf("configuration must be a JSON object")
	}
	limits := make(map[im.FrozenString]int, len(decoded))
	for name, limit := range decoded {
		if limit < 0 {
			return fmt.Errorf("limit for %s must not be negative", name)
		}
		limits[im.FreezeString(name)] = limit
	}
	frozen := im.FreezeMap(limits)
	l.current.Store(&Config{Version: version + 1, Limits: frozen})
	l.last = text
	return nil
}

// Version returns the last valid configuration version.
func (l *ConfigLoader) Version() int {
	config := l.current.Load()
	if config == nil {
		return 0
	}
	return config.Version
}

// Limit returns the current limit for a key; an absent key has a zero limit.
func (l *ConfigLoader) Limit(key string) int {
	config := l.current.Load()
	if config == nil {
		return 0
	}
	limit, _ := config.Limits.Get(im.FreezeString(key))
	return limit
}

// Problem returns the latest parse error, or an empty string after a valid read.
func (l *ConfigLoader) Problem() im.FrozenString {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return im.FreezeString(l.problem)
}
