package process

import (
	"context"
	"net/http"
	"os"
	"sync"

	"github.com/lucasavila00/immutago/pkg/proc"
)

// seen is a mutable map that two processes share.
var seen = map[string]int{}

// HitCounter writes the shared map.
type HitCounter struct{ proc.Process[proc.None] }

// Run writes the shared map until ctx ends.
func (*HitCounter) Run(ctx context.Context) {
	for ctx.Err() == nil {
		seen["hits"]++
	}
}

// HitReader reads the shared map.
type HitReader struct{ proc.Process[proc.None] }

// Run reads the shared map until ctx ends.
func (*HitReader) Run(ctx context.Context) {
	for ctx.Err() == nil {
		_ = seen["hits"]
	}
}

// Point is an ordinary mutable struct.
type Point struct{ X int }

var (
	queue    []int
	origin   = &Point{}
	center   Point
	lock     sync.Mutex
	interval = 5
	retries  = 3
	Timeout  = 30
)

// setRetries changes retries after package initialization.
func setRetries(n int) { retries = n }

// GlobalUser uses package variables that the process rules reject.
type GlobalUser struct{ proc.Process[proc.None] }

// Run waits until ctx ends.
func (*GlobalUser) Run(ctx context.Context) { <-ctx.Done() }

// Queued reads a package slice.
func (*GlobalUser) Queued() int { return len(queue) }

// Origin reads through a package pointer.
func (*GlobalUser) Origin() int { return origin.X }

// Center reads a package struct.
func (*GlobalUser) Center() int { return center.X }

// Lock uses a package mutex.
func (*GlobalUser) Lock() { lock.Lock(); lock.Unlock() }

// Interval takes the address of a package scalar.
func (*GlobalUser) Interval() int { p := &interval; return *p }

// Retries reads a scalar that setRetries changes.
func (*GlobalUser) Retries() int { return retries }

// Limit reads an exported scalar.
func (*GlobalUser) Limit() int { return Timeout }

// Print uses a standard library file variable.
func (*GlobalUser) Print() { _, _ = os.Stdout.WriteString("x") }

// Client uses the standard library default client.
func (*GlobalUser) Client() bool { return http.DefaultClient.Timeout == 0 }
