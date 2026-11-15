package contracts

import "github.com/lucasavila00/immutago/pkg/im"

type Raw interface {
	im.FrozenInterface
	Text() string
}
type Tuple interface {
	im.FrozenInterface
	Pair() (int, bool)
}
type Variadic interface {
	im.FrozenInterface
	Sum(...int) int
}
type Callback interface {
	im.FrozenInterface
	Call(func())
}
type Dynamic interface {
	im.FrozenInterface
	Use(any)
}
type Private interface {
	im.FrozenInterface
	read() int
}
