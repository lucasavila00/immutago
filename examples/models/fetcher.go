// Package models polls an HTTP source and serves predictions from its last valid model.
package models

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/lucasavila00/immutago/examples/models/model"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// FetcherArgs selects the model source and its concurrent dependencies.
type FetcherArgs struct {
	im.Shared
	URL      string
	Every    time.Duration
	Log      supervise.ConcurrentSafe[*slog.Logger]
	Client   supervise.ConcurrentSafe[*http.Client]
	NewStore supervise.ConcurrentSafe[func(name string) Store]
}

// snapshot is one published model version.
type snapshot struct {
	im.Scoped
	Model *model.Model
}

// Prediction identifies the model used to calculate one score.
type Prediction struct {
	im.Shared
	Name  string
	Score float64
}

// document is the JSON model format supplied by the HTTP source.
type document struct {
	Name    string    `json:"name"`
	Weights []float64 `json:"weights"`
}

// Fetcher loads before startup and keeps serving the last valid model on refresh errors.
type Fetcher struct {
	supervise.Service[FetcherArgs]
	store   Store
	current im.Owner[snapshot]
}

// Init creates the document store and loads the first model before callers can predict.
func (f *Fetcher) Init() error {
	if f.Args().Every <= 0 {
		return fmt.Errorf("refresh interval must be positive")
	}
	factory := f.Args().NewStore.Value()
	f.store = factory("model")
	return f.refresh(context.Background())
}

// Run polls until shutdown; a failed refresh leaves the published model unchanged.
func (f *Fetcher) Run(ctx context.Context) {
	every := f.Args().Every
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	log := f.Args().Log.Value()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := f.refresh(ctx); err != nil {
				log.Warn("model refresh failed", "error", err)
			}
		}
	}
}

// refresh saves and publishes a model only after the HTTP response and JSON are valid.
func (f *Fetcher) refresh(ctx context.Context) error {
	url := f.Args().URL
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	client := f.Args().Client.Value()
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("model source returned %s", response.Status)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	var next document
	if err := json.Unmarshal(data, &next); err != nil {
		return err
	}
	if next.Name == "" || len(next.Weights) == 0 {
		return fmt.Errorf("model needs a name and coefficients")
	}
	if err := f.store.Save(string(data)); err != nil {
		return err
	}
	m := model.Load(next.Name, next.Weights)
	f.current.Store(&snapshot{Model: m})
	log := f.Args().Log.Value()
	log.Info("model loaded", "name", next.Name)
	return nil
}

// Predict uses one model version for the whole prediction.
func (f *Fetcher) Predict(features im.FrozenSlice[float64]) Prediction {
	current := f.current.Load()
	return Prediction{Name: current.Model.Name, Score: model.Predict(current.Model, features)}
}
