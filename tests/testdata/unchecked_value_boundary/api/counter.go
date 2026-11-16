package api

import (
	"github.com/lucasavila00/immutago/pkg/im"
	"sync/atomic"
)

type counter struct{ value atomic.Int64 }

// Counter is a Frozen handle to mutable state owned by this package.
type Counter struct {
	im.Frozen
	state im.UncheckedValue[counter]
}

// NewCounter creates one shared counter.
func NewCounter() Counter { return Counter{state: im.Trust(&counter{})} }

// Add changes the target while the handle stays fixed.
func (c Counter) Add(n int64) int64 { return im.Unwrap(c.state).value.Add(n) }

// Pointer returns this immutable handle wrapper.
func (c *Counter) Pointer() *Counter { return c }
