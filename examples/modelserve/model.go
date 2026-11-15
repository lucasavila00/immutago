//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

// Package modelserve compares local house-price models under nested supervision.
package modelserve

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"sync"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Parameters holds one complete set of model coefficients.
type Parameters struct {
	im.Frozen
	Version        string
	Base           float64
	PerSquareMeter float64
	PerRoom        float64
}

// modelDocument is the editable model file.
type modelDocument struct {
	Version        string  `json:"version"`
	Base           float64 `json:"base"`
	PerSquareMeter float64 `json:"per_square_meter"`
	PerRoom        float64 `json:"per_room"`
}

// ModelArgs supplies one model file and its refresh interval.
type ModelArgs struct {
	Files fs.FS
	Path  string
	Every time.Duration
	Log   *log.Logger
}

// Model refreshes its own coefficients without replacing its parent or peers.
type Model struct {
	supervise.Service[ModelArgs]
	current ParametersCell
	mu      sync.RWMutex
	problem string
}

// Init requires a valid model before its reference becomes available.
func (s *Model) Init() error {
	if err := s.reload(); err != nil {
		return err
	}
	s.Args().Log.Printf("model ready: %s", s.Args().Path)
	return nil
}

// Run keeps the last valid model when a file edit is incomplete.
func (s *Model) Run(ctx context.Context) {
	args := s.Args()
	ticker := time.NewTicker(args.Every)
	defer ticker.Stop()
	defer args.Log.Printf("model stopped: %s", args.Path)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.reload()
		}
	}
}

// Read selects the coefficients used for one complete prediction.
func (s *Model) Read() ParametersProjection { return s.current.Load() }

// Problem reports the last ordinary file error.
func (s *Model) Problem() im.StringProjection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return im.ProjectString(s.problem)
}

// reload validates all coefficients before it publishes them together.
func (s *Model) reload() error {
	data, err := fs.ReadFile(s.Args().Files, s.Args().Path)
	var document modelDocument
	if err == nil {
		err = json.Unmarshal(data, &document)
	}
	if err == nil && document.Version == "" {
		err = fmt.Errorf("model version must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.problem = err.Error()
		return err
	}
	s.current.Store(Parameters{Version: document.Version, Base: document.Base,
		PerSquareMeter: document.PerSquareMeter, PerRoom: document.PerRoom})
	s.problem = ""
	return nil
}

// Features contains the measurements supplied by one request.
type Features struct {
	Area  float64 `json:"area_m2"`
	Rooms int     `json:"rooms"`
}

// Prediction identifies the selected version and its estimate in dollars.
type Prediction struct {
	Version  string  `json:"version"`
	Estimate float64 `json:"estimate"`
}

// Predict uses the selected coefficients even if their service has stopped.
func Predict(model ParametersProjection, input Features) Prediction {
	return Prediction{Version: model.Version().Unsafe(),
		Estimate: model.Base() + input.Area*model.PerSquareMeter() + float64(input.Rooms)*model.PerRoom()}
}
