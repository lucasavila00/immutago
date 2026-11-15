//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package api

import "github.com/lucasavila00/immutago/pkg/im"

type Product struct {
	im.Frozen
	Price int
}
type Settings struct {
	im.Local
	Limit int
}

func (p Product) Cost() int                { return p.Price }
func (p *Product) ReadCost() int           { return p.Price }
func (s *Settings) Remaining(used int) int { return s.Limit - used }
