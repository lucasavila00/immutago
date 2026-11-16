package rejected

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

type HandleState struct {
	Reader api.Reader
	Text   im.FrozenString
}
type StateAlias = HandleState
type BadAlias = api.Promoted
type RecordHolder struct{ Value api.Count }
type LocalHolder struct{ Values [0]im.StringProjection }

func unprovedReader(api.Reader)            {}
func unprovedCall()                        { unprovedReader(&BadAlias{}) }
func unprovedReturn() api.Reader           { return &BadAlias{} }
func unprovedPair() (*BadAlias, int)       { return &BadAlias{}, 1 }
func unprovedTuple() (api.Reader, int)     { return unprovedPair() }
func unprovedField(h *StateAlias)          { h.Reader = &BadAlias{} }
func unprovedLiteral()                     { _ = StateAlias{Reader: &BadAlias{}} }
func unprovedMap()                         { _ = map[string]api.Reader{"bad": &BadAlias{}} }
func unprovedAppend(values []api.Reader)   { values = append(values, &BadAlias{}) }
func unprovedSend(ready chan<- api.Reader) { ready <- &BadAlias{} }
func unprovedGenericLiteral()              { _ = Box[api.Reader]{Value: &BadAlias{}} }

func frozenTarget(h im.FrozenSlice[*api.Count]) { h.At(0).N++ }
func frozenOwner(root *api.Data) {
	root.Text = im.FreezeString("changed")
	_ = &root.Text
}
func frozenBuilderAlias() {
	values := []im.FrozenString{im.FreezeString("x")}
	alias := values
	_ = alias
	frozen := im.FreezeSlice(values)
	_ = frozen
}
func frozenBuilderConsumed() {
	values := []im.FrozenString{im.FreezeString("x")}
	frozen := im.FreezeSlice(values)
	_ = frozen
	values[0] = im.FreezeString("changed")
}

func localSend(ready chan<- int, value im.StringProjection) {
	ready <- value.Len()
	select {
	case ready <- value.Len():
	default:
	}
}

var recordChannel chan api.Count

func localRangeIndex(values []im.FrozenString, local im.StringProjection) {
	var indexes [1]int
	var value im.FrozenString
	for indexes[local.Len()], value = range values {
	}
	_ = value
}

func builderRangeIndex(values []im.FrozenString) {
	builder := []int{0}
	var indexes [1]int
	var value im.FrozenString
	for indexes[builder[0]], value = range values {
	}
	_ = value
	_ = im.FreezeSlice(builder)
}
