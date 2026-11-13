package rejected

import "github.com/lucasavila00/immutago/pkg/im"
type Good struct { im.Marker; X int }

var Global Good
func field() *int { return &Global.X }
func bad() { pointer := field(); *pointer = 2 } // want "borrow|shared reference|unsupported|package-level immutable"
