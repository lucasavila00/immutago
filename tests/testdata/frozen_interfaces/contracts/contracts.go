package contracts

import "github.com/lucasavila00/immutago/pkg/im"

type Raw interface {
	im.FrozenInterface
	Text() []im.StringProjection
}
type Tuple interface {
	im.FrozenInterface
	Pair() (int, *Local)
}
type Variadic interface {
	im.FrozenInterface
	Sum(...im.StringProjection) int
}
type Callback interface {
	im.FrozenInterface
	Call(func() im.StringProjection)
}
type Dynamic interface {
	im.FrozenInterface
	Use(interface{ Read() im.StringProjection })
}
type Private interface {
	im.FrozenInterface
	read() int
}

type Local struct {
	im.Local
	N int
}

// Protected variadic elements retain their native-slice restrictions.
type LocalVariadic interface {
	im.FrozenInterface
	Use(...*Local)
}
type CallbackVariadic interface {
	im.FrozenInterface
	Use(...func() im.StringProjection)
}
