package rejected

import im "github.com/lucasavila00/immutago/pkg/im"
type Good struct { im.Marker; X int }

var external func(*int)
func wrap(pointer *int) { external(pointer) }
func bad() { var value Good; wrap(&value.X) } // want "borrow|shared reference|unsupported"
