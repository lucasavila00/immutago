package borrowapi

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Shared
	X int
}

func Read(pointer *Good) int { return pointer.X }
func Make(number int) Good   { return Good{X: number} }

func Shared(number int) *Good   { return &Good{X: number} }
func Forward(value *Good) *Good { return value }

type Local struct {
	im.Scoped
	X int
}

func ReadLocal(value *Local) int { return value.X }
