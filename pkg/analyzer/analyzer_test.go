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
 type Shared struct{}
 type Scoped struct{}
 type Owner[T any] struct{}
 func (o *Owner[T]) Load() *T { return nil }
 func (o *Owner[T]) Store(p *T) {}
 type UncheckedValue[T any] struct { value *T }
 func Trust[T any](value *T) UncheckedValue[T] { return UncheckedValue[T]{value: value} }
 func Unwrap[T any](value UncheckedValue[T]) *T { return value.value }
 type FrozenSlice[T any] struct { values []T }
 type FrozenMap[K comparable, V any] struct { values map[K]V }
 func FreezeSlice[T any](values []T) FrozenSlice[T] { return FrozenSlice[T]{} }
 func FreezeMap[K comparable, V any](values map[K]V) FrozenMap[K,V] { return FrozenMap[K,V]{} }
 func (values FrozenSlice[T]) Clone() []T { return nil }
 func (values FrozenMap[K,V]) Clone() map[K]V { return nil }
 func (values FrozenMap[K,V]) Get(key K)(V,bool) { var zero V;return zero,false }
 func (values FrozenSlice[T]) At(index int)T { var zero T;return zero }
 func (values FrozenSlice[T]) Len()int {return 0}
 func (values FrozenSlice[T]) All()func(func(int,T)bool) {return nil}
 func (values FrozenMap[K,V]) All()func(func(K,V)bool) {return nil}
`

const testRuntime = `package proc
 import ("context"; "github.com/lucasavila00/immutago/pkg/im")
 type Process[A any] struct { args A }
 func (p *Process[A]) Args() A { return p.args }
 func (p *Process[A]) bind(args A) { p.args = args }
 func (p *Process[A]) Supervisor() *Supervisor { return nil }
 type None struct { im.Shared }
 type Supervisor struct{}
 func NewRoot() *Supervisor { return nil }
 func StartSupervisor(parent *Supervisor) *Supervisor { return nil }
 func (s *Supervisor) Stop() <-chan struct{} { return nil }
 type PID[T any] struct { value *T }
 func (p PID[T]) Proc() *T { return p.value }
 func Reserve[T any]() PID[T] { return PID[T]{} }
 func Start[T, A any, P interface{ *T; Run(context.Context); bind(A) }](parent *Supervisor, args A) PID[T] {
  return PID[T]{}
 }
 func StartReserved[T, A any, P interface{ *T; Run(context.Context); bind(A) }](
  parent *Supervisor, pid PID[T], args A) {}
 type UncheckedDependency[T any] struct { value T }
 func Trust[T any](v T) UncheckedDependency[T] { return UncheckedDependency[T]{value: v} }
 func (d UncheckedDependency[T]) Value() T { return d.value }
`

// testImporter gives compiler tests the library's type identities without loading external packages.
type testImporter struct {
	library, runtime *types.Package
	standard         types.Importer
}

// Import resolves the sealed library and standard packages used in test programs.
func (i testImporter) Import(path string) (*types.Package, error) {
	if path == "unsafe" {
		return types.Unsafe, nil
	}
	if path == checker.LibraryPath {
		return i.library, nil
	}
	if path == checker.RuntimePath {
		return i.runtime, nil
	}
	return i.standard.Import(path)
}

// checkSource type-checks a complete Go package, then runs the analyzer over its compiler data.
func checkSource(t *testing.T, declarations string) string {
	t.Helper()
	pass := sourcePass(t, declarations)
	var reports []string
	pass.Report = func(d analysis.Diagnostic) { reports = append(reports, d.Message) }
	if _, err := run(pass); err != nil {
		t.Fatal(err)
	}
	return strings.Join(reports, "\n")
}

// sourcePass prepares compiler data without including setup in analyzer benchmarks.
func sourcePass(t testing.TB, declarations string) *analysis.Pass {
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
	runtimeFile, err := parser.ParseFile(positions, "proc.go", testRuntime, 0)
	if err != nil {
		t.Fatal(err)
	}
	standard := importer.Default()
	runtimeConfig := types.Config{Importer: testImporter{library, nil, standard}}
	procPackage, err := runtimeConfig.Check(checker.RuntimePath, positions, []*ast.File{runtimeFile}, nil)
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
	config := types.Config{Importer: testImporter{library, procPackage, standard}}
	pkg, err := config.Check("sample", positions, []*ast.File{file}, info)
	if err != nil {
		t.Fatalf("test source does not compile: %v", err)
	}
	return &analysis.Pass{
		TypesSizes: types.SizesFor("gc", runtime.GOARCH),
		Analyzer:   Analyzer, Fset: positions, Files: []*ast.File{file}, Pkg: pkg, TypesInfo: info,
	}
}

// TestValidTypes covers recursive proofs, aliases, and ordinary mutable scalar locals.
func TestValidTypes(t *testing.T) {
	cases := []string{
		`type Value struct { im.Shared; Flag bool; Text string; Number complex128; Array [4]int }`,
		`type Node struct { im.Shared; Children im.FrozenSlice[Node] }`,
		`type A struct { im.Shared; B im.FrozenSlice[B] }
   type B struct { im.Shared; A im.FrozenMap[string,A] }`,
		`type Alias = im.Shared
   type Value struct { Alias; Text string }
   type Copy Value`,
		`type Value struct { im.Shared; Items im.FrozenSlice[im.FrozenMap[string,int]] }`,
		`type Value struct { im.Shared; X int }
   func replace() { v := Value{X: 1}; v = Value{X: 2}; _ = v }`,
		`type Value struct { im.Shared; A [2]int }
   func copyArray(v Value) { a := v.A; a[0] = 1 }`,
		`type Value struct { im.Shared; S string }
   func text(v Value) string { return v.S[:] }`,
		`type V struct { im.Shared }; func f(x *V) {}`,
		`var _ = im.FreezeSlice([]int{1, 2})
   var _ = im.FreezeMap(map[string]int{"a": 1})`,
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
		{"named array", `type Count [2]int; type V struct { im.Shared; X Count }`, "named array"},
		{"pointer", `type V struct { im.Shared; X *int }`, "concrete marked struct target"},
		{"slice", `type V struct { im.Shared; X []int }`, "not provably immutable"},
		{"interface", `type V struct { im.Shared; X any }`, "not provably immutable"},
		{"function", `type V struct { im.Shared; X func() }`, "not provably immutable"},
		{"channel", `type V struct { im.Shared; X chan int }`, "not provably immutable"},
		{"map", `type V struct { im.Shared; X map[int]int }`, "not provably immutable"},
		{"unmarked", `type V struct { im.Shared; X struct{ N int } }`, "does not embed"},
		{"named marker", `type V struct { M im.Shared }`, "must be embedded by value"},
		{"pointer marker", `type V struct { *im.Shared }`, "must be embedded by value"},
		{"recursive failure", `type V struct { im.Shared; Next im.FrozenSlice[V]; Bad []int }`, "Bad"},
		{"generic", `type V[T any] struct { im.Shared; X T }`, "type parameters are unsupported"},
		{"mixed terms", `type V[T interface{ ~int | ~[]int }] struct { im.Shared; X T }`, "type parameters are unsupported"},
		{"repeated failed bound", `type Bad struct{ X int }
         type V[T interface{ Bad; Bad }] struct { im.Shared; X T }`, "type parameters are unsupported"},
		{"failed recursive alternatives", `type Bad struct { im.Shared; Next im.FrozenSlice[Bad]; X []int }
         type V[T interface{ Bad; Bad }] struct { im.Shared; X T }`, "type parameters are unsupported"},
		{"bad argument", `var _ im.FrozenSlice[*int]`, "concrete marked struct target"},
		{"bad inference", `var _ = im.FreezeSlice([]*int{new(int)})`, "concrete marked struct target"},
		{"bad key", `var _ im.FrozenMap[*int,int]`, "concrete marked struct target"},
		{"unmarked owner", `type V struct { im.Shared }; type W struct { V V }`, "unmarked struct"},
		{"unmarked slice", `type V struct { im.Shared }; type W struct { V []V }`, "unmarked struct"},
		{"unmarked map", `type V struct { im.Shared }; type W struct { V map[int]V }`, "unmarked struct"},
		{"unmarked array", `type V struct { im.Shared }; type W struct { V [2]V }`, "unmarked struct"},
		{"unmarked collection", `type W struct { V im.FrozenSlice[int] }`, "unmarked struct"},
		{"constrained owner", `type V struct { im.Shared }; type W[T V] struct { X T }`, "unmarked struct"},
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
