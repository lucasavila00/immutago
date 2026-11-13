package invalid_freeze

import "github.com/lucasavila00/immutago/pkg/im"

type View = im.FrozenSlice[int]

func Value() View {
	values := []int{1}
	frozen := im.FreezeSlice(values)
	values[0] = 2
	return frozen
}
