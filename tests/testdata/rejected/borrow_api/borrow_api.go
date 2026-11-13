package borrow_api

import im "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	X int
}

func Identity(pointer *Good) *Good { return pointer }
func Read(pointer *Good) int       { return pointer.X }

func Field(pointer *Good) *int { return &pointer.X }

func FreshFields() (*int, *int) {
	value := Good{X: 4}
	return &value.X, &value.X
}

func Fresh(number int) *Good { pointer := &Good{X: number}; return pointer }

func FreshPair() (*Good, *Good) {
	pointer := &Good{}
	return pointer, pointer
}
