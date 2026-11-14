package state

import (
	"errors"
	"sync/atomic"

	"github.com/lucasavila00/immutago/pkg/im"
)

// Totals is one published total.
type Totals struct {
	im.Scoped
	Count int
}

// Current is a package Owner slot.
var Current im.Owner[Totals]

// Hits is a mutable map of a checked package.
var Hits = map[string]int{}

// Limit is an exported scalar of a checked package.
var Limit = 10

// Record changes Hits.
func Record(key string) { Hits[key]++ }

// Publish uses only the Owner slot.
func Publish(count int) { Current.Store(&Totals{Count: count}) }

var visits atomic.Int64

// Visit counts calls in a scalar atomic.
func Visit() int64 { return visits.Add(1) }

// ErrFull is an error that no code changes after package initialization.
var ErrFull = errors.New("full")

// ErrGone is an error that Forget changes.
var ErrGone = errors.New("gone")

// Forget changes ErrGone.
func Forget() { ErrGone = nil }
