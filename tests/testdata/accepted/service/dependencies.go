package service

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Store reads values by key.
type Store interface {
	Get(key string) (string, bool)
}

// FetcherArgs are the arguments of a Fetcher. Each dependency is safe for concurrent use.
type FetcherArgs struct {
	im.Shared
	BaseURL  string
	Log      supervise.ConcurrentSafe[*slog.Logger]
	Client   supervise.ConcurrentSafe[*http.Client]
	NewStore supervise.ConcurrentSafe[func(name string) Store]
}

// Fetcher uses a logger, an HTTP client, and a store from its factory.
type Fetcher struct {
	supervise.Service[FetcherArgs]
	store Store
}

// Run builds the store and logs the configured URL.
func (f *Fetcher) Run(ctx context.Context) {
	log := f.Args().Log.Value()
	f.store = f.Args().NewStore.Value()("pages")
	log.Info("start", "url", f.Args().BaseURL, "timeout", f.Args().Client.Value().Timeout)
	<-ctx.Done()
}

// fakeStore is a test fake for Store.
type fakeStore struct{}

// Get returns no value.
func (fakeStore) Get(string) (string, bool) { return "", false }

// StartFetcher starts a Fetcher with trusted dependencies.
func StartFetcher(root *supervise.Group, log *slog.Logger) (supervise.Ref[Fetcher], error) {
	return supervise.Start[Fetcher](root, FetcherArgs{
		BaseURL:  "https://example.com",
		Log:      supervise.AssumeConcurrentSafe(log),
		Client:   supervise.AssumeConcurrentSafe(http.DefaultClient),
		NewStore: supervise.AssumeConcurrentSafe(func(string) Store { return fakeStore{} }),
	})
}

// Tracer is a service whose argument is one dependency.
type Tracer struct {
	supervise.Service[supervise.ConcurrentSafe[*slog.Logger]]
}

// Run logs once.
func (t *Tracer) Run(ctx context.Context) { t.Args().Value().Info("trace") }

// StartTracer starts a Tracer.
func StartTracer(root *supervise.Group, log *slog.Logger) (supervise.Ref[Tracer], error) {
	return supervise.Start[Tracer](root, supervise.AssumeConcurrentSafe(log))
}
