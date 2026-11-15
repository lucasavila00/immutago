package analyzer

import (
	"go/ast"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
)

const sharedHolderTypes = `type Holder struct{Slot im.Cell[int]; N int}
type Nested struct{Child Holder}
type Parent struct{Child *Holder}
type Embedded struct{*Holder}
type Alias = Holder
func(h *Holder) Add(n int){h.N+=n;h.Slot.Store(h.N)}
func(h *Holder) View()im.Projection[int]{return h.Slot.Load()}
`

// TestSharedHolderPointers checks ordinary references without permitting slot copies.
func TestSharedHolderPointers(t *testing.T) {
	for _, source := range []string{
		`func makeHolder()*Holder{return &Holder{N:1}}`,
		`func makeHolder()*Holder{var h Holder;return &h}`,
		`func child(n *Nested)*Holder{return &n.Child}`,
		`func f(h *Holder){h.N++;(*h).N=2;p:=&h.N;*p=3}`,
		`var saved=new(Holder);func f(h *Holder){saved=h}`,
		`func f(h *Holder)*Holder{return Parent{Child:h}.Child}`,
		`func f(h *Holder){p:=&h;*p=new(Holder);h, *p=*p,h}`,
		`func pair()(*Holder,*Holder){return new(Holder),nil};var a,b=pair()`,
		`func f(h *Holder){a:=[]*Holder{h,{N:1}};a=append(a,h);copy(a,a);clear(a)}`,
		`func f(h *Holder){a:=[1]*Holder{h};for _,v:=range a{v.Add(1)}}`,
		`func f(h *Holder){m:=map[*Holder]*Holder{h:h};for k,v:=range m{k.Add(v.N)};delete(m,h)}`,
		`func f(h *Holder){ch:=make(chan *Holder,1);select{case ch<-h:default:};h=<-ch;h.Add(1)}`,
		`func f(h *Holder){var x any=h;p,ok:=x.(*Holder);if ok{p.Add(1)};switch v:=x.(type){case *Holder:v.Add(1)}}`,
		`func f(h *Holder){switch h{case nil:case h:};_=h==(*Alias)(nil)}`,
		`type Pointer *Holder;func f(h *Holder){p:=Pointer(h);p.N++;_=(*Holder)(p)}`,
		`func f(h *Holder){call:=h.Add;call(1);expr:=(*Holder).Add;expr(h,2);go call(3);defer expr(h,4)}`,
		`func f(){var h Holder;call:=h.Add;call(1);defer h.Add(2)}`,
		`func f(n *Nested){call:=n.Child.Add;call(1);(&n.Child).Add(2)}`,
		`func f(h *Holder){e:=Embedded{h};e.Add(1);call:=e.Add;call(2);Embedded.Add(e,3)}`,
		`type Count int
func(c Count)Read()int{return int(c)}
type Mixed struct{Slot im.Cell[int];Count}
func f(){var h Mixed;_=h.Read();call:=h.Read;_=call();_=(&h).Read();pointerCall:=(&h).Read;_=pointerCall()}`,
		`type Adder interface{Add(int)};func f(h *Holder){var a Adder=h;a.Add(1)}`,
		`func f(h *Holder)func()*Holder{return func()*Holder{h.Add(1);return h}}`,
		`func f()func()int{var h Holder;p:=&h;return func()int{return p.N}}`,
		`func f()func()int{var h Holder;return func()int{return h.N}}`,
		`func f()func(){var h Holder;return func(){h.Add(1)}}`,
		`func f()func()*Holder{var h Holder;return func()*Holder{return &h}}`,
		`func f()func()int{var h Nested;return func()int{return h.Child.N}}`,
		`var h Holder;func f()func()int{return func()int{return h.N}}`,
		`var slot im.Cell[int];func f()func(){return func(){slot.Store(1)}}`,
		`func id[T any](v T)T{return v};func f(h *Holder)*Holder{return id(h)}`,
		`func id[T ~*Holder](v T)T{return v};func f(h *Holder)*Holder{return id(h)}`,
		`type Box[T any]struct{Value T};func f(h *Holder)Box[*Holder]{return Box[*Holder]{h}}`,
		`func f[T any](v T)*Holder{return new(Holder)}`,
		`func id[T any](v T)T{return v};var empty=id[*Holder](nil)`,
		`func f()*struct{Slot im.Cell[int]}{return &struct{Slot im.Cell[int]}{}}`,
		`type Callback func(...*Holder)*Holder;func f(cb Callback,h *Holder)*Holder{return cb(h,h)}`,
	} {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, sharedHolderTypes+source); got != "" {
				t.Fatalf("holder pointer rejected: %s", got)
			}
		})
	}
}

// TestSharedHolderRejectsCopies keeps values, raw slots, and Local results restricted.
func TestSharedHolderRejectsCopies(t *testing.T) {
	for _, source := range []string{
		`func f(h *Holder){copy:=*h;_=copy}`,
		`func f(h *Holder){*h=Holder{}}`,
		`func f(h *Holder){*h=*h}`,
		`func f(h *Holder){h.Slot=im.Cell[int]{}}`,
		`func f(h *Holder){var copy Alias=*h;_=copy}`,
		`func f(h *Holder){var x any=*h;_=x}`,
		`func f(h Holder){}`,
		`type Contract interface{Copy(Holder) Holder}`,
		`type Callback func(Holder) Holder`,
		`type Contract interface{Slot() im.Cell[int]}`,
		`func f(h *Holder)Holder{return *h}`,
		`func f(h *Holder)(int,Holder){return 0,*h}`,
		`func(h Holder) Copy(){}`,
		`func f(n *Nested){n.Child=Holder{}}`,
		`func f(n *Nested){_,n.Child=0,Holder{}}`,
		`func f(h *Holder){_,*h=0,Holder{}}`,
		`type Bad struct{Holder}`,
		`type Bad struct{Values [1]Holder}`,
		`type Bad struct{im.Frozen;Child *Holder}`,
		`type Bad struct{im.Local;Child *Holder}`,
		`var bad im.Cell[*Holder]`,
		`func f(h *Holder){_=Parent{Child:&Holder{Slot:h.Slot}}}`,
		`func f(h *Holder){_=Nested{Child:*h}}`,
		`func id[T any](v *T)*T{return v};func f(h *Holder){_=id(h)}`,
		`func copy[T ~*Holder](p T){_=*p}`,
		`func f(h *Holder){slot:=&h.Slot;_=slot}`,
		`func f(){_=new(im.Cell[int])}`,
		`func f(e Embedded){_=e.Slot.Load()}`,
		`var saved im.Projection[int];func f(h *Holder){saved=h.View()}`,
		`func f(h *Holder)func(){view:=h.View();return func(){_=view}}`,
		`func f(h *Holder){view:=h.View;_=view}`,
		`func f(h *Holder){go h.View()}`,
		`type Count int
func(c Count)Read()int{return int(c)}
type Mixed struct{Slot im.Cell[int];Count}
func f(p *Mixed){_=Mixed.Read(*p)}`,
	} {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, sharedHolderTypes+source); got == "" {
				t.Fatal("slot copy or Local escape accepted")
			}
		})
	}
}

// TestImplicitSlotReceiverCopy checks selections without relying on the declaration error.
func TestImplicitSlotReceiverCopy(t *testing.T) {
	for _, use := range []string{"h.Copy()", "call:=h.Copy;call()", "(*Holder).Copy(h)", "e.Copy()"} {
		pass := sourcePass(t, sharedHolderTypes+`func(h Holder)Copy(){};func f(h *Holder,e Embedded){`+use+`}`)
		var diagnostics []string
		pass.Report = func(d analysis.Diagnostic) { diagnostics = append(diagnostics, d.Message) }
		checks := newSubsetChecks(pass, nil)
		for _, file := range pass.Files {
			ast.Inspect(file, func(node ast.Node) bool {
				if selector, ok := node.(*ast.SelectorExpr); ok {
					checks.checkSlotSelection(selector)
				}
				return true
			})
		}
		if !strings.Contains(strings.Join(diagnostics, "\n"), "copied by a value receiver") {
			t.Fatalf("%s: implicit holder copy accepted", use)
		}
	}
}
