package rejected

import im "github.com/lucasavila00/immutago/pkg/im"
type Good struct { im.Marker; X int }

type Holder struct { Pointer *int }
func wrap(pointer *int) Holder { return Holder{Pointer: pointer} }
func bad() { var value Good; holder := wrap(&value.X); *holder.Pointer = 2 } // want "borrow|shared reference|unsupported"
