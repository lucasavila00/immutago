package owner

import "github.com/lucasavila00/immutago/pkg/im"

type Model struct {
	im.Scoped
	X int
}
type Shared struct{ im.Shared }
type Loader struct{ Current im.Owner[Model] }
type BadOwner struct {
	im.Shared
	Current im.Owner[Model]
}

var bad im.Owner[Shared]
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
