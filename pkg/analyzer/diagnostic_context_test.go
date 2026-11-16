package analyzer

import (
	"fmt"
	"strings"
	"testing"
)

// TestValueDiagnosticContext keeps inferred and explicit type argument repairs equal to the call repair.
func TestValueDiagnosticContext(t *testing.T) {
	cases := []struct{ declaration, payload, repair string }{
		{"", "string", "use im.StringOf(projection)"},
		{"type Text string", "Text", "use im.StringOf(projection)"},
		{"type Text = string", "Text", "use im.StringOf(projection)"},
		{"", "im.FrozenString", "use im.StringOfFrozen(projection)"},
		{"type Text = im.FrozenString", "Text", "use im.StringOfFrozen(projection)"},
		{"type Settings struct{im.Frozen; Limit int}", "Settings", "im.CopyRecord(projection)"},
		{"type Settings struct{im.Frozen; Limit int}; type Alias = Settings", "Alias", "im.CopyRecord(projection)"},
		{"", "im.FrozenSlice[string]", "use im.SliceOf(projection)"},
		{"", "im.FrozenMap[string,int]", "use im.MapOf(projection)"},
		{"", "[2]string", "use projection.Unsafe() for shared access"},
		{"type Settings struct{im.Frozen; Limit int}", "*Settings", "use projection.Unsafe() for shared access"},
		{"type Reader interface{im.FrozenInterface}", "Reader", "use projection.Unsafe() for shared access"},
	}
	for _, test := range cases {
		for _, typeArguments := range []string{"", "[" + test.payload + "]"} {
			source := fmt.Sprintf("%s\nfunc f(p im.Projection[%s]){_=im.Value%s(p)}",
				test.declaration, test.payload, typeArguments)
			got := checkSource(t, source)
			if strings.Count(got, test.repair) != 2 || !strings.Contains(got, "immutable type argument:") ||
				!strings.Contains(got, "unsupported immutable operation:") {
				t.Fatalf("payload %s, arguments %s: %s", test.payload, typeArguments, got)
			}
		}
	}
}

// TestAssertionDiagnosticContext uses a concrete recovery for unproved source interfaces.
func TestAssertionDiagnosticContext(t *testing.T) {
	for _, source := range []string{"any", "interface{Score()int}", "Box"} {
		got := checkSource(t, `type Box = any;type Reader interface{im.FrozenInterface;Score()int};
func f(boxed `+source+`){_=boxed.(Reader)}`)
		if !strings.Contains(got, "assert the concrete Frozen record value or pointer first") {
			t.Fatal(got)
		}
	}
	got := checkSource(t, `func f(boxed any){_=boxed.(im.FrozenInterface)}`)
	if !strings.Contains(got, "embed im.FrozenInterface in a named interface") ||
		strings.Contains(got, "assert the concrete Frozen record value") {
		t.Fatal(got)
	}
}

// TestCopiedFieldDiagnosticContext prevents a repair cycle between text and Local field types.
func TestCopiedFieldDiagnosticContext(t *testing.T) {
	for _, field := range []string{"string", "im.FrozenString", "im.StringProjection", "Reply", "[2]Reply", "[0]string"} {
		got := checkSource(t, `type Reply struct{im.Local;Name im.StringProjection};
type Score struct{im.Copied;Label `+field+`};type Outer struct{im.Copied;Scores [2]Score}`)
		if !strings.Contains(got, "Label: ") || !strings.Contains(got, "is not a Copied value") ||
			!strings.Contains(got, "replace its im.Copied marker with im.Local") ||
			strings.Contains(got, "Frozen storage cannot contain Local values") {
			t.Fatal(got)
		}
	}
	got := checkSource(t, `type Counts [2]int;type Score struct{im.Copied;Counts Counts};
type Outer struct{im.Copied;Scores [2]Score}`)
	if !strings.Contains(got, "use an unnamed array type") || strings.Contains(got, "if a Copied record") {
		t.Fatal(got)
	}
}

// TestMarkerFieldDiagnosticContext retains annotation repairs through marked owners and arrays.
func TestMarkerFieldDiagnosticContext(t *testing.T) {
	for _, owner := range []string{"im.Frozen", "im.Copied", "im.Local"} {
		for _, marker := range []string{"im.Frozen", "im.Local", "im.Copied", "im.FrozenInterface"} {
			for _, field := range []string{"Marker", "[0]Marker", "[2][1]Marker"} {
				got := checkTypeSource(t, "type Marker="+marker+";type Record struct{"+owner+";Data "+field+"};"+
					"type Outer struct{"+owner+";Records [2]Record}")
				if !(strings.Contains(got, "immutable markers are annotations, not data") ||
					strings.Contains(got, "immutable markers must be embedded by value")) {
					t.Fatal(got)
				}
				if marker == "im.FrozenInterface" {
					repair := map[string]string{
						"im.Frozen": "use that interface as the payload",
						"im.Local":  "use its generated projection as the read result",
						"im.Copied": "replace its im.Copied marker with im.Local",
					}[owner]
					if !strings.Contains(got, repair) {
						t.Fatal(got)
					}
				}
			}
		}
	}
}

// TestCloneDiagnosticContext reports the Copied rule for each projection argument.
func TestCloneDiagnosticContext(t *testing.T) {
	for _, payload := range []string{"Settings", "Alias", "[2]Settings", "[0][2]Alias", "*Settings", "Scorer"} {
		for _, projection := range []string{
			"im.SliceProjection[%s]", "im.MapProjection[int,%s]", "im.MapProjection[%s,int]",
		} {
			source := `type Settings struct{im.Frozen;Limit int};type Alias=Settings;
type Scorer interface{im.FrozenInterface;Score()int};func f(p ` + fmt.Sprintf(projection, payload) + `){_=p.Clone()}`
			got := checkSource(t, source)
			if !strings.Contains(got, "is not a Copied value\n  source: p.Clone()") ||
				!strings.Contains(got, "fix: iterate the projection with All; "+
					"read or clone each element into a new native collection") ||
				strings.Contains(got, "native slices of immutable storage") ||
				strings.Contains(got, "native maps of immutable storage") {
				t.Fatalf("%s: %s", source, got)
			}
		}
	}
}

// TestCloneReceiverDiagnosticContext checks the receiver before the result payload.
func TestCloneReceiverDiagnosticContext(t *testing.T) {
	for _, source := range []string{
		`type Holder struct{im.Local;im.SliceProjection[Settings]};func f(p Holder){_=p.Clone()}`,
		`func f(p im.SliceProjection[Settings]){_=im.SliceProjection[Settings].Clone(p)}`,
	} {
		got := checkSource(t, `type Settings struct{im.Frozen;Limit int};`+source)
		if !strings.Contains(got, "collection and projection calls require a supported receiver") ||
			!strings.Contains(got, "for an embedded collection or projection, use holder.Field.Method(...)") ||
			strings.Contains(got, "is not a Copied value") {
			t.Fatal(got)
		}
	}
}

// TestNativeLocalFieldRepair requires Frozen elements before replacing a native container.
func TestNativeLocalFieldRepair(t *testing.T) {
	for _, field := range []string{
		"[]im.StringProjection", "map[string]im.StringProjection", "[2][]im.StringProjection",
		"map[string][0]im.StringProjection", "[]Reply", "map[string]Reply",
	} {
		got := checkTypeSource(t, "type Reply struct{im.Local;Name im.StringProjection};"+
			"type Settings struct{im.Frozen;Items "+field+"}")
		if !strings.Contains(got, "immutable field Items") || !strings.Contains(got, "Frozen type") ||
			!strings.Contains(got, "keep Local") {
			t.Fatal(got)
		}
	}
}

// TestReadResultProjectionRepair retains the projection route for existing Local fields and service results.
func TestReadResultProjectionRepair(t *testing.T) {
	for _, payload := range []string{"Data", "*Data", "im.FrozenSlice[int]", "im.FrozenMap[int,int]", "Reader"} {
		got := checkSource(t, `import "github.com/lucasavila00/immutago/pkg/supervise"
type Data struct{im.Frozen;N int};type Reader interface{im.FrozenInterface};
type Reply struct{im.Local;Value `+payload+`};type Worker struct{supervise.Service[supervise.NoArgs]};
func(w *Worker)Read()`+payload+`{var result `+payload+`;return result}`)
		if strings.Count(got, "for Local fields or results, use matching projections") < 2 {
			t.Fatal(got)
		}
	}
}
