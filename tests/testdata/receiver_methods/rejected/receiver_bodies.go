package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Settings struct {
	im.Local
	Limit int
}

var saved *Settings

func (p *Product) Change()            { p.Price = 1 }
func (p Product) ChangeCopy()         { p.Price = 1 }
func (s *Settings) Save()             { saved = s }
func (s *Settings) Return() *Settings { return s }
func (s *Settings) Nil() *Settings    { return nil }
func (s *Settings) Capture()          { _ = func() int { return s.Limit } }

type Ordinary struct{}

func (o Ordinary) Read(p *Product) int { return p.Price }

type Generic[T any] struct {
	im.Frozen
	Value T
}

func (g Generic[T]) Read() T { return g.Value }
