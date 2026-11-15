package analyzer

import "testing"

// TestCellSlotChecks covers accepted calls and rejected slot transport.
func TestCellSlotChecks(t *testing.T) {
	const definitions = `type Model struct { im.Frozen; X int }
 type Loader struct { Current im.Cell[Model] }
 `
	cases := []struct {
		name, source string
		reject       bool
	}{
		{"fresh store and local load", `func f(l *Loader) { l.Current.Store(Model{X:1}); m:=l.Current.Load(); read(m) }
func read(im.Projection[Model]){}`, false},
		{"new store", `func f(l *Loader) { l.Current.Store(Model{}); m:=l.Current.Load(); _=m }`, false},
		{"package slot", `var slot im.Cell[Model]; func f() { slot.Store(Model{}); m:=slot.Load(); _=m }`, false},
		{"snapshot argument", `type S struct{im.Frozen}; var slot im.Cell[S]`, false},
		{"invalid local argument", `type S struct{im.Local; Values []int}; var slot im.Cell[S]`, true},
		{"marked owner", `type Bad struct{im.Frozen; Slot im.Cell[Model]}`, true},
		{"slot copy", `func f(l *Loader) { other:=l.Current; _=other }`, true},
		{"slot address", `func f(l *Loader) { other:=&l.Current; _=other }`, true},
		{"zero holder assignment", `func f(l *Loader) { *l=Loader{} }`, true},
		{"zero slot assignment", `func f(l *Loader) { l.Current=im.Cell[Model]{} }`, true},
		{"range callback holder", `func seq(func(Loader) bool){};func f(){for range seq{}}`, true},
		{"tuple holder result", `func f()(int,Loader){return 0,Loader{}};func g(){_,_=f()}`, true},
		{"defined slot", `type Hidden im.Cell[Model]`, true},
		{"holder copy", `func f(l *Loader) { other:=*l; _=other }`, true},
		{"holder value parameter", `func f(Loader){}`, true},
		{"holder value result", `func f(l *Loader) Loader { return *l }`, true},
		{"holder value method", `func (l Loader) f(){}`, true},
		{"holder container", `var slots []Loader`, true},
		{"nested named holder", `type Outer struct{ Value Loader }`, false},
		{"embedded holder", `type Outer struct{ Loader }`, true},
		{"slot pointer", `type Outer struct{ Value *im.Cell[Model] }`, true},
		{"store retained value", `func f(l *Loader) { m:=l.Current.LoadShared(); l.Current.Store(m) }`, false},
		{"store existing value", `func f(l *Loader) { m:=Model{}; l.Current.Store(m) }`, false},
		{"return load", `func f(l *Loader) im.Projection[Model] { return l.Current.Load() }`, false},
		{"load into global", `var m im.Projection[Model]; func f(l *Loader) { m=l.Current.Load() }`, true},
		{"shared field write", `func f(l *Loader) { m:=l.Current.LoadShared(); m.X=1 }`, true},
		{"method value", `func f(l *Loader) { load:=l.Current.Load; _=load }`, true},
		{"concurrent store", `func f(l *Loader) { go l.Current.Store(Model{}) }`, true},
		{"deferred store", `func f(l *Loader) { defer l.Current.Store(Model{}) }`, true},
		{"loaded pointer closure", `func f(l *Loader){ p:=l.Current.Load(); g:=func(){_=p}; g() }`, true},
		{"holder closure", `func f(l *Loader){ g:=func(){l.Current.Store(Model{})}; g() }`, false},
		{"variadic pointer escape", `var saved []*Loader;func retain(v ...*Loader){saved=v}
func f(){l:=&Loader{};retain(l)}`, false},
		{"variadic pointer alias", `type P = *Loader; func retain(...P){};func f(l *Loader){retain(l)}`, false},
		{"pointer global", `var saved *Loader`, false},
		{"pointer alias global", `type P = *Loader;var saved P`, false},
		{"defined pointer global", `type P *Loader;var saved P`, false},
		{"pointer indirection", `var saved **Loader`, false},
		{"pointer array", `var saved [1]*Loader`, false},
		{"pointer slice", `var saved []*Loader`, false},
		{"pointer map", `var saved map[string]*Loader`, false},
		{"pointer channel", `var saved chan *Loader`, false},
		{"pointer field", `type Outer struct{ P *Loader }`, false},
		{"pointer result nil", `func f()*Loader{return nil}`, false},
		{"pointer named result", `func f()(p *Loader){return}`, false},
		{"pointer tuple result", `func f()(int,*Loader){return 0,nil}`, false},
		{"generic pointer result", `func f[T any]()(p *Loader){return}`, false},
		{"pointer type set", `type Set interface{*Loader}`, false},
		{"pointer generic constraint", `func f[T interface{*Loader}](){}`, false},
		{"pointer generic call", `func f[T any](){};func g(){f[*Loader]()}`, false},
		{"pointer interface", `type Callback interface{ Invoke(*Loader) }; var x Callback`, false},
		{"pointer callback", `var callback func(*Loader)`, false},
		{"pointer factory", `var factory func()*Loader`, false},
		{"pointer callback container", `var callbacks []func(*Loader)`, false},
		{"deferred publisher", `func publish(l *Loader){};func f(l *Loader){defer publish(l)}`, false},
		{"fixed holder parameter", `func publish(l *Loader, v ...int){};func f(l *Loader){publish(l,1)}`, false},
		{"nil pointer conversion", `func f(){p:=(*Loader)(nil);_=p}`, false},
		{"holder boxing", `func f(l *Loader){ var x any=l; _=x }`, false},
		{"holder pointer capture", `var saved *Loader; func f(l *Loader){ saved=l }`, false},
		{"background publisher", `func publish(l *Loader){l.Current.Store(Model{})}
func f(l *Loader){go publish(l)}`, false},
		{"holder new allocation", `func f(){ l:=new(Loader); l.Current.Store(Model{}) }`, false},
		{"holder named callback", `func publish(l *Loader){};func f(){callback:=publish;_=callback}`, false},
		{"holder method callback", `func(l *Loader) publish(){};func f(){callback:=(*Loader).publish;_=callback}`, false},
		{"holder construction", `func f(){ l:=&Loader{}; l.Current.Store(Model{}) }`, false},
		{"generic holder", `func f[T any](*T){}; func g(l *Loader){ f(l) }`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := checkSource(t, definitions+tc.source)
			if (got != "") != tc.reject {
				t.Fatalf("diagnostics=%q rejected=%v", got, tc.reject)
			}
		})
	}
}
