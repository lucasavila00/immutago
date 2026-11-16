//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

// Package api supplies checked Frozen and result types to another package.
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

type PointResult struct {
	im.Local
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
	Name  im.StringProjection
	Score int
	Pair  [2]im.Projection[string]
}

func (c *ModelCell) Count() int { return 2 }

type Holder struct{ Current ModelCell }

var current im.Cell[Model]

func Publish(value Model)         { current.Store(value) }
func Read() im.Projection[*Model] { return current.LoadPointer() }
func Current() *Model             { return current.LoadPointer().Unchecked() }

type Catalog struct {
	supervise.Service[supervise.NoArgs]
	state im.Cell[Model]
}

func (s *Catalog) Run(ctx context.Context)       { <-ctx.Done() }
func (s *Catalog) Frozen() im.Projection[*Model] { return s.state.LoadPointer() }
func (s *Catalog) Count() int                    { return 2 }
func (s *Catalog) Origin() PointResult           { return PointResult{X: 1} }
func (s *Catalog) Result() Result {
	projection := s.state.LoadPointer()
	name := NewModelProjection(projection).Name()
	return Result{Name: name, Score: 2}
}
