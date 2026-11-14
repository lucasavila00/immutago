package analyzer

import "testing"

// TestOwnerSlotChecks covers accepted calls and rejected slot transport.
func TestOwnerSlotChecks(t *testing.T) {
	const definitions = `type Model struct { im.Scoped; X int }
 type Loader struct { Current im.Owner[Model] }
 `
	cases := []struct {
		name, source string
		reject       bool
	}{
		{"fresh store and local load", `func f(l *Loader) { l.Current.Store(&Model{X:1}); m:=l.Current.Load(); read(m) }
func read(*Model){}`, false},
		{"new store", `func f(l *Loader) { l.Current.Store(new(Model)); m:=l.Current.Load(); _=m }`, false},
		{"package slot", `var slot im.Owner[Model]; func f() { slot.Store(&Model{}); m:=slot.Load(); _=m }`, false},
		{"shared argument", `type S struct{im.Shared}; var slot im.Owner[S]`, true},
		{"invalid local argument", `type S struct{im.Scoped; Values []int}; var slot im.Owner[S]`, true},
		{"marked owner", `type Bad struct{im.Shared; Slot im.Owner[Model]}`, true},
		{"slot copy", `func f(l *Loader) { other:=l.Current; _=other }`, true},
		{"slot address", `func f(l *Loader) { other:=&l.Current; _=other }`, true},
		{"zero holder assignment", `func f(l *Loader) { *l=Loader{} }`, true},
		{"zero slot assignment", `func f(l *Loader) { l.Current=im.Owner[Model]{} }`, true},
		{"range callback holder", `func seq(func(Loader) bool){};func f(){for range seq{}}`, true},
		{"tuple holder result", `func f()(int,Loader){return 0,Loader{}};func g(){_,_=f()}`, true},
		{"defined slot", `type Hidden im.Owner[Model]`, true},
		{"holder copy", `func f(l *Loader) { other:=*l; _=other }`, true},
		{"holder value parameter", `func f(Loader){}`, true},
		{"holder value result", `func f(l *Loader) Loader { return *l }`, true},
		{"holder value method", `func (l Loader) f(){}`, true},
		{"holder container", `var slots []Loader`, true},
		{"nested holder", `type Outer struct{ Value Loader }`, true},
		{"slot pointer", `type Outer struct{ Value *im.Owner[Model] }`, true},
		{"store loaded pointer", `func f(l *Loader) { m:=l.Current.Load(); l.Current.Store(m) }`, true},
		{"store new value", `func f(l *Loader) { m:=Model{}; l.Current.Store(new(m)) }`, true},
		{"return load", `func f(l *Loader) *Model { return l.Current.Load() }`, true},
		{"load into global", `var m *Model; func f(l *Loader) { m=l.Current.Load() }`, true},
		{"load write", `func f(l *Loader) { l.Current.Load().X=1 }`, true},
		{"method value", `func f(l *Loader) { load:=l.Current.Load; _=load }`, true},
		{"concurrent store", `func f(l *Loader) { go l.Current.Store(&Model{}) }`, true},
		{"deferred store", `func f(l *Loader) { defer l.Current.Store(&Model{}) }`, true},
		{"holder closure", `func f(l *Loader){ g:=func(){l.Current.Store(&Model{})}; g() }`, true},
		{"variadic pointer escape", `var saved []*Loader;func retain(v ...*Loader){saved=v}
func f(){l:=&Loader{};retain(l)}`, true},
		{"variadic pointer alias", `type P = *Loader; func retain(...P){};func f(l *Loader){retain(l)}`, true},
		{"pointer global", `var saved *Loader`, true},
		{"pointer alias global", `type P = *Loader;var saved P`, true},
		{"defined pointer global", `type P *Loader;var saved P`, true},
		{"pointer indirection", `var saved **Loader`, true},
		{"pointer array", `var saved [1]*Loader`, true},
		{"pointer slice", `var saved []*Loader`, true},
		{"pointer map", `var saved map[string]*Loader`, true},
		{"pointer channel", `var saved chan *Loader`, true},
		{"pointer field", `type Outer struct{ P *Loader }`, true},
		{"pointer result nil", `func f()*Loader{return nil}`, true},
		{"pointer named result", `func f()(p *Loader){return}`, true},
		{"pointer tuple result", `func f()(int,*Loader){return 0,nil}`, true},
		{"generic pointer result", `func f[T any]()(p *Loader){return}`, true},
		{"pointer type set", `type Set interface{*Loader}`, true},
		{"pointer generic constraint", `func f[T interface{*Loader}](){}`, true},
		{"pointer generic call", `func f[T any](){};func g(){f[*Loader]()}`, true},
		{"pointer interface", `type Callback interface{ Invoke(*Loader) }; var x Callback`, true},
		{"pointer callback", `var callback func(*Loader)`, true},
		{"pointer factory", `var factory func()*Loader`, true},
		{"pointer callback container", `var callbacks []func(*Loader)`, true},
		{"deferred publisher", `func publish(l *Loader){};func f(l *Loader){defer publish(l)}`, true},
		{"fixed holder parameter", `func publish(l *Loader, v ...int){};func f(l *Loader){publish(l,1)}`, false},
		{"nil pointer conversion", `func f(){p:=(*Loader)(nil);_=p}`, true},
		{"holder boxing", `func f(l *Loader){ var x any=l; _=x }`, true},
		{"holder pointer capture", `var saved *Loader; func f(l *Loader){ saved=l }`, true},
		{"background publisher", `func publish(l *Loader){l.Current.Store(&Model{})}
func f(l *Loader){go publish(l)}`, false},
		{"holder new allocation", `func f(){ l:=new(Loader); l.Current.Store(&Model{}) }`, false},
		{"holder named callback", `func publish(l *Loader){};func f(){callback:=publish;_=callback}`, true},
		{"holder method callback", `func(l *Loader) publish(){};func f(){callback:=(*Loader).publish;_=callback}`, true},
		{"holder construction", `func f(){ l:=&Loader{}; l.Current.Store(&Model{}) }`, false},
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
