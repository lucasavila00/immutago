// Package api supplies checked snapshot and result types to another package.
package api

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Point struct {
	im.Frozen
	X, Y int
}

type Entry struct {
	im.Frozen
	Name  im.FrozenString
	Score int
}

type Model struct {
	im.Frozen
	Name    im.FrozenString
	Native  string
	Names   im.FrozenSlice[im.FrozenString]
	Entries im.FrozenMap[string, Entry]
	ByName  im.FrozenMap[im.FrozenString, int]
	Child   *Entry
	Points  [2]Point
}

type Result struct {
	im.Local
	Name  im.StringView
	Score int
	Pair  [2]im.View[string]
}

type ModelView struct {
	im.Local
	value im.View[Model]
}

func (v ModelView) Name() im.StringView {
	return im.String(im.Select(v.value, func(m Model) im.FrozenString { return m.Name }))
}

type ModelCell struct{ value im.Cell[Model] }

func (c *ModelCell) Store(value Model) { c.value.Store(value) }
func (c *ModelCell) Load() ModelView   { return ModelView{value: c.value.Load()} }
func (c *ModelCell) Shared() Model     { return c.value.Shared() }
func (c *ModelCell) Count() int        { return 2 }

type Holder struct{ Current ModelCell }

var current im.Cell[Model]

func Publish(value Model)  { current.Store(value) }
func Read() im.View[Model] { return current.Load() }
func Shared() Model        { return current.Shared() }

type Catalog struct {
	supervise.Service[supervise.NoArgs]
	state im.Cell[Model]
}

func (s *Catalog) Run(ctx context.Context) { <-ctx.Done() }
func (s *Catalog) Frozen() im.View[Model]  { return s.state.Load() }
func (s *Catalog) Count() int              { return 2 }
func (s *Catalog) Origin() Point           { return Point{X: 1} }
func (s *Catalog) Result() Result {
	view := s.state.Load()
	name := im.String(im.Select(view, func(m Model) im.FrozenString { return m.Name }))
	return Result{Name: name, Score: 2}
}
