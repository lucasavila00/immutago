package review_convergence

import "github.com/lucasavila00/immutago/pkg/im"

type CapturedValue struct {
	im.Shared
	N int
}

var readLaterValue = func() int { return laterValue.N }
var laterValue = &CapturedValue{N: 1}
