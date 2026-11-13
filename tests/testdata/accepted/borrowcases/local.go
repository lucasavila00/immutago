package borrowcases

import im "github.com/lucasavila00/immutago/pkg/im"

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
func identity[T any](pointer *T) *T      { return pointer }
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

func DisjointFields() Good {
	var value Good
	pointer := &value.X
	value.Y = 2
	_ = readInt(pointer)
	value.X = 3
	return value
}

func NestedDisjointFields() Good {
	var value Good
	pointer := &value.Child.X
	value.Child.Y = 2
	_ = readInt(pointer)
	value.Child.X = 3
	return value
}

func DisjointArrayElements() Good {
	var value Good
	pointer := &value.Array[0]
	value.Array[1] = 2
	_ = readInt(pointer)
	value.Array[0] = 3
	return value
}

func RHSRead() Good {
	var value Good
	value.X = read(&value) + 1
	return value
}

func FreshPointer() *Good {
	value := Good{X: 1}
	return &value
}

func IdentityLastUse() Good {
	var value Good
	pointer := identity(&value)
	_ = read(pointer)
	value.X = 2
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

func SliceViewLastUse() Good {
	var value Good
	view := value.Array[:]
	_ = view[0]
	value.Array[0] = 2
	return value
}

func FreshOwner(condition bool) *Good {
	pointer := &Good{}
	if condition {
		pointer.X = 1
	} else {
		pointer.X = 2
	}
	pointer.Y++
	return pointer
}

func NewOwner() *Good {
	pointer := new(Good)
	pointer.X = 1
	return pointer
}

func OwnerAfterBorrow() *Good {
	owner := &Good{}
	shared := owner
	_ = read(shared)
	owner.X = 2
	return owner
}
