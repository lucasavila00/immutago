package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Immut
	X int
}

func read(pointer *Good) int { return pointer.X }

func bad(count int) {
	var value Good
	var previous *Good
	for index := 0; index < count; index++ {
		value.X = index
		pointer := &value
		if previous != nil {
			_ = read(previous)
		}
		previous = pointer
	}
}
