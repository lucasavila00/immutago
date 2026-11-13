package consumer

import (
	"example.com/consumer/model"
	"github.com/lucasavila00/immutago/pkg/im"
)

func Iterate() int {
	values := im.FreezeSlice([]model.Leaf{{Count: 2}, {Count: 3}})
	total := 0
	for index, value := range values.All() {
		total += index + int(value.Count)
	}
	mapping := im.FreezeMap(map[string]model.Leaf{"first": {Count: 4}})
	for key, value := range mapping.All() {
		total += len(key) + int(value.Count)
	}
	return total
}
