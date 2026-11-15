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

// Raw cannot bypass the generated-only Read rule.
func Raw(value im.Projection[valid.Handle]) valid.Handle { return value.Read() }

// Clone still requires Copied elements.
func Clone(value im.SliceProjection[valid.Handle]) []valid.Handle { return value.Clone() }

// CloneMap still requires Copied values.
func CloneMap(value im.MapProjection[int, valid.Handle]) map[int]valid.Handle { return value.Clone() }

// FrozenRecord cannot be extracted by Value.
func FrozenRecord(value im.Projection[Snapshot]) Snapshot { return im.Value(value) }

// FrozenPointer cannot be extracted by Value.
func FrozenPointer(value im.Projection[*Snapshot]) *Snapshot { return im.Value(value) }

// FrozenInterface cannot be extracted by Value.
func FrozenInterface(value im.Projection[Scorer]) Scorer { return im.Value(value) }

// Local cannot be extracted by Value because its payload is not Frozen.
func Local(value im.Projection[valid.Serving]) valid.Serving { return im.Value(value) }

// String cannot be extracted by Value.
func String(value im.Projection[string]) string { return im.Value(value) }

// Collection cannot be extracted by Value.
func Collection(value im.Projection[im.FrozenSlice[int]]) im.FrozenSlice[int] { return im.Value(value) }

// Marker cannot be a projection payload or a Value result.
func Marker(value im.Projection[im.Copied]) im.Copied { return im.Value(value) }

// Write cannot replace a handle inside a Local owner.
func Write(value valid.Serving) { value.Model = valid.Handle{} }

// Address cannot expose the storage of a handle inside a Local owner.
func Address(value valid.Serving) *valid.Handle { return &value.Model }
