package process

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// Store reads values by key.
type Store interface {
	Get(key string) (string, bool)
}

// FetcherArgs are the arguments of a Fetcher. Each dependency is safe for concurrent use.
type FetcherArgs struct {
	im.Shared
	BaseURL  string
	Log      proc.UncheckedDependency[*slog.Logger]
	Client   proc.UncheckedDependency[*http.Client]
	NewStore proc.UncheckedDependency[func(name string) Store]
}

// Fetcher uses a logger, an HTTP client, and a store from its factory.
type Fetcher struct {
	proc.Process[FetcherArgs]
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
func StartFetcher(root *proc.Supervisor, log *slog.Logger) proc.PID[Fetcher] {
	return proc.Start[Fetcher](root, FetcherArgs{
		BaseURL:  "https://example.com",
		Log:      proc.Trust(log),
		Client:   proc.Trust(http.DefaultClient),
		NewStore: proc.Trust(func(string) Store { return fakeStore{} }),
	})
}

// Tracer is a process whose argument is one dependency.
type Tracer struct{ proc.Process[proc.UncheckedDependency[*slog.Logger]] }

// Run logs once.
func (t *Tracer) Run(ctx context.Context) { t.Args().Value().Info("trace") }

// StartTracer starts a Tracer.
func StartTracer(root *proc.Supervisor, log *slog.Logger) proc.PID[Tracer] {
	return proc.Start[Tracer](root, proc.Trust(log))
}
