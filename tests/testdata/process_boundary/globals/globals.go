package globals

import (
	"context"
	"errors"
	"io"

	"example.com/consumer/plain"
	"example.com/consumer/relay"
	"example.com/consumer/state"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// Tally uses package variables of other packages.
type Tally struct{ proc.Process[proc.None] }

// Run publishes a total through an Owner slot of another package.
func (*Tally) Run(ctx context.Context) {
	state.Current.Store(&state.Totals{Count: 1})
	<-ctx.Done()
}

// Hits reads a map of a checked package.
func (*Tally) Hits() int { return state.Hits["a"] }

// Limit reads an exported scalar of a checked package.
func (*Tally) Limit() int { return state.Limit }

// Name reads a map of a package that the checker does not check.
func (*Tally) Name() string { return plain.Names["a"] }

// Version reads a string of a package that the checker does not check.
func (*Tally) Version() string { return plain.Version }

// Record calls a function that changes a map of a checked package.
func (*Tally) Record() { state.Record("a") }

// Relay reaches the same map through a function of a third package.
func (*Tally) Relay() { relay.Relay("a") }

// Track calls a method of a package that this package does not import.
func (*Tally) Track() { relay.NewTracker().Add("a") }

// Publish calls a function that uses only an Owner slot.
func (*Tally) Publish() { state.Publish(2) }

// Remember calls a function of a package that the checker does not check.
func (*Tally) Remember() { plain.Remember("a") }

// Forward reaches a checked map through a package that the checker does not check.
func (*Tally) Forward() { plain.Forward("a") }

// Visit calls a function that uses only a scalar atomic.
func (*Tally) Visit() int64 { return state.Visit() }

// full compares with an error that its package does not change.
func (*Tally) full(err error) bool { return errors.Is(err, state.ErrFull) || err == io.EOF }

// gone compares with an error that its package changes.
func (*Tally) gone(err error) bool { return err == state.ErrGone }

// clearFull changes an error variable of another package.
func clearFull() { state.ErrFull = nil }
