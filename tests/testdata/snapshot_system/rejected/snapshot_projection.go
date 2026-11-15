package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Data struct {
	im.Snapshot
	Name  string
	Names [2]string
	Next  *Data
}
type Inner struct {
	im.Snapshot
	N int
}
type Outer struct {
	im.Snapshot
	Inner
}
type Generic[T any] = [1]T

var index int
var leaked string
var integerCell im.Cell[int]
var stringCell im.Cell[string]

func projection(view im.View[Data], outer im.View[Outer], array im.View[[1]Inner], p *Inner) {
	_ = im.Select(view, func(d Data) Data { return d })
	_ = im.Select(view, func(d Data) string { return d.Name[:] })
	_ = im.Select(view, func(d Data) string { return identity(d.Name) })
	_ = im.Select(view, func(d Data) string { leaked = d.Name; return d.Name })
	_ = im.Select(view, func(d Data) string { return d.Names[index] })
	i := 0
	_ = im.Select(view, func(d Data) string { return d.Names[i+1] })
	_ = im.Select(view, func(d Data) (name string) { return d.Name })
	_ = im.Select(view, named)
	_ = im.Select(outer, func(o Outer) int { return o.N })
	_ = im.Select(array, func(a Generic[Inner]) int { return a[0].N })
	_ = im.Select(im.ViewOf([1]int{}), func(a [len([1]im.View[int]{})]int) int { return a[0] })
	_ = im.Select(im.ViewOf([1]int{}), func(a [len([1]*Inner{p})]int) int { return a[0] })
}

func identity(value string) string { return value }
func named(value Data) string      { return value.Name }

func cellStatements() {
	go integerCell.Shared()
	defer stringCell.Shared()
	go identity(stringCell.Shared())
	defer identity(stringCell.Shared())
	go integerCell.Load()
	defer stringCell.Load()
	go stringCell.Store("text")
	defer stringCell.Store("text")
}
