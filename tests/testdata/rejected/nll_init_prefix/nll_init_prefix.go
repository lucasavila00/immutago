package rejected

import "github.com/lucasavila00/immutago/pkg/im"
type Good struct { im.Marker; X int }

var Global Good
func initializeLater() { Global.X = 2 } // want "borrow|shared reference|unsupported|package-level immutable"
func bad() { initializeLater() }
