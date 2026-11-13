package rejected

import im "github.com/lucasavila00/immutago/pkg/im"
type Good struct { im.Marker; X int }

func setter(pointer *int) func() { return func() { *pointer = 2 } }
func bad() { var value Good; change := setter(&value.X); change() } // want "borrow|shared reference|unsupported"
