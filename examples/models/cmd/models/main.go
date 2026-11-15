// Command models prints predictions while an HTTP model changes.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/lucasavila00/immutago/examples/models"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// main reads the source and cache paths and stops the service on Ctrl-C.
func main() {
	url := flag.String("url", "http://127.0.0.1:8080/model.json", "HTTP model URL")
	cache := flag.String("cache", "model-cache.json", "file for the last valid model")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := run(ctx, *url, *cache); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run starts the loader and prints a prediction for two fixed input features.
func run(ctx context.Context, url, cache string) error {
	root := supervise.NewRoot(nil)
	defer func() { <-root.Stop() }()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	client := &http.Client{Timeout: 2 * time.Second}
	factory := func(string) models.Store { return models.FileStore{Path: cache} }
	args := models.FetcherArgs{
		URL: url, Every: time.Second,
		Log:      logger,
		Client:   client,
		NewStore: factory,
	}
	fetcher, err := supervise.Start[models.Fetcher](root, args)
	if err != nil {
		return err
	}
	features := im.FreezeSlice([]float64{2, 3})
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		prediction := fetcher.Current().Predict(features)
		fmt.Printf("%s: score %.0f\n", prediction.Name.Unsafe(), prediction.Score)
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
