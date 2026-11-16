package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
)

// TestCopiedDiagnosticPaths keeps the same repair for nested category failures.
func TestCopiedDiagnosticPaths(t *testing.T) {
	for _, prefix := range []string{"", "array element: ", "Result: array element: "} {
		reason := prefix + "string is not a Copied value"
		for _, fix := range []string{subsetFix(reason), structuralFix(types.Typ[types.Int], reason)} {
			if fix != "copy scalar fields, or use a local projection" {
				t.Fatal(fix)
			}
		}
	}
}

// TestStructuralDiagnosticContext checks the owner, failed field type, and supported repair.
func TestStructuralDiagnosticContext(t *testing.T) {
	cases := []struct {
		source string
		want   []string
	}{
		{`type Value struct{im.Frozen; Items []string}`, []string{
			"immutable field Items", "owner: sample.Value; field type: []string",
			"fix: replace the native slice with im.FrozenSlice[T]",
		}},
		{`type Child struct{im.Frozen}; type Parent struct{Child Child}`, []string{
			"unmarked struct", "owner: sample.Parent; field type: sample.Child",
			"fix: store *sample.Child",
		}},
		{`type Parent struct{Text im.StringProjection}`, []string{
			"unmarked struct field Text contains protected storage", "store the matching im.FrozenString value",
		}},
		{`type Parent struct{Values im.SliceProjection[int]}`, []string{
			"store the matching im.FrozenSlice value",
		}},
		{`type Parent struct{Values im.MapProjection[int,int]}`, []string{
			"store the matching im.FrozenMap value",
		}},
		{`type Reply struct{im.Local;N int};type Parent struct{Reply Reply}`, []string{
			"store Frozen pointers or handles; keep Local data in local bindings",
		}},
		{`var _ = im.FreezeSlice([]*int{new(int)})`, []string{
			"immutable type argument", "type: *int", "fix: replace the pointer field or argument with an owned immutable value",
		}},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			got := checkTypeSource(t, tc.source)
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Fatalf("missing %q in %s", want, got)
				}
			}
		})
	}
}

// TestErrorInterfaceDiagnosticRepair distinguishes aliases from new error interface types.
func TestErrorInterfaceDiagnosticRepair(t *testing.T) {
	prefix := `import "github.com/lucasavila00/immutago/pkg/supervise"
type Worker struct{supervise.Service[supervise.NoArgs]};type Count struct{im.Copied;N int};`
	for _, declaration := range []string{
		`type Failure error`,
		`type Failure interface{error;Code()int}`,
		`type error interface{Error()string};type Failure=error`,
	} {
		got := checkSource(t, prefix+declaration+`;func(w *Worker)Retry()Failure{return nil}`)
		if !strings.Contains(got, "must return a read result or error") ||
			!strings.Contains(got, "declare the result as Go's predeclared error") ||
			!strings.Contains(got, "rename any declaration that shadows error") {
			t.Fatal(got)
		}
	}
	got := checkSource(t, prefix+`type Failure = error;func(w *Worker)Retry()(int,Failure){return 1,nil}`)
	if got != "" {
		t.Fatal(got)
	}
}

// TestProofDiagnosticCategories names the rule at the field or type boundary.
func TestProofDiagnosticCategories(t *testing.T) {
	cases := []struct {
		source string
		want   string
	}{
		{`type Label struct{im.Copied;Text string}`, "Copied record: Text: string is not a Copied value"},
		{`type Reply struct{im.Local;Text string}`,
			"immutable field Text requires a read result: string is not a Copied value"},
		{`func f(value im.StringProjection){frozen:=im.FreezeSlice([]im.StringProjection{value});_=frozen}`,
			"Frozen collection: Frozen storage cannot contain Local values"},
		{`type Point struct{im.Frozen};var points []Point`, "native builder points requires a fresh local"},
		{`func f(value im.Projection[*int]){_=value}`,
			"local projection: projection payload 1: stored pointers require a concrete Frozen record target"},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			got := checkSource(t, tc.source)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("missing %q in %s", tc.want, got)
			}
		})
	}
}

// TestProjectionDiagnosticRepairs names value reads for each rejected pointer API.
func TestProjectionDiagnosticRepairs(t *testing.T) {
	cases := []struct {
		source string
		want   string
	}{
		{`func f(value im.Projection[int]){_=im.CopyRecord(value)}`, "im.Value(projection)"},
		{`var cell im.Cell[int];func f(){_=cell.LoadPointer()}`, "im.Value(cell.Load())"},
		{`func f(value im.SliceProjection[int]){_=value.AtPointer(0)}`, "im.Value(slice.At(index))"},
		{`type Reply struct{im.Local;Text string}`, "im.ProjectString(text)"},
		{`type Reply struct{im.Local;Text im.FrozenString}`, "text.Projection()"},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			got := checkSource(t, tc.source)
			if !strings.Contains(got, "fix: ") || !strings.Contains(got, tc.want) {
				t.Fatalf("missing repair %q in %s", tc.want, got)
			}
		})
	}
}

// TestServiceDiagnosticRepair uses the same read-result repair as a Local field.
func TestServiceDiagnosticRepair(t *testing.T) {
	got := checkSource(t, `import "github.com/lucasavila00/immutago/pkg/supervise"
type Worker struct{supervise.Service[supervise.NoArgs]}
func(w *Worker)Name()string{return ""}
func(w *Worker)Text()im.StringProjection{return im.ProjectString("")}`)
	for _, want := range []string{
		"service method Name must return a read result or error: string is not a Copied value",
		"fix: use im.StringProjection for a local string result; construct it with im.ProjectString(text)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
}

// TestDependencyWitnessSelection keeps the first cause without changing local errors or imported facts.
func TestDependencyWitnessSelection(t *testing.T) {
	failures := [...]proofFailure{
		{Location: "origin/a.go:1:1", Message: "rule A"},
		{Location: "origin/b.go:2:1", Message: "rule B"},
		{Location: "origin/a.go:1:1", Message: "rule A"},
	}
	before := failures
	original := invalidPackage{Relevant: true, Origin: "earlier", Failure: failures[0]}
	imports := map[string]invalidPackage{"wrapper": original}
	for range 10 {
		got := packageFailureFact("origin", false, imports, failures[:])
		if got.Relevant || got.Origin != "earlier" || got.Failure != failures[0] {
			t.Fatalf("first cause changed: %+v", got)
		}
		if failures != before || imports["wrapper"] != original {
			t.Fatal("selecting a witness changed an original error")
		}
	}
}

// TestFieldDiagnosticsUseOneDeclarationSite reports each failed field once across repeated type visits.
func TestFieldDiagnosticsUseOneDeclarationSite(t *testing.T) {
	got := checkTypeSource(t, `type Value struct{im.Frozen; Items []string; Pointer *int}; var _ Value`)
	for _, field := range []string{"Items", "Pointer"} {
		if count := strings.Count(got, "immutable field "+field+" requires a Frozen type:"); count != 1 {
			t.Fatalf("field %s reported %d times: %s", field, count, got)
		}
	}
}

// TestPointerConstructorAcceptsCompilerBuiltinSignature checks the fresh allocation boundary.
func TestPointerConstructorAcceptsCompilerBuiltinSignature(t *testing.T) {
	got := checkSource(t, `type Value struct{im.Frozen};func f(){_=new(Value)}`)
	if got != "" {
		t.Fatalf("incorrect constructor diagnostic: %s", got)
	}
}

// checkTypeSource tests type diagnostics directly, without the generation prerequisite.
func checkTypeSource(t *testing.T, source string) string {
	t.Helper()
	pass := sourcePass(t, source)
	var reports []string
	pass.Report = func(d analysis.Diagnostic) { reports = append(reports, d.Message) }
	c := &checks{pass: pass, reported: make(map[token.Pos]bool)}
	for _, file := range pass.Files {
		ast.Inspect(file, c.visit)
	}
	return strings.Join(reports, "\n")
}
