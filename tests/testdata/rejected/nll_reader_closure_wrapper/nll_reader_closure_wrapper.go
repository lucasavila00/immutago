package rejected

import im "github.com/lucasavila00/immutago/pkg/im"
type Good struct { im.Marker; X int }

func reader(pointer *int) func() int { return func() int { return *pointer } }
func bad() { var value Good; later := reader(&value.X); value.X = 2; _ = later() } // want "borrow|shared reference|unsupported"
