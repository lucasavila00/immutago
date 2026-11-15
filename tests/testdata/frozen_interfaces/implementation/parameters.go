package implementation

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

var SavedInput any
var SavedCallback func() int

// Apply can retain ordinary inputs under their Go contracts.
func (t *Tree) Apply(value any, callback func() int) int {
	SavedInput = value
	SavedCallback = callback
	result := int(t.Weight)
	if value != nil {
		result += value.(int)
	}
	if callback != nil {
		result += callback()
	}
	return result
}

func (t *Tree) Named(value api.NumberReader, callback api.Callback) int {
	if value == nil {
		return t.Apply(nil, callback)
	}
	return t.Apply(value.Number(), callback)
}

func (t *Tree) Aliased(value api.ReaderAlias, callback api.CallbackAlias) int {
	return t.Named(value, callback)
}

func (t *Tree) ReadLocal(value im.StringProjection) int { return int(t.Weight) + value.Len() }
func (t *Tree) WithScorer(value api.Scorer) int         { return int(t.Weight + value.Score(nil)) }
func (c Count) Number() int                             { return c.N }
