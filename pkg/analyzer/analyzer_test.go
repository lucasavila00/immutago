package analyzer

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/lucasavila00/immutago/internal/checker"
	"github.com/lucasavila00/immutago/internal/viewgen"
	"golang.org/x/tools/go/analysis"
)

const testLibrary = `package im
 type Frozen struct{}
 type Local struct{}
 type Cell[T any] struct{}
 func (o *Cell[T]) Load() View[T] { return View[T]{} }
 func (o *Cell[T]) LoadPointer() View[*T] { return View[*T]{} }
 func (o *Cell[T]) Shared() T { var zero T; return zero }
 func (o *Cell[T]) Store(value T) {}
 type View[T any] struct { value T }
 func ViewOf[T any](value T) View[T] { return View[T]{value:value} }
 func CopyView[T any](value View[T]) View[*T] { return View[*T]{} }
 func Select[T,F any](value View[T], project func(T)F) View[F] { return View[F]{} }
 func Copy[T any](value View[T]) T { return value.value }
 func IsNil[T any](value View[*T]) bool { return value.value == nil }
 type StringView struct { value string }
 func String(value View[FrozenString]) StringView { return StringView{} }
 func Text[S ~string](value View[S]) StringView { return StringView{} }
 func (s StringView) Len() int { return 0 }
 func (s StringView) At(index int) byte { return 0 }
 func (s StringView) Clone() string { return "" }
 func (s StringView) Equal(other StringView) bool { return false }
 func (s StringView) EqualString(other string) bool { return false }
 func (s StringView) Unsafe() string { return s.value }
 type SliceView[E any] struct { values []E }
 func Slice[E any](value View[FrozenSlice[E]]) SliceView[E] { return SliceView[E]{} }
 func (s SliceView[E]) Len() int { return 0 }
 func (s SliceView[E]) At(index int) View[E] { return View[E]{} }
 func (s SliceView[E]) AtPointer(index int) View[*E] { return View[*E]{} }
 func (s SliceView[E]) All()func(func(int,View[E])bool) { return nil }
 func (s SliceView[E]) Clone() []E { return nil }
 type MapView[K comparable,V any] struct { values map[K]V }
 func Map[K comparable,V any](value View[FrozenMap[K,V]]) MapView[K,V] { return MapView[K,V]{} }
 func (m MapView[K,V]) Len() int { return 0 }
 func (m MapView[K,V]) Get(key K)(View[V],bool) { return View[V]{},false }
 func (m MapView[K,V]) All()func(func(View[K],View[V])bool) { return nil }
 func (m MapView[K,V]) Clone() map[K]V { return nil }
 type UncheckedValue[T any] struct { value *T }
 func Trust[T any](value *T) UncheckedValue[T] { return UncheckedValue[T]{value: value} }
 func Unwrap[T any](value UncheckedValue[T]) *T { return value.value }
 type FrozenSlice[T any] struct { values []T }
 type FrozenMap[K comparable, V any] struct { values map[K]V }
 type FrozenString struct { value string }
 func FreezeString(value string) FrozenString { return FrozenString{} }
 func (value FrozenString) Len() int { return 0 }
 func (value FrozenString) At(index int) byte { return 0 }
 func (value FrozenString) Clone() string { return "" }
 func (value FrozenString) Equal(other FrozenString) bool { return false }
 func (value FrozenString) EqualString(other string) bool { return false }
 func (value FrozenString) Unsafe() string { return value.value }
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

const testRuntime = `package supervise
 import "context"
 type Service[A any] struct { args A }
 func (p *Service[A]) Init() error { return nil }
 func (p *Service[A]) Args() A { return p.args }
 func (p *Service[A]) bind(args A) { p.args = args }
 func (p *Service[A]) Children() *Group { return nil }
 type NoArgs struct{}
 type Group struct{}
 func NewRoot(func(any, []byte)) *Group { return nil }
 func StartGroup(parent *Group) *Group { return nil }
 func (s *Group) Stop() <-chan struct{} { return nil }
 type Ref[T any] struct { value *T }
 func (p Ref[T]) Current() *T { return p.value }
 func Reserve[T any]() Ref[T] { return Ref[T]{} }
 func Start[T, A any, P interface{ *T; Run(context.Context); Init() error; bind(A) }](
  parent *Group, args A) (Ref[T], error) {
  return Ref[T]{}, nil
 }
 func StartReserved[T, A any, P interface{ *T; Run(context.Context); Init() error; bind(A) }](
  parent *Group, ref Ref[T], args A) error { return nil }
 type ConcurrentSafe[T any] struct { value T }
 func AssumeConcurrentSafe[T any](v T) ConcurrentSafe[T] { return ConcurrentSafe[T]{value: v} }
 func (d ConcurrentSafe[T]) Value() T { return d.value }
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
	runtimeFile, err := parser.ParseFile(positions, "supervise.go", testRuntime, 0)
	if err != nil {
		t.Fatal(err)
	}
	standard := importer.Default()
	runtimeConfig := types.Config{Importer: testImporter{library, nil, standard}}
	servicePackage, err := runtimeConfig.Check(checker.RuntimePath, positions, []*ast.File{runtimeFile}, nil)
	if err != nil {
		t.Fatal(err)
	}
	source := viewgen.Directive + "\npackage sample\nimport \"" + checker.LibraryPath + "\"\n" + declarations
	file, err := parser.ParseFile(positions, "sample.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := sourceTypeInfo()
	config := types.Config{Importer: testImporter{library, servicePackage, standard}}
	files := []*ast.File{file}
	pkg, err := config.Check("sample", positions, files, info)
	if err != nil {
		t.Fatalf("test source does not compile: %v", err)
	}
	generated, generationErr := viewgen.GenerateTypes(pkg, positions)
	if generationErr == nil {
		output, parseErr := parser.ParseFile(positions, viewgen.Filename, generated, parser.ParseComments)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		files = append(files, output)
		info = sourceTypeInfo()
		pkg, err = config.Check("sample", positions, files, info)
		if err != nil {
			t.Fatalf("generated test source does not compile: %v", err)
		}
	}
	return &analysis.Pass{
		TypesSizes: types.SizesFor("gc", runtime.GOARCH),
		Analyzer:   Analyzer, Fset: positions, Files: files, Pkg: pkg, TypesInfo: info,
		ReadFile: func(name string) ([]byte, error) {
			if filepath.Base(name) == viewgen.Filename {
				return generated, nil
			}
			if filepath.Base(name) == "sample.go" {
				return []byte(source), nil
			}
			return nil, os.ErrNotExist
		},
	}
}

// sourceTypeInfo records the compiler facts used by the production analyzer.
func sourceTypeInfo() *types.Info {
	return &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object), Instances: make(map[*ast.Ident]types.Instance),
		Selections: make(map[*ast.SelectorExpr]*types.Selection), Scopes: make(map[ast.Node]*types.Scope),
		Implicits: make(map[ast.Node]types.Object),
	}
}

// TestValidTypes covers recursive proofs, aliases, and ordinary mutable scalar locals.
func TestValidTypes(t *testing.T) {
	cases := []string{
		`type Value struct { im.Frozen; Flag bool; Text string; Number complex128; Array [4]int }`,
		`type Node struct { im.Frozen; Children im.FrozenSlice[Node] }`,
		`type A struct { im.Frozen; B im.FrozenSlice[B] }
   type B struct { im.Frozen; A im.FrozenMap[string,A] }`,
		`type Alias = im.Frozen
   type Value struct { Alias; Text string }
   type Copy Value`,
		`type Value struct { im.Frozen; Items im.FrozenSlice[im.FrozenMap[string,int]] }`,
		`type Value struct { im.Frozen; X int }
   func replace() { v := Value{X: 1}; v = Value{X: 2}; _ = v }`,
		`type Value struct { im.Frozen; A [2]int }
   func copyArray(v Value) { a := v.A; a[0] = 1 }`,
		`type Value struct { im.Frozen; S string }
   func text(v Value) string { return v.S[:] }`,
		`type V struct { im.Frozen }; func f(x *V) {}`,
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
		{"named array", `type Count [2]int; type V struct { im.Frozen; X Count }`, "named array"},
		{"pointer", `type V struct { im.Frozen; X *int }`, "concrete marked struct target"},
		{"slice", `type V struct { im.Frozen; X []int }`, "not provably immutable"},
		{"interface", `type V struct { im.Frozen; X any }`, "not provably immutable"},
		{"function", `type V struct { im.Frozen; X func() }`, "not provably immutable"},
		{"channel", `type V struct { im.Frozen; X chan int }`, "not provably immutable"},
		{"map", `type V struct { im.Frozen; X map[int]int }`, "not provably immutable"},
		{"unmarked", `type V struct { im.Frozen; X struct{ N int } }`, "does not embed"},
		{"named marker", `type V struct { M im.Frozen }`, "must be embedded by value"},
		{"pointer marker", `type V struct { *im.Frozen }`, "must be embedded by value"},
		{"recursive failure", `type V struct { im.Frozen; Next im.FrozenSlice[V]; Bad []int }`, "Bad"},
		{"generic", `type V[T any] struct { im.Frozen; X T }`, "type parameters are unsupported"},
		{"mixed terms", `type V[T interface{ ~int | ~[]int }] struct { im.Frozen; X T }`, "type parameters are unsupported"},
		{"repeated failed bound", `type Bad struct{ X int }
         type V[T interface{ Bad; Bad }] struct { im.Frozen; X T }`, "type parameters are unsupported"},
		{"failed recursive alternatives", `type Bad struct { im.Frozen; Next im.FrozenSlice[Bad]; X []int }
         type V[T interface{ Bad; Bad }] struct { im.Frozen; X T }`, "type parameters are unsupported"},
		{"bad argument", `var _ im.FrozenSlice[*int]`, "concrete marked struct target"},
		{"bad inference", `var _ = im.FreezeSlice([]*int{new(int)})`, "concrete marked struct target"},
		{"bad key", `var _ im.FrozenMap[*int,int]`, "concrete marked struct target"},
		{"unmarked owner", `type V struct { im.Frozen }; type W struct { V V }`, "unmarked struct"},
		{"unmarked slice", `type V struct { im.Frozen }; type W struct { V []V }`, "unmarked struct"},
		{"unmarked map", `type V struct { im.Frozen }; type W struct { V map[int]V }`, "unmarked struct"},
		{"unmarked array", `type V struct { im.Frozen }; type W struct { V [2]V }`, "unmarked struct"},
		{"unmarked collection", `type W struct { V im.FrozenSlice[int] }`, "unmarked struct"},
		{"constrained owner", `type V struct { im.Frozen }; type W[T V] struct { X T }`, "unmarked struct"},
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
