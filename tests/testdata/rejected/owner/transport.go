package owner

var savedLoaders []*Loader
var savedLoader *Loader
var holderMap map[string]*Loader
var holderChannel chan *Loader

type HolderPointer = *Loader
type HolderBox struct{ Pointer *Loader }

func retain(loaders ...*Loader)  { savedLoaders = loaders }
func makeLoader() (int, *Loader) { return 0, nil }

func badTransport(l *Loader) {
	retain(l)
	savedLoader = l
}

func publish(l *Loader)  { l.Current.Store(&Model{}) }
func (*Loader) publish() {}
func badCallbacks() {
	f := publish
	_ = f
	g := (*Loader).publish
	_ = g
}
