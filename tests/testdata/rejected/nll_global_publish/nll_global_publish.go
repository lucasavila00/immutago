package rejected

import im "github.com/lucasavila00/immutago/pkg/im"
type Good struct { im.Marker; X int }

var Published *int
func save() { var value Good; Published = &value.X } // want "borrow|shared reference|unsupported"
func mutate() { *Published = 2 }
