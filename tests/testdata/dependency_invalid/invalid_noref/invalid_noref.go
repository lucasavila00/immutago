package invalid_noref

import "github.com/lucasavila00/immutago/pkg/im"

type Local struct {
	im.ImmutNoRef
	Value int
}

var retained *Local

func Retain(value *Local) { retained = value }
func Number() int         { return 1 }
