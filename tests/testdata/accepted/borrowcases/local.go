package borrowcases

import "github.com/lucasavila00/immutago/pkg/im"

type Child struct {
	im.Marker
	X, Y int
}
type Good struct {
	im.Marker
	X, Y  int
	Child Child
	Array [2]int
}

func read(pointer *Good) int             { return pointer.X }
func readInt(pointer *int) int           { return *pointer }
func (value *Good) Read() int            { return value.X }
func (value Good) WithX(number int) Good { value.X = number; return value }

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

func LastUse() Good {
	var value Good
	pointer := &value
	_ = read(pointer)
	value.X = 2
	value = Good{X: 3}
	return value
}

func CopyAlias() Good {
	var value Good
	pointer := &value
	alias := pointer
	pointer = nil
	_ = read(alias)
	value.X = 2
	_ = pointer
	return value
}

func RebindAlias() Good {
	var first, second Good
	pointer := &first
	_ = read(pointer)
	pointer = &second
	first.X = 2
	_ = read(pointer)
	second.X = 3
	return first
}

func RHSRead() Good {
	var value Good
	value.X = read(&value) + 1
	return value
}

func ValueCopies() Good {
	value := Good{X: 1}
	copied := value
	pointer := &value
	copied.X = 2
	_ = read(pointer)
	return copied
}
