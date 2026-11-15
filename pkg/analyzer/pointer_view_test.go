package analyzer

import (
	"strings"
	"testing"
)

const pointerViewTypes = `type Child struct{im.Frozen;Name string;Number int}
type Root struct{im.Frozen;Child Child;Children [2]Child;Pointer *Child}
var cell im.Cell[Root]
`

// TestPointerViewForms checks immutable field addresses and explicit root copies.
func TestPointerViewForms(t *testing.T) {
	for _, source := range []string{
		`func f(){cell.Store(Root{});view:=cell.LoadPointer();if !im.IsNil(view){
child:=im.Select(view,func(root *Root)*Child{return &root.Child})
_=im.Text(im.Select(child,func(child *Child)string{return child.Name})).Len()}}`,
		`func f(view im.View[*Root],index int){
child:=im.Select(view,func(root *Root)*Child{return &root.Children[index]})
_=im.Copy(im.Select(child,func(child *Child)int{return child.Number}))}`,
		`func f(view im.View[*Root]){_=im.Select(view,func(root *Root)*Child{return &(*root).Child})}`,
		`func f(){frozen:=im.FreezeSlice([]Child{{}});values:=im.Slice(im.ViewOf(frozen));_=values.AtPointer(0)}`,
		`func f(){frozen:=im.FreezeMap(map[string]Child{"name":{}});values:=im.Map(im.ViewOf(frozen))
value,ok:=values.Get("name");_=ok;pointer:=im.CopyView(value);_=im.IsNil(pointer)}`,
	} {
		if got := checkSource(t, pointerViewTypes+source); got != "" {
			t.Fatal(got)
		}
	}
}

// TestPointerProjectionBoundaries rejects addresses that do not refer to a frozen root.
func TestPointerProjectionBoundaries(t *testing.T) {
	for _, projection := range []string{
		`func(root Root)*Child{return &root.Child}`,
		`func(root *Root)**Root{return &root}`,
		`func(root *Root)**Child{return &root.Pointer}`,
		`func(root *Root)*int{return &root.Child.Number}`,
		`func(root *Root)*Root{return &*root}`,
		`func(root *Root)*Child{return &outside}`,
	} {
		input := "im.ViewOf(&Root{})"
		if strings.Contains(projection, "(root Root)") {
			input = "im.ViewOf(Root{})"
		}
		source := pointerViewTypes + `func f(){outside:=Child{};_=outside;_=im.Select(` + input + `,` + projection + `)}`
		if got := checkSource(t, source); !strings.Contains(got, "inline projection path") {
			t.Fatal(got)
		}
	}
}

// TestPointerAPIRecordTypes uses the same Frozen pointer proof for each producer.
func TestPointerAPIRecordTypes(t *testing.T) {
	for _, source := range []string{
		`var cell im.Cell[int];func f(){_=cell.LoadPointer()}`,
		`func f(){_=im.CopyView(im.ViewOf(1))}`,
		`func f(view im.SliceView[string]){_=view.AtPointer(0)}`,
	} {
		if got := checkSource(t, source); !strings.Contains(got, "concrete marked struct target") {
			t.Fatal(got)
		}
	}
}
