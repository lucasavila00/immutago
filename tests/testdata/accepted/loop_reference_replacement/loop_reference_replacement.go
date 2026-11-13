package accepted

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Shared
	X int
}

func read(pointer *Good) int { return pointer.X }

func replaceInLoop(count int) {
	var value Good
	var previous *Good
	for index := 0; index < count; index++ {
		value = Good{X: index}
		pointer := new(value)
		if previous != nil {
			_ = read(previous)
		}
		previous = pointer
	}
}
