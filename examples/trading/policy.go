package trading

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

// PolicySnapshot publishes complete risk limits and the most recent file error.
type PolicySnapshot struct {
	im.Frozen
	Version uint64
	Limits  Limits
	Problem string
}

// PolicyArgs supplies the risk file and its refresh interval.
type PolicyArgs struct {
	Files fs.FS
	Every time.Duration
	Log   *log.Logger
}

// Policy refreshes risk limits independently from all market workers.
type Policy struct {
	supervise.Service[PolicyArgs]
	current PolicySnapshotCell
}

// Init requires a valid first policy before any strategy can start.
func (p *Policy) Init() error {
	if p.Args().Every <= 0 {
		return fmt.Errorf("policy read interval must be positive")
	}
	return p.reload()
}

// Run keeps the last valid limits when a file edit fails.
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

// Read selects the single policy version used for one price-to-quote calculation.
func (p *Policy) Read() PolicySnapshotProjection { return p.current.Load() }

// reload validates file values before it publishes a new immutable policy.
func (p *Policy) reload() error {
	data, err := fs.ReadFile(p.Args().Files, "policy.json")
	var document struct {
		Version uint64 `json:"version"`
		Limits
	}
	if err == nil {
		err = json.Unmarshal(data, &document)
	}
	if err == nil && (document.Version == 0 || document.MaxInventory < 1 || document.MaxInventory > 1_000_000 ||
		document.MaxSpread < 1 || document.MaxSpread > 1_000_000) {
		err = fmt.Errorf("policy needs a positive version and limits in 1..1000000")
	}
	if err != nil {
		var previous PolicySnapshot
		selected := p.current.Load()
		if !selected.IsNil() {
			previous = *selected.Unchecked()
		}
		p.current.Store(PolicySnapshot{Version: previous.Version, Limits: previous.Limits, Problem: err.Error()})
		return err
	}
	p.current.Store(PolicySnapshot{Version: document.Version, Limits: document.Limits})
	return nil
}
