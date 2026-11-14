package process

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/lucasavila00/immutago/pkg/proc"
)

// FetcherArgs are ordinary dependencies and the PID of a peer.
type FetcherArgs struct {
	Logger *slog.Logger
	Client *http.Client
	URL    string
	Loader proc.PID[ConfigLoader]
}

// Fetcher fetches a URL each time it is woken.
type Fetcher struct {
	proc.Process[FetcherArgs]
	wake chan struct{}
}

// Init prepares the channel before callers can reach the process.
func (f *Fetcher) Init() { f.wake = make(chan struct{}, 1) }

// Run fetches the URL after each wake until ctx ends.
func (f *Fetcher) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-f.wake:
		}
		args := f.Args()
		response, err := args.Client.Get(args.URL)
		if err != nil {
			args.Logger.Warn("fetch failed", "error", err, "version", args.Loader.Proc().Version())
			continue
		}
		_ = response.Body.Close()
	}
}

// Wake asks for one more fetch.
func (f *Fetcher) Wake() {
	select {
	case f.wake <- struct{}{}:
	default:
	}
}
