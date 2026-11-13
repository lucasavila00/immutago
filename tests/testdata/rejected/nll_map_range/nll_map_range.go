package rejected
import "github.com/lucasavila00/immutago/pkg/im"
type Good struct { im.Marker; X int }
func bad(values map[int]*Good) { for _,pointer:=range values { pointer.X=2 } } // want "borrow|shared reference|unsupported"
