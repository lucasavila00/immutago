package rejected
import im "github.com/lucasavila00/immutago/pkg/im"
type Good struct { im.Marker; X int }
func bad(values [1]*Good) { values[0].X=2 } // want "borrow|shared reference|unsupported"
