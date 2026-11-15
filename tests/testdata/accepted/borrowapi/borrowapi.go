//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package borrowapi

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Frozen
	X int
}

func Read(pointer *Good) int { return pointer.X }
func Make(number int) Good   { return Good{X: number} }

func Frozen(number int) *Good   { return &Good{X: number} }
func Forward(value *Good) *Good { return value }

type Local struct {
	im.Local
	X int
}

func ReadLocal(value *Local) int { return value.X }
