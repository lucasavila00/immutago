package invalid_borrow

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Shared
	X int
}

var Published *int

func Publish()    { var value Good; Published = &value.X }
func Mutate()     { *Published = 2 }
func Number() int { return 1 }
