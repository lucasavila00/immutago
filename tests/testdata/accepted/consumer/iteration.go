package consumer

import (
	"example.com/consumer/model"
	im "github.com/lucasavila00/immutago/pkg/immutable"
)

func Iterate() int {
	values := im.NewSlice(model.Leaf{Count: 2}, model.Leaf{Count: 3})
	total := 0
	for index, value := range values.All() {
		total += index + int(value.Count)
	}
	mapping := im.NewMap(map[string]model.Leaf{"first": {Count: 4}})
	for key, value := range mapping.All() {
		total += len(key) + int(value.Count)
	}
	return total
}
