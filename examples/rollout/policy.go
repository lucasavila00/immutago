package rollout

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Limits compares candidate errors and latency with the primary.
type Limits struct {
	im.Copied
	MinRequests             uint64  `json:"min_requests"`
	MaxErrorIncreasePercent float64 `json:"max_error_increase_percent"`
	MaxP95IncreaseMillis    int     `json:"max_p95_increase_ms"`
}

// PolicySnapshot holds one valid policy and the most recent file error.
type PolicySnapshot struct {
	im.Frozen
	Version string
	Limits  Limits
	Problem string
}

// PolicyArgs supplies the policy file directory and its refresh interval.
type PolicyArgs struct {
	Files fs.FS
	Every time.Duration
	Log   *log.Logger
}

// Policy is a simple root service that publishes gate configuration in the background.
type Policy struct {
	supervise.Service[PolicyArgs]
	current PolicySnapshotCell
}

// Init requires a valid first policy before its reference becomes available.
func (p *Policy) Init() error {
	if p.Args().Every <= 0 {
		return fmt.Errorf("policy read interval must be positive")
	}
	return p.reload()
}

// Run keeps the last valid policy on file errors and tries the next edit.
func (p *Policy) Run(ctx context.Context) {
	args := p.Args()
	args.Log.Print("policy started")
	defer args.Log.Print("policy stopped")
	ticker := time.NewTicker(args.Every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.reload(); err != nil {
				args.Log.Printf("policy: %v", err)
			}
		}
	}
}

// Read selects one complete immutable policy version.
func (p *Policy) Read() PolicySnapshotProjection { return p.current.Load() }

// reload validates a complete file before it changes the limits.
func (p *Policy) reload() error {
	data, err := fs.ReadFile(p.Args().Files, "policy.json")
	var document struct {
		Version string `json:"version"`
		Limits
	}
	if err == nil {
		err = json.Unmarshal(data, &document)
	}
	if err == nil && (document.Version == "" || document.MinRequests == 0 ||
		document.MaxErrorIncreasePercent < 0 || document.MaxP95IncreaseMillis < 0) {
		err = fmt.Errorf("policy needs a version, positive min_requests, and nonnegative error and latency limits")
	}
	if err != nil {
		var previous PolicySnapshot
		selected := p.current.Load()
		if !selected.IsNil() {
			previous = *selected.Unsafe()
		}
		p.current.Store(PolicySnapshot{Version: previous.Version, Limits: previous.Limits, Problem: err.Error()})
		return err
	}
	p.current.Store(PolicySnapshot{Version: document.Version, Limits: document.Limits})
	return nil
}
