package borrowapi

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	X int
}

func Read(pointer *Good) int               { return pointer.X }
func Identity(pointer *Good) *Good         { return pointer }
func GenericIdentity[T any](pointer *T) *T { return pointer }
func Fresh(number int) *Good               { pointer := &Good{X: number}; return pointer }

func Field(pointer *Good) *int { return &pointer.X }

func FreshFields() (*int, *int) {
	value := Good{X: 4}
	return &value.X, &value.X
}

func FreshPair() (*Good, *Good) {
	pointer := &Good{}
	return pointer, pointer
}

func SeparateFresh() (*Good, *Good) { return &Good{}, &Good{} }
