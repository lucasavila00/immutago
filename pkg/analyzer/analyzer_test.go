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
	"github.com/lucasavila00/immutago/internal/projectiongen"
	"golang.org/x/tools/go/analysis"
)

const testLibrary = `package im
 type Frozen struct{}
 func(Frozen) immutagoFrozen() {}
 type FrozenInterface interface { immutagoFrozen() }
 type Local struct{}
 type Copied struct{}
 type Cell[T any] struct{}
 func (o *Cell[T]) Load() Projection[T] { return Projection[T]{} }
 func (o *Cell[T]) LoadPointer() Projection[*T] { return Projection[*T]{} }
 func (o *Cell[T]) Store(value T) {}
 type Projection[T any] struct { value T }
 func ProjectionOf[T any](value T) Projection[T] { return Projection[T]{value:value} }
 func CopyRecord[T any](value Projection[T]) Projection[*T] { return Projection[*T]{} }
 func (value Projection[T]) Unsafe() T { return value.value }
 func Value[T any](value Projection[T]) T { return value.value }
 func IsNil[T any](value Projection[*T]) bool { return value.value == nil }
 type StringProjection struct { value string }
 func ProjectString[S ~string](value S) StringProjection { return StringProjection{} }
 func StringOfFrozen(value Projection[FrozenString]) StringProjection { return StringProjection{} }
 func StringOf[S ~string](value Projection[S]) StringProjection { return StringProjection{} }
 func (s StringProjection) Len() int { return 0 }
 func (s StringProjection) At(index int) byte { return 0 }
 func (s StringProjection) Clone() string { return "" }
 func (s StringProjection) Equal(other StringProjection) bool { return false }
 func (s StringProjection) Is(other string) bool { return false }
 func (s StringProjection) Unsafe() string { return s.value }
 type SliceProjection[E any] struct { values []E }
 func SliceOf[E any](value Projection[FrozenSlice[E]]) SliceProjection[E] { return SliceProjection[E]{} }
 func (s SliceProjection[E]) Len() int { return 0 }
 func (s SliceProjection[E]) At(index int) Projection[E] { return Projection[E]{} }
 func (s SliceProjection[E]) AtPointer(index int) Projection[*E] { return Projection[*E]{} }
 func (s SliceProjection[E]) All()func(func(int,Projection[E])bool) { return nil }
 func (s SliceProjection[E]) Clone() []E { return nil }
 type MapProjection[K comparable,V any] struct { values map[K]V }
 func MapOf[K comparable,V any](value Projection[FrozenMap[K,V]]) MapProjection[K,V] {
 return MapProjection[K,V]{} }
 func (m MapProjection[K,V]) Len() int { return 0 }
 func (m MapProjection[K,V]) Get(key K)(Projection[V],bool) { return Projection[V]{},false }
 func (m MapProjection[K,V]) All()func(func(Projection[K],Projection[V])bool) { return nil }
 func (m MapProjection[K,V]) Clone() map[K]V { return nil }
 type UncheckedValue[T any] struct { value *T }
 func Trust[T any](value *T) UncheckedValue[T] { return UncheckedValue[T]{value: value} }
 func Unwrap[T any](value UncheckedValue[T]) *T { return value.value }
 type FrozenSlice[T any] struct { values []T }
 type FrozenMap[K comparable, V any] struct { values map[K]V }
 type FrozenString struct { value string }
 func (value FrozenString) Projection() StringProjection { return StringProjection{} }
 func (value FrozenSlice[T]) Projection() SliceProjection[T] { return SliceProjection[T]{} }
 func (value FrozenMap[K,V]) Projection() MapProjection[K,V] { return MapProjection[K,V]{} }
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
 func (p Ref[T]) Borrow() *T { return p.value }
 func Start[T, A any, P interface{ *T; Run(context.Context); Init() error; bind(A) }](
  parent *Group, args A) (Ref[T], error) {
  return Ref[T]{}, nil
 }
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
	source := projectiongen.Directive + "\npackage sample\nimport \"" + checker.LibraryPath + "\"\n" + declarations
	file, err := parser.ParseFile(positions, "sample.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := sourceTypeInfo()
	config := types.Config{Importer: testImporter{library, servicePackage, standard}, IgnoreFuncBodies: true}
	files := []*ast.File{file}
	pkg, err := config.Check("sample", positions, files, info)
	if err != nil {
		t.Fatalf("test source does not compile: %v", err)
	}
	generated, generationErr := projectiongen.GenerateTypes(pkg, positions)
	if generationErr == nil {
		output, parseErr := parser.ParseFile(positions, projectiongen.Filename, generated, parser.ParseComments)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		files = append(files, output)
	}
	config.IgnoreFuncBodies = false
	info = sourceTypeInfo()
	pkg, err = config.Check("sample", positions, files, info)
	if err != nil {
		t.Fatalf("generated test source does not compile: %v", err)
	}
	return &analysis.Pass{
		TypesSizes: types.SizesFor("gc", runtime.GOARCH),
		Analyzer:   Analyzer, Fset: positions, Files: files, Pkg: pkg, TypesInfo: info,
		ReadFile: func(name string) ([]byte, error) {
			if filepath.Base(name) == projectiongen.Filename {
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
		{"pointer", `type V struct { im.Frozen; X *int }`, "concrete Frozen record target"},
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
		{"bad argument", `var _ im.FrozenSlice[*int]`, "concrete Frozen record target"},
		{"bad inference", `var _ = im.FreezeSlice([]*int{new(int)})`, "concrete Frozen record target"},
		{"bad key", `var _ im.FrozenMap[*int,int]`, "concrete Frozen record target"},
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
