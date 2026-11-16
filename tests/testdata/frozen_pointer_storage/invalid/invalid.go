package invalid

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type ValueOwner struct{ Value api.Settings }
type Local struct {
	im.Local
	N int
}
type LocalOwner struct{ Value *Local }
type ProjectionOwner struct{ Value im.Projection[*api.Settings] }

var scalar im.Cell[int]
var copied im.Cell[[2]int]

func BadLoads() {
	_ = scalar.LoadPointer().Unchecked()
	_ = copied.LoadPointer().Unchecked()
}

func Writes(state *api.State, values []*api.Settings, versions map[string]*api.Settings) {
	state.Settings.Limit = 1
	*state.Settings = api.Settings{}
	state.Settings.Array[0]++
	values[0].Limit++
	versions["x"].Limit++
	_ = &state.Settings.Limit
	state.Settings.Count.Inc()
}

func Promoted(outer *api.Outer) {
	outer.Limit++
	outer.Array[0] = 1
	_ = &outer.Limit
	outer.Count.Inc()
}

func BoxedTarget(state *api.State) {
	var value any = state.Settings
	value.(*api.Settings).Limit++
	p := identity(state.Settings)
	p.Limit++
}

func identity[T any](value T) T { return value }

func FreezeAlias(p *api.Settings) {
	values := []*api.Settings{p}
	alias := values
	_ = im.FreezeSlice(values)
	_ = alias
}

func GenericTarget[T ~*api.Settings](p T) { (*p).Limit++ }

func GenericReplacement[T ~*api.Settings](p T) { *p = api.Settings{} }

func SlotTarget(state *api.State) {
	slot := &state.Settings
	(**slot).Limit++
}

type Selected *api.Settings

func NamedTarget(p Selected) { p.Limit++ }

type Worker struct {
	supervise.Service[supervise.NoArgs]
}

func (w *Worker) Read() *api.Settings { return api.Current() }
