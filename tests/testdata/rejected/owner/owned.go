package owner

import "github.com/lucasavila00/immutago/pkg/im"

type Model struct {
	im.Frozen
	X int
}
type Local struct{ im.Local }
type Loader struct{ Current im.Cell[Model] }
type BadCell struct {
	im.Frozen
	Current im.Cell[Model]
}

var bad im.Cell[Local]
var saved im.Projection[Model]

func badUses(l *Loader) {
	alias := l.Current
	_ = alias
	copy := *l
	_ = copy
	saved = l.Current.Load()
	m := l.Current.LoadShared()
	m.X = 2
}
