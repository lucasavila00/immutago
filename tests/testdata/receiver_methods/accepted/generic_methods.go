package accepted

import "github.com/lucasavila00/immutago/pkg/im"

func (p Product) Map[T any](read func(int) T) T         { return read(p.Price) }
func (p *Product) MapPointer[T any](read func(int) T) T { return read(p.Price) }
func mapProduct[T any](p Product, read func(int) T) T   { return read(p.Price) }
func number(n int) int                                  { return n }

// GenericReads uses method type parameters with each supported receiver form.
func GenericReads() int {
	value := Alias{Price: 3}
	pointer := &Product{Price: 3}
	result := value.Map(number)
	result += ((value).Map[int])(number)
	result += pointer.MapPointer(number)
	result += pointer.MapPointer[int](number)
	bound := value.Map[int]
	expression := Product.Map[int]
	result += bound(number)
	result += expression(value, number)
	result += mapProduct(value, number)
	order := Order{Product: value}
	result += order.Product.Map[int](number)
	values := []Product{value}
	result += values[0].Map[int](number)
	_ = im.FreezeSlice(values)
	return result
}
