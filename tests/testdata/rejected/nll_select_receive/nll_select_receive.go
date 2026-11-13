package rejected
import im "github.com/lucasavila00/immutago/pkg/im"
type Good struct { im.Marker; X int }
func bad(channel <-chan *Good) { select {case pointer:=<-channel: pointer.X=2; default:} } // want "borrow|shared reference|unsupported"
