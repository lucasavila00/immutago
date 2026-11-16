package analyzer

import (
	"strings"
	"testing"
)

// TestOpaqueHandleMethods keeps embedded value calls but rejects implicit Frozen addresses.
func TestOpaqueHandleMethods(t *testing.T) {
	const definitions = opaqueHandleDefinitions + `
func (h *Handle) Self() *Handle { return h }
type Holder struct { Handle }
type PointerHolder struct { *Handle }
type Nested struct { Holder }
type NestedPointer struct { PointerHolder }
type Alias = Handle
type AliasHolder struct { Alias }
type Generic[T any] struct { im.Frozen; payload im.UncheckedValue[state] }
func (h *Generic[T]) Self() *Generic[T] { return h }
type GenericHolder struct { Generic[int] }
type GenericPointerHolder struct { *Generic[int] }
`
	for _, source := range []string{
		`func use(h Holder) *state { return h.Pointer() }`,
		`func use(h *Holder) func() *state { return h.Pointer }`,
		`func use(h Nested) *state { return Nested.Pointer(h) }`,
		`func use(h *Handle) *Handle { return h.Self() }`,
		`func use() *Handle { return new(Make()).Self() }`,
		`func use(h PointerHolder) *Handle { return h.Self() }`,
		`func use(h *PointerHolder) func() *Handle { return h.Self }`,
		`func use(h NestedPointer) *Handle { return NestedPointer.Self(h) }`,
		`func use(h *NestedPointer) *Handle { return (*NestedPointer).Self(h) }`,
		`func use(h GenericPointerHolder) *Generic[int] { return h.Self() }`,
	} {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, definitions+source); got != "" {
				t.Fatal(got)
			}
		})
	}
	for _, source := range []string{
		`func use(h Holder) *Handle { return h.Self() }`,
		`func use(h *Holder) *Handle { return h.Self() }`,
		`func use(h Holder) func() *Handle { return h.Self }`,
		`func use(h *Holder) *Handle { return (*Holder).Self(h) }`,
		`func use(h Nested) *Handle { return h.Self() }`,
		`func use(h *Nested) *Handle { return h.Self() }`,
		`func use(h AliasHolder) *Handle { return h.Self() }`,
		`func use(h GenericHolder) *Generic[int] { return h.Self() }`,
		`func use(h []Holder) *Handle { return h[0].Self() }`,
		`func use() *Handle { h := struct{Handle}{Make()}; return h.Self() }`,
		`func use() *Handle { h := Holder{Handle: Make()}; kept := h.Self(); h.Handle = Make(); return kept }`,
	} {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, definitions+source); !strings.Contains(got, "receivers must match their declared type exactly") {
				t.Fatalf("implicit Frozen address accepted: %s", got)
			}
		})
	}
}
