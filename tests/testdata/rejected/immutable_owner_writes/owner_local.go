package borrowcases

import "github.com/lucasavila00/immutago/pkg/im"

type Child struct {
	im.Shared
	X, Y int
}
type Good struct {
	im.Shared
	X, Y  int
	Child Child
	Array [2]int
}

func read(pointer *Good) int   { return pointer.X }
func readInt(pointer *int) int { return *pointer }

func Conditional(condition bool) Good {
	var value Good
	if condition {
		value.X = 1
	} else {
		value.X = 2
	}
	value.X += 3
	value.Y++
	return value
}

func AfterRead() Good {
	var value Good
	pointer := new(value)
	_ = read(pointer)
	value.X = 2
	value = Good{X: 3}
	return value
}

func SeparateReferences() Good {
	var first Good
	var second Good
	pointer := new(first)
	_ = read(pointer)
	other := new(second)
	first.X = 2
	_ = read(other)
	second.X = 3
	return first
}

func RHSRead() Good {
	var value Good
	value.X = read(new(value)) + 1
	return value
}

func ValueCopies() Good {
	value := Good{X: 1}
	copied := value
	pointer := new(value)
	copied.X = 2
	_ = read(pointer)
	return copied
}
