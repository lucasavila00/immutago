package owner

import "github.com/lucasavila00/immutago/pkg/im"

type Model struct {
	im.Snapshot
	X int
}
type Local struct{ im.Local }
type Loader struct{ Current im.Cell[Model] }
type BadCell struct {
	im.Snapshot
	Current im.Cell[Model]
}

var bad im.Cell[Local]
var saved im.View[Model]

func badUses(l *Loader) {
	alias := l.Current
	_ = alias
	copy := *l
	_ = copy
	saved = l.Current.Load()
	m := l.Current.Shared()
	m.X = 2
}
