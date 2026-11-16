package analyzer

import "testing"

const opaqueHandleDefinitions = `
type state struct { N int }
type Handle struct { im.Frozen; payload im.UncheckedValue[state] }
func Make() Handle { return Handle{payload: im.Trust(&state{})} }
func (h Handle) Pointer() *state { return im.Unwrap(h.payload) }
`

// TestOpaqueHandles checks ordinary storage and immutable wrapper fields.
func TestOpaqueHandles(t *testing.T) {
	for _, source := range []string{
		`type Holder struct{ H Handle }; var stored Handle; func use(h Handle) *state { stored=h; h.Pointer().N++; return h.Pointer() }`,
		`type HandleReply struct { im.Local; H Handle }; func(c *Counter) Handle() Handle {return Make()}`,
		`type Wrapper struct{im.Frozen; H Handle};func use(w Wrapper) Handle{return w.H}`,
		`func use(h Handle){ var boxed any=h; _=boxed; f:=func()*state{return h.Pointer()};go f() }`,
		`type data[T any] struct{Value T}; type Generic[T any] struct{im.Frozen; payload im.UncheckedValue[data[T]]}
    func (h Generic[T]) Pointer()*data[T]{return im.Unwrap(h.payload)}
    func makeGeneric() Generic[int]{return Generic[int]{payload:im.Trust(&data[int]{})}}
    func useGeneric() int { h:=makeGeneric();h.Pointer().Value=2;return h.Pointer().Value }`,
	} {
		if got := checkSource(t, serviceDefinitions+opaqueHandleDefinitions+source); got != "" {
			t.Fatal(got)
		}
	}
	for _, source := range []string{
		`func use(h Handle){h.payload=im.Trust(&state{})}`,
		`func use(h *Handle){*h=Make()}`,
		`func use(h Handle){_= &h.payload}`,
		`func use(h Handle)*Handle{return &h}`,
		`type Bad struct{im.Copied;H Handle}`,
		`type Local struct{im.Local};type Bad struct{im.Frozen;payload im.UncheckedValue[Local]}`,
	} {
		if got := checkSource(t, serviceDefinitions+opaqueHandleDefinitions+source); got == "" {
			t.Fatalf("accepted: %s", source)
		}
	}
}
