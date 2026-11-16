package accepted

import (
	"iter"

	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

var savedPoints iter.Seq2[int, *api.Count]
var title im.FrozenString

func sumPoints(items iter.Seq2[int, *api.Count]) int {
	total := 0
	for _, point := range items {
		total += point.N
	}
	return total
}

func pointerIterator(points im.FrozenSlice[*api.Count]) int {
	items := points.All()
	savedPoints = items
	return sumPoints(items)
}

func frozenReceivers(boxed any, texts [1]im.FrozenString, build func() im.FrozenString) int {
	return title.Len() + boxed.(im.FrozenString).Len() + texts[0].Len() + build().Len()
}
