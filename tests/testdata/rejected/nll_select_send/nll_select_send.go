package rejected
import "github.com/lucasavila00/immutago/pkg/im"
type Good struct { im.Marker; X int }
var channel=make(chan *int,1)
func send(pointer *int) { select {case channel<-pointer: default:} }
func bad() { var value Good; send(&value.X); pointer:=<-channel; *pointer=2 } // want "borrow|shared reference|unsupported"
