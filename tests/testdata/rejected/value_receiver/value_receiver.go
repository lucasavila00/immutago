package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Shared
	Value int
	Array [2]int
}

func Mutate(value Good) { value.Value = 1 }
