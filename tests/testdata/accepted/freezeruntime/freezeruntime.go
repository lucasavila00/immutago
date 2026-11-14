package main

import (
	"example.com/consumer/freezebuilders"
	"fmt"
	"github.com/lucasavila00/immutago/pkg/im"
)

func main() {
	slices := freezebuilders.Slice()
	first := slices.First
	second := slices.Second
	fmt.Println(first.At(0), second.At(0), first.Len(), second.Len())
	maps := freezebuilders.Map()
	a := maps.First
	b := maps.Second
	old, _ := a.Get("a")
	next, _ := b.Get("a")
	fmt.Println(old, next, a.Len(), b.Len())
	fmt.Println(freezebuilders.MarkedSlice().At(0).X)
	item, _ := freezebuilders.MarkedMap().Get("a")
	fmt.Println(item.X)
	fmt.Println(freezebuilders.Branch(true).At(0), freezebuilders.Loop(3))
	fmt.Println(freezebuilders.Nil().Clone() == nil)
	fmt.Println(freezebuilders.AppendMarked().At(0).X, freezebuilders.CopyMarked().At(0).X)
	marked := freezebuilders.ClearMarkedMap()
	entry, ok := marked.Get(freezebuilders.Value{X: 3})
	fmt.Println(entry.X, ok, marked.Len())
	shared := &freezebuilders.Value{X: 5}
	alias := shared
	shared = &freezebuilders.Value{X: 6}
	native := []*freezebuilders.Value{alias}
	frozen := im.FreezeSlice(native)
	clone := frozen.Clone()
	clone[0] = shared
	updated := im.FreezeSlice(clone)
	fmt.Println(alias.X, shared.X, frozen.At(0).X, updated.At(0).X)
}
