package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct { im.Marker; X int }
func choose(payload any, direct bool) *Good {
 if direct { return payload.(*Good) }
 return *payload.(**Good)
}
func bad() { var value Good; pointer:=&value; alias:=choose(&pointer,false); value.X=2; _=alias.X } // want "borrow|shared reference|unsupported"
