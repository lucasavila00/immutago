package process

import (
	"context"
	"errors"
	"io"
	"sync/atomic"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// Rates is one published rate table.
type Rates struct {
	im.Scoped
	Base int
}

// rates is a package Owner slot.
var rates im.Owner[Rates]

// defaultBase is never changed after its declaration.
var defaultBase = 10

// steps is filled only by init.
var steps [4]int

func init() {
	for i := range steps {
		steps[i] = i * defaultBase
	}
}

// lookups counts Step calls.
var lookups atomic.Int64

const maxStep = 3

// clampStep keeps a step index in range.
func clampStep(i int) int { return min(i, maxStep) }

// publishRates uses only the Owner slot.
func publishRates(base int) { rates.Store(&Rates{Base: base}) }

// RateKeeper reads package variables that the process rules permit.
type RateKeeper struct {
	proc.Process[proc.None]
	base int
}

// Run publishes the first rate table and waits until ctx ends.
func (r *RateKeeper) Run(ctx context.Context) {
	r.base = defaultBase
	publishRates(r.base)
	<-ctx.Done()
}

// Step returns a step from the table that init fills.
func (r *RateKeeper) Step(i int) int {
	lookups.Add(1)
	return steps[clampStep(i)]
}

// Lookups returns the number of Step calls.
func (r *RateKeeper) Lookups() int64 { return lookups.Load() }

// Base returns the published base, or zero before the first table.
func (r *RateKeeper) Base() int {
	current := rates.Load()
	if current == nil {
		return 0
	}
	return current.Base
}

// done reports whether err marks the end of input.
func (r *RateKeeper) done(err error) bool {
	switch err {
	case io.EOF, io.ErrUnexpectedEOF:
		return true
	}
	return errors.Is(err, io.ErrClosedPipe)
}
