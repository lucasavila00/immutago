//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package invalid

import (
	"example.com/consumer/valid"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Snapshot is Frozen, not a read result.
type Snapshot struct {
	im.Frozen
	N int
}

// Scorer is Frozen, not a read result.
type Scorer interface {
	im.FrozenInterface
	Score() int
}

// Copied cannot contain trusted reference handles.
type Copied struct {
	im.Copied
	Model valid.Handle
}

// Named arrays do not become read results.
type Named [2]valid.Handle

// Loader supplies rejected result signatures.
type Loader struct {
	supervise.Service[supervise.NoArgs]
}

// Named returns a named array.
func (*Loader) Named() Named { return Named{} }

// Pointer returns a pointer instead of the handle value.
func (*Loader) Pointer() *valid.Handle { return nil }

// Native returns an ordinary native slice.
func (*Loader) Native() []valid.Handle { return nil }

// saved cannot store a Local record persistently.
var saved valid.Serving

// Capture cannot capture the Local owner when only its handle is used.
func Capture(value valid.Serving) func() valid.Handle {
	return func() valid.Handle { return value.Model }
}

// CaptureProjection cannot capture a Local projection for a later handle read.
func CaptureProjection(value valid.ModelProjection) func() valid.Handle {
	return func() valid.Handle { return value.Table() }
}
