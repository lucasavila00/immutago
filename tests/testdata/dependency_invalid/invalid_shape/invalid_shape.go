package invalid_shape

import "github.com/lucasavila00/immutago/pkg/im"

type Bad struct {
	im.Frozen
	Values []int
}

func Number() int { return 1 }
