package docrejected

import "github.com/lucasavila00/immutago/pkg/im"

// ConstantLocal is a Scoped object used in a constant array operand.
type ConstantLocal struct{ im.Scoped }

// constantCapture reads a protected reference in a closure, even though Go does not evaluate it.
func constantCapture(local *ConstantLocal) func() int {
	return func() int { return len([1]*ConstantLocal{local}) }
}

// constantLength uses protected construction in a constant declaration.
const constantLength = len([1]Point{})

// constantTypeLength uses protected construction in type syntax.
type constantTypeLength [len([1]Point{})]int

// constantGeneric uses protected construction in a generic declaration.
func constantGeneric[T any]() { var values [len([1]Point{})]int; _ = values }

// constantBuilderAlias mentions a builder in an array before Freeze.
func constantBuilderAlias() {
	values := []int{1}
	const n = len([1][]int{values})
	_ = im.FreezeSlice(values)
	println(n)
}

// constantConsumedElement reads a consumed builder inside a constant operand.
func constantConsumedElement() {
	values := []int{1}
	_ = im.FreezeSlice(values)
	const n = len([1]int{values[99]})
	println(n)
}
