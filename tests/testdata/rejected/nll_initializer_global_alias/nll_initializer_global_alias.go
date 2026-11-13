package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct { im.Marker; X int }
var Global Good
func globalPointer() *Good { return &Global }
func init() { pointer:=globalPointer(); Global.X=2; _=pointer.X } // want "borrow|shared reference|unsupported"
