package rejected

import im "github.com/lucasavila00/immutago/pkg/im"
type Good struct { im.Marker; X int }

func panicWith(pointer *int) { panic(pointer) }
func bad() { var value Good; defer func() { if payload := recover(); payload != nil { *payload.(*int) = 2 } }(); panicWith(&value.X) } // want "borrow|shared reference|unsupported"
