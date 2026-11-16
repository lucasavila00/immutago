package main

import (
	"context"
	"io/fs"
	"testing/fstest"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Indexer struct{ supervise.Service[fs.FS] }
type Counter struct{ supervise.Service[int64] }

// Run waits for shutdown.
func (*Indexer) Run(ctx context.Context) { <-ctx.Done() }

// Run waits for shutdown.
func (*Counter) Run(ctx context.Context) { <-ctx.Done() }

// startArguments checks explicit A and conversions when argument types differ.
func startArguments(root *supervise.Group) {
	_, _ = supervise.Start[Indexer, fs.FS](root, fstest.MapFS{})
	_, _ = supervise.Start[Indexer](root, fs.FS(fstest.MapFS{}))
	_, _ = supervise.Start[Counter, int64](root, 5)
	_, _ = supervise.Start[Counter](root, int64(5))
}
