package analyzer

import "testing"

const holderProjectionTypes = `type Model struct{im.Frozen;Name string;Count int}
type TypedModelProjection struct{im.Local;value im.Projection[Model]}
func(v TypedModelProjection) Count()int{return NewModelProjection(im.CopyProjection(v.value)).Count()}
type TypedModelCell struct{value im.Cell[Model]}
func(c *TypedModelCell) Store(v Model){c.value.Store(v)}
func(c *TypedModelCell) Load()TypedModelProjection{return TypedModelProjection{value:c.value.Load()}}
func(c *TypedModelCell) Shared()Model{return c.value.Shared()}
func(c *TypedModelCell) Count()int{return 0}
type Holder struct{Current TypedModelCell}
var current TypedModelCell
`

// TestHolderMethods checks ordinary typed wrappers without generated-code privileges.
func TestHolderMethods(t *testing.T) {
	cases := []struct {
		name, source string
		reject       bool
	}{
		{"package", `func f(){current.Store(Model{});v:=current.Load();_=v.Count()}`, false},
		{"local value", `func f(){var c TypedModelCell;c.Store(Model{});_=c.Load()}`, false},
		{"local pointer", `func f(){c:=&TypedModelCell{};c.Store(Model{});_=c.Load()}`, false},
		{"pointer parameter", `func f(c *TypedModelCell){c.Store(Model{});_=c.Shared()}`, false},
		{"nested holder", `func f(h *Holder){h.Current.Store(Model{});_=h.Current.Load()}`, false},
		{"nested dereference", `func f(h *Holder){(*h).Current.Store(Model{});_=(*h).Current.Load()}`, false},
		{"new holder", `func f(){h:=new(Holder);h.Current.Store(Model{});_=h.Current.Load()}`, false},
		{"existing direct call", `func read(*TypedModelCell){};func f(c *TypedModelCell){read(c)}`, false},
		{"copy", `func f(){other:=current;_=other}`, true},
		{"nested copy", `func f(h *Holder){other:=h.Current;_=other}`, true},
		{"replace", `func f(){current=TypedModelCell{}}`, true},
		{"nested replace", `func f(h *Holder){h.Current=TypedModelCell{}}`, true},
		{"nested address", `func f(h *Holder){p:=&h.Current;_=p}`, true},
		{"value method", `func(c TypedModelCell) Bad(){}`, true},
		{"method value", `func f(){load:=current.Load;_=load}`, true},
		{"method expression", `func f(c *TypedModelCell){_=(*TypedModelCell).Load(c)}`, true},
		{"promoted method", `type Outer struct{TypedModelCell};func f(o *Outer){_=o.Load()}`, true},
		{"generic method", `type Generic[T any]struct{value im.Cell[int]}
func(c *Generic[T]) Read()int{return c.value.Shared()};func f(c *Generic[int]){_=c.Read()}`, true},
		{"generic holder declaration", `type Generic[T any]struct{value im.Cell[int]}`, true},
		{"generic holder alias", `type Generic[T any]=struct{value im.Cell[int]}`, true},
		{"unused generic method", `type Generic[T any]struct{value im.Cell[int]};func(c *Generic[T]) Read(){}`, true},
		{"generic holder constraint", `type Generic[T interface{TypedModelCell}]struct{}`, true},
		{"named slot constraint", `type Slots interface{im.Cell[int]};func F[T Slots](value T)T{return value}`, true},
		{"slot union constraint", `type Slots interface{~int|im.Cell[int]};type Alias=Slots
func F[T Alias](value T)T{return value}`, true},
		{"go projection", `func f(){go current.Load()}`, true},
		{"defer scalar", `func f(){defer current.Count()}`, true},
		{"nested go scalar", `func use(int){};func f(){go use(current.Count())}`, true},
		{"boxed", `func f(c *TypedModelCell){_=any(c)}`, true},
		{"result", `func f()*TypedModelCell{return nil}`, true},
		{"array field", `type Outer struct{Values [1]TypedModelCell}`, true},
		{"pointer field", `type Outer struct{Value *TypedModelCell}`, true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := checkSource(t, holderProjectionTypes+test.source); (got != "") != test.reject {
				t.Fatalf("diagnostics=%q; reject=%v", got, test.reject)
			}
		})
	}
}
