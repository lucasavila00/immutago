package accepted

import "github.com/lucasavila00/immutago/pkg/im"

type Product struct {
	im.Shared
	Price int
}
type Alias = Product
type Settings struct {
	im.ByValue
	Limit int
}
type Order struct {
	im.Shared
	Product
	Pointer *Product
}

func (p Product) Cost() int                 { return p.Price }
func (p *Product) ReadCost() int            { return p.Price }
func (p *Product) Self() *Product           { return p }
func (p Product) Add(other Product) Product { return Product{Price: p.Price + other.Price} }
func (p Product) Pair() (Product, bool)     { return p, true }
func (s Settings) LimitValue() int          { return s.Limit }
func (s *Settings) Remaining(used int) int  { return s.Limit - used }
func (s *Settings) Copy() Settings          { return *s }
func (s *Settings) Rebind() int             { s = &Settings{Limit: 3}; return s.Limit }
func product() Product                      { return Product{Price: 8} }
func productPointer() *Product              { return &Product{Price: 8} }
func pair(p Product) (Product, bool)        { return p.Pair() }
func takePair(p Product, found bool) int {
	if found {
		return p.Price
	}
	return 0
}

func Reads() int {
	value := Alias{Price: 8}
	pointer := &Alias{Price: 8}
	settings := &Settings{Limit: 100}
	order := Order{Product: value, Pointer: pointer}
	values := [1]Product{value}
	pointers := []*Product{pointer}
	result := value.Cost()
	result += (pointer).ReadCost()
	result += (*pointer).Cost()
	result += order.Product.Cost()
	result += order.Pointer.ReadCost()
	result += values[0].Cost()
	result += pointers[0].ReadCost()
	result += Product{Price: 1}.Cost()
	result += (&Product{Price: 2}).ReadCost()
	result += product().Cost()
	result += productPointer().ReadCost()
	result += settings.Remaining(30)
	copy := settings.Copy()
	result += copy.LimitValue()
	result += settings.Rebind()
	added := value.Add(value)
	result += added.Cost()
	result += takePair(value.Pair())
	shared := pointer.Self()
	result += shared.ReadCost()
	frozen := im.FreezeSlice(pointers)
	result += frozen.At(0).ReadCost()
	return result
}
