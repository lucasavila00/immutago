package owned

import "github.com/lucasavila00/immutago/pkg/im"

type Model struct {
	im.LocalRef
	X int
}
type Shared struct{ im.Shared }
type Loader struct{ Current im.Owned[Model] }
type BadOwner struct {
	im.Shared
	Current im.Owned[Model]
}

var bad im.Owned[Shared]
var saved *Model

func badUses(l *Loader) {
	alias := l.Current
	_ = alias
	copy := *l
	_ = copy
	m := l.Current.Load()
	l.Current.Store(m)
	saved = l.Current.Load()
	l.Current.Load().X = 2
}
