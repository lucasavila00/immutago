// Package api supplies checked snapshot and result types to another package.
package api

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Point struct {
	im.Snapshot
	X, Y int
}

type Entry struct {
	im.Snapshot
	Name  im.FrozenString
	Score int
}

type Model struct {
	im.Snapshot
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

var current im.Cell[Model]

func Publish(value Model)  { current.Store(value) }
func Read() im.View[Model] { return current.Load() }
func Shared() Model        { return current.Shared() }

type Catalog struct {
	supervise.Service[supervise.NoArgs]
	state im.Cell[Model]
}

func (s *Catalog) Run(ctx context.Context)  { <-ctx.Done() }
func (s *Catalog) Snapshot() im.View[Model] { return s.state.Load() }
func (s *Catalog) Count() int               { return 2 }
func (s *Catalog) Origin() Point            { return Point{X: 1} }
func (s *Catalog) Result() Result {
	view := s.state.Load()
	name := im.String(im.Select(view, func(m Model) im.FrozenString { return m.Name }))
	return Result{Name: name, Score: 2}
}
