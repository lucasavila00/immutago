package invalid

import "github.com/lucasavila00/immutago/pkg/im"

type Product struct {
	im.Snapshot
	Price int
}

func (p *Product) Change() { p.Price = 1 }
