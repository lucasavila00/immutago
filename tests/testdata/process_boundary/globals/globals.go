package globals

import (
	"context"

	"example.com/consumer/plain"
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
