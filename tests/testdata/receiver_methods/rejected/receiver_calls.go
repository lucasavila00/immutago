package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Product struct {
	im.Snapshot
	Price int
}
type Order struct {
	im.Snapshot
	Product
}
type Counter int

func (c *Counter) Inc() { *c++ }

type Counted struct {
	im.Snapshot
	Counter
}

func (p Product) Cost() int            { return p.Price }
func (p *Product) ReadCost() int       { return p.Price }
func (p Product) Get() (Product, bool) { return p, true }

func calls() {
	value := Product{}
	pointer := &Product{}
	order := Order{}
	counted := &Counted{}
	_ = value.ReadCost()
	_ = pointer.Cost()
	_ = order.Cost()
	_ = order.Product.ReadCost()
	counted.Inc()
	counted.Counter.Inc()
	bound := pointer.ReadCost
	_ = bound
	_ = Product.Cost(value)
	_ = (*Product).ReadCost(pointer)
	var reader interface{ ReadCost() int } = pointer
	_ = reader.ReadCost()
	go pointer.ReadCost()
	defer pointer.ReadCost()
	_ = func() int { return pointer.ReadCost() }
	got, found := value.Get()
	_ = got
	_ = found
}
