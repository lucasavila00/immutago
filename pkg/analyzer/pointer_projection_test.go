package analyzer

import (
	"strings"
	"testing"
)

const pointerProjectionTypes = `type Child struct{im.Frozen;Name string;Number int}
type Root struct{im.Frozen;Child Child;Children [2]Child;Pointer *Child}
var cell im.Cell[Root]
`

// TestPointerProjectionForms checks generated record addresses and explicit root copies.
func TestPointerProjectionForms(t *testing.T) {
	for _, source := range []string{
		`func f(){cell.Store(Root{});projection:=NewRootProjection(cell.LoadPointer())
if !projection.IsNil(){_=projection.Child().Name().Len()}}`,
		`func f(projection im.Projection[*Root],index int){_=NewRootProjection(projection).Children().At(index).Number()}`,
		`func f(){frozen:=im.FreezeSlice([]Child{{}});values:=frozen.Projection()
_=NewChildProjection(values.AtPointer(0)).Number()}`,
		`func f(){frozen:=im.FreezeMap(map[string]Child{"name":{}});values:=frozen.Projection()
value,ok:=values.Get("name");_=ok;pointer:=im.CopyRecord(value);_=NewChildProjection(pointer).Name().Len()}`,
	} {
		if got := checkSource(t, pointerProjectionTypes+source); got != "" {
			t.Fatal(got)
		}
	}
}

// TestPointerProjectionBoundaries rejects handwritten interior pointers, including rewrapped ones.
func TestPointerProjectionBoundaries(t *testing.T) {
	for _, operation := range []string{
		`_ = &projection.Unchecked().Child`,
		`_ = im.ProjectionOf(&projection.Unchecked().Child)`,
		`_ = &projection.Unchecked().Children[0]`,
		`_ = &projection.Unchecked().Pointer`,
		`_ = &projection.Unchecked().Child.Number`,
		`_ = &*projection.Unchecked()`,
	} {
		source := pointerProjectionTypes + `func f(projection im.Projection[*Root]){` + operation + `}`
		if got := checkSource(t, source); got == "" {
			t.Fatal(got)
		}
	}
}

// TestPointerAPIRecordTypes uses the same Frozen pointer proof for each producer.
func TestPointerAPIRecordTypes(t *testing.T) {
	for _, source := range []string{
		`var cell im.Cell[int];func f(){_=cell.LoadPointer()}`,
		`func f(){_=im.CopyRecord(im.ProjectionOf(1))}`,
		`func f(projection im.SliceProjection[string]){_=projection.AtPointer(0)}`,
	} {
		if got := checkSource(t, source); !strings.Contains(got, "concrete Frozen record target") {
			t.Fatal(got)
		}
	}
}
