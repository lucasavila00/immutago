package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct { im.Marker; X int }
func chooseScalar(payload any, direct bool) *int {
 if direct { return payload.(*int) }
 return *payload.(**int)
}
func bad() { var value Good; pointer:=&value.X; alias:=chooseScalar(&pointer,false); value.X=2; _=*alias } // want "borrow|shared reference|unsupported"
