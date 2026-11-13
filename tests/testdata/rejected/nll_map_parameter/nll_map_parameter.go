package rejected

import im "github.com/lucasavila00/immutago/pkg/im"
type Good struct { im.Marker; X int }

func bad(values map[string]*Good) { pointer := values["key"]; pointer.X = 2 } // want "borrow|shared reference|unsupported"
