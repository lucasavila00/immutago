package borrowapi

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Immut
	X int
}

func Read(pointer *Good) int { return pointer.X }
func Make(number int) Good   { return Good{X: number} }

func Shared(number int) *Good   { return &Good{X: number} }
func Forward(value *Good) *Good { return value }
