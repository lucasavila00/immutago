package borrowapi

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	X int
}

func Read(pointer *Good) int { return pointer.X }
func Make(number int) Good   { var value Good; value.X = number; return value }
