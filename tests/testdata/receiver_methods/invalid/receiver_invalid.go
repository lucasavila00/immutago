package invalid

import "github.com/lucasavila00/immutago/pkg/im"

type Product struct {
	im.Frozen
	Price int
}

func (p *Product) Change() { p.Price = 1 }
