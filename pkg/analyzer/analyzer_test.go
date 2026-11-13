package analyzer

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"runtime"
	"strings"
	"testing"

	"github.com/lucasavila00/immutago/internal/checker"
	"golang.org/x/tools/go/analysis"
)

const testLibrary = `package im
 type Marker struct{}
 type Slice[T any] struct { values []T }
 type Map[K comparable, V any] struct { values map[K]V }
 func NewSlice[T any](values ...T) Slice[T] { return Slice[T]{} }
 func NewMap[K comparable, V any](values map[K]V) Map[K,V] { return Map[K,V]{} }
`

// testImporter gives compiler tests the library's type identities without loading external packages.
type testImporter struct{ library *types.Package }

// Import resolves the sealed library and standard packages used in test programs.
func (i testImporter) Import(path string) (*types.Package, error) {
	if path == "unsafe" {
		return types.Unsafe, nil
	}
	if path == checker.LibraryPath {
		return i.library, nil
	}
	return importer.Default().Import(path)
}

// checkSource type-checks a complete Go package, then runs the analyzer over its compiler data.
func checkSource(t *testing.T, declarations string) string {
	t.Helper()
	positions := token.NewFileSet()
	libraryFile, err := parser.ParseFile(positions, "im.go", testLibrary, 0)
	if err != nil {
		t.Fatal(err)
	}
	library, err := new(types.Config).Check(checker.LibraryPath, positions, []*ast.File{libraryFile}, nil)
	if err != nil {
		t.Fatal(err)
	}
	source := "package sample\nimport \"" + checker.LibraryPath + "\"\n" + declarations
	file, err := parser.ParseFile(positions, "sample.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Instances:  make(map[*ast.Ident]types.Instance),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
		Implicits:  make(map[ast.Node]types.Object),
	}
	config := types.Config{Importer: testImporter{library}}
	pkg, err := config.Check("sample", positions, []*ast.File{file}, info)
	if err != nil {
		t.Fatalf("test source does not compile: %v", err)
	}
	var reports []string
	pass := &analysis.Pass{
		TypesSizes: types.SizesFor("gc", runtime.GOARCH),
		Analyzer:   Analyzer, Fset: positions, Files: []*ast.File{file}, Pkg: pkg, TypesInfo: info,
		Report: func(d analysis.Diagnostic) { reports = append(reports, d.Message) },
	}
	if _, err := run(pass); err != nil {
		t.Fatal(err)
	}
	return strings.Join(reports, "\n")
}

// TestValidTypes covers recursive proofs, aliases, generic type sets, and ordinary mutable locals.
func TestValidTypes(t *testing.T) {
	cases := []string{
		`type Value struct { im.Marker; Flag bool; Text string; Number complex128; Array [4]int }`,
		`type Node struct { im.Marker; Children im.Slice[Node] }`,
		`type A struct { im.Marker; B im.Slice[B] }
   type B struct { im.Marker; A im.Map[string,A] }`,
		`type Alias = im.Marker
   type Value struct { Alias; Text string }
   type Copy Value`,
		`type Scalar interface { ~int | ~string }
   type Value[T Scalar] struct { im.Marker; Item T }
   var _ Value[int]`,
		`type Scalar interface { ~int | ~string }
   type Smaller interface { Scalar; ~int }
   type Value[T Smaller] struct { im.Marker; Item T }`,
		`type Value struct { im.Marker; Items im.Slice[im.Map[string,int]] }`,
		`type Value struct { im.Marker; X int }
   func replace() { v := Value{X: 1}; v = Value{X: 2}; _ = v }`,
		`type Value struct { im.Marker; A [2]int }
   func copyArray(v Value) { a := v.A; a[0] = 1 }`,
		`type Value struct { im.Marker; S string }
   func text(v Value) string { return v.S[:] }`,
		`type V struct { im.Marker }; func f[T V](x *T) {}`,
		`var _ = im.NewSlice(1, 2)
   var _ = im.NewMap(map[string]int{"a": 1})`,
	}
	for _, source := range cases {
		t.Run(source[:min(len(source), 50)], func(t *testing.T) {
			if reports := checkSource(t, source); reports != "" {
				t.Fatalf("unexpected diagnostics:\n%s", reports)
			}
		})
	}
}

// TestRejectedTypes covers failed recursive obligations, marker misuse, and immutable ownership boundaries.
func TestRejectedTypes(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"named array", `type Count [2]int; type V struct { im.Marker; X Count }`, "named array"},
		{"pointer", `type V struct { im.Marker; X *int }`, "not provably immutable"},
		{"slice", `type V struct { im.Marker; X []int }`, "not provably immutable"},
		{"interface", `type V struct { im.Marker; X any }`, "not provably immutable"},
		{"function", `type V struct { im.Marker; X func() }`, "not provably immutable"},
		{"channel", `type V struct { im.Marker; X chan int }`, "not provably immutable"},
		{"map", `type V struct { im.Marker; X map[int]int }`, "not provably immutable"},
		{"unmarked", `type V struct { im.Marker; X struct{ N int } }`, "does not embed"},
		{"named marker", `type V struct { M im.Marker }`, "must be embedded by value"},
		{"pointer marker", `type V struct { *im.Marker }`, "must be embedded by value"},
		{"recursive failure", `type V struct { im.Marker; Next im.Slice[V]; Bad []int }`, "Bad"},
		{"generic", `type V[T any] struct { im.Marker; X T }`, "constraint does not prove"},
		{"mixed terms", `type V[T interface{ ~int | ~[]int }] struct { im.Marker; X T }`, "constraint"},
		{"repeated failed bound", `type Bad struct{ X int }
         type V[T interface{ Bad; Bad }] struct { im.Marker; X T }`, "constraint"},
		{"failed recursive alternatives", `type Bad struct { im.Marker; Next im.Slice[Bad]; X []int }
         type V[T interface{ Bad; Bad }] struct { im.Marker; X T }`, "constraint"},
		{"bad argument", `var _ im.Slice[*int]`, "not provably immutable"},
		{"bad inference", `var _ = im.NewSlice(new(int))`, "not provably immutable"},
		{"bad key", `var _ im.Map[*int,int]`, "not provably immutable"},
		{"unmarked owner", `type V struct { im.Marker }; type W struct { V V }`, "unmarked struct"},
		{"unmarked slice", `type V struct { im.Marker }; type W struct { V []V }`, "unmarked struct"},
		{"unmarked map", `type V struct { im.Marker }; type W struct { V map[int]V }`, "unmarked struct"},
		{"unmarked array", `type V struct { im.Marker }; type W struct { V [2]V }`, "unmarked struct"},
		{"unmarked collection", `type W struct { V im.Slice[int] }`, "unmarked struct"},
		{"constrained owner", `type V struct { im.Marker }; type W[T V] struct { X T }`, "unmarked struct"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if reports := checkSource(t, tc.source); !strings.Contains(reports, tc.want) {
				t.Fatalf("diagnostics = %q; want %q", reports, tc.want)
			}
		})
	}
}

// TestAnalyzerContract checks the analyzer's registration contract for go vet and analysis drivers.
func TestAnalyzerContract(t *testing.T) {
	if err := analysis.Validate([]*analysis.Analyzer{Analyzer}); err != nil {
		t.Fatal(err)
	}
}
