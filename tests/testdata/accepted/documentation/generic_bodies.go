package main

// genericPoint reads a concrete Frozen value inside a function with ordinary type parameters.
func genericPoint[T any]() int {
	value := Point{X: 1}
	return value.X
}

type ordinaryReader[T any] struct{ Value T }

// Read uses the same concrete field read inside a generic receiver body.
func (ordinaryReader[T]) Read() int {
	value := Point{X: 2}
	return value.X
}

// genericBodies instantiates ordinary types while both bodies check their concrete Frozen values.
func genericBodies() {
	_ = genericPoint[int]()
	reader := ordinaryReader[int]{}
	_ = reader.Read()
}
