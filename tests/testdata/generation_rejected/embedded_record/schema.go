package embedded_record

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import "github.com/lucasavila00/immutago/pkg/im"

type Product struct {
	im.Frozen
	Price int
}

func (p Product) Cost() int { return p.Price }

type Order struct {
	im.Frozen
	Product
}

func read(order Order) int { return order.Cost() }
