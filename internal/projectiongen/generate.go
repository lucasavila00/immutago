// Package projectiongen generates typed readers for Frozen records and interfaces.
package projectiongen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/lucasavila00/immutago/internal/checker"
	"golang.org/x/tools/go/packages"
)

// Filename is the generated source file in each package.
const Filename = "zz_immutago.gen.go"

// Directive is the required package generation command.
const Directive = "//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen"

const imPath = "github.com/lucasavila00/immutago/pkg/im"

// Generate reads one package and generates its Frozen record and interface readers.
// Existing generated declarations are excluded, so changed fields can regenerate.
// A package without Frozen declarations returns no output.
func Generate(dir string) ([]byte, error) {
	config := &packages.Config{
		Dir: dir,
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax |
			packages.NeedTypesInfo | packages.NeedImports | packages.NeedFiles,
		ParseFile: parseSource,
	}
	pkg, err := loadPackage(config, false)
	if err != nil {
		return nil, err
	}
	provisional, err := generateTypes(pkg.Types, pkg.Fset, true)
	if err != nil {
		return nil, err
	}
	path, err := filepath.Abs(filepath.Join(dir, Filename))
	if err != nil {
		return nil, err
	}
	config.ParseFile = nil
	config.Overlay = map[string][]byte{path: provisional}
	if len(provisional) == 0 {
		config.Overlay[path] = []byte("package " + pkg.Name + "\n")
	}
	pkg, err = loadPackage(config, false)
	if err != nil {
		return nil, err
	}
	output, err := GenerateTypes(pkg.Types, pkg.Fset)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(provisional, output) {
		config.Overlay[path] = output
		if _, err := loadPackage(config, true); err != nil {
			return nil, err
		}
	} else if len(pkg.Errors) != 0 {
		return nil, fmt.Errorf("%s", pkg.Errors[0])
	}
	return output, nil
}

// loadPackage reports syntax errors first and permits missing generated names during bootstrap.
func loadPackage(config *packages.Config, complete bool) (*packages.Package, error) {
	loaded, err := packages.Load(config, ".")
	if err != nil {
		return nil, err
	}
	if len(loaded) != 1 || loaded[0].Types == nil {
		return nil, fmt.Errorf("load exactly one Go package in %s", config.Dir)
	}
	for _, failure := range loaded[0].Errors {
		if complete || failure.Kind == packages.ParseError {
			return nil, fmt.Errorf("%s", failure)
		}
	}
	return loaded[0], nil
}

// GenerateTypes produces canonical output from compiler-resolved source types.
// Generated declarations are not inputs. Normal analysis must still check output.
func GenerateTypes(pkg *types.Package, fset *token.FileSet) ([]byte, error) {
	return generateTypes(pkg, fset, false)
}

// generateTypes first supplies projection names, then proves the resolved method results.
func generateTypes(pkg *types.Package, fset *token.FileSet, bootstrap bool) ([]byte, error) {
	g := generator{
		pkg: pkg, fset: fset, imports: make(map[string]string), names: make(map[string]string), bootstrap: bootstrap,
	}
	for _, name := range pkg.Scope().Names() {
		object, ok := pkg.Scope().Lookup(name).(*types.TypeName)
		if !ok || object.IsAlias() || g.generated(object) {
			continue
		}
		typ := object.Type()
		if !checker.Frozen(typ) && !checker.FrozenInterface(typ) {
			continue
		}
		if reason := g.proofs.FrozenReason(typ); reason != "" && !bootstrap {
			return nil, fmt.Errorf("%s: %s", name, reason)
		}
		g.imports[imPath] = "im"
		if checker.FrozenInterface(typ) {
			if err := g.addInterface(typ, name+"Projection"); err != nil {
				return nil, err
			}
		} else if err := g.addRecord(types.NewPointer(typ), name+"Projection"); err != nil {
			return nil, err
		}
	}
	if len(g.records) == 0 && len(g.interfaces) == 0 {
		return nil, nil
	}
	return g.render()
}

// generated identifies declarations in the single generated output file.
func (g *generator) generated(object types.Object) bool {
	return filepath.Base(g.fset.PositionFor(object.Pos(), false).Filename) == Filename
}

// parseSource hides stale generated declarations while retaining the package name.
func parseSource(fset *token.FileSet, filename string, source []byte) (*ast.File, error) {
	mode := parser.ParseComments
	if filepath.Base(filename) == Filename {
		mode = parser.PackageClauseOnly
	}
	return parser.ParseFile(fset, filename, source, mode)
}

// generator records the typed output and its required imports.
type generator struct {
	pkg        *types.Package
	fset       *token.FileSet
	proofs     checker.Prover
	imports    map[string]string
	names      map[string]string
	records    []record
	arrays     []array
	interfaces []record
	bootstrap  bool
}

// record describes a generated reader and its publication cell.
type record struct {
	typ  types.Type
	name string
}

// array describes a fixed-array reader whose elements cannot leave a projection directly.
type array struct {
	typ     *types.Array
	name    string
	root    types.Type
	path    string
	indices []string
}

// reserve rejects generated identifiers that overlap source declarations or each other.
func (g *generator) reserve(name, source string) error {
	previous := g.names[name]
	if object := g.pkg.Scope().Lookup(name); previous == "" && object != nil && !g.generated(object) {
		previous = "source declaration " + object.Name()
	}
	if previous != "" {
		return fmt.Errorf("generated name %s conflicts between %s and %s; rename one source declaration",
			name, source, previous)
	}
	g.names[name] = source
	return nil
}

// addRecord queues a typed reader and its constructor.
func (g *generator) addRecord(typ types.Type, name string) error {
	source := "record " + g.typeName(types.Unalias(typ).(*types.Pointer).Elem())
	if err := g.reserve(name, source); err != nil {
		return err
	}
	if err := g.reserve(constructorName(name), source); err != nil {
		return err
	}
	if err := g.reserve(strings.TrimSuffix(name, "Projection")+"Cell", source); err != nil {
		return err
	}
	g.records = append(g.records, record{typ: typ, name: name})
	return nil
}

// typeName prints Go types and assigns stable import names.
func (g *generator) typeName(typ types.Type) string {
	return types.TypeString(typ, func(pkg *types.Package) string {
		if pkg == g.pkg {
			return ""
		}
		if name, exists := g.imports[pkg.Path()]; exists {
			return name
		}
		name := pkg.Name()
		for suffix := 2; g.importUsed(name) || g.pkg.Scope().Lookup(name) != nil; suffix++ {
			name = pkg.Name() + strconv.Itoa(suffix)
		}
		g.imports[pkg.Path()] = name
		return name
	})
}

// importUsed checks names already assigned to generated imports.
func (g *generator) importUsed(name string) bool {
	for _, assigned := range g.imports {
		if assigned == name {
			return true
		}
	}
	return false
}

// render builds all requested readers before writing the import list.
func (g *generator) render() ([]byte, error) {
	var body bytes.Buffer
	for _, item := range g.interfaces {
		if err := g.writeInterface(&body, item); err != nil {
			return nil, err
		}
	}
	for ri, ai := 0, 0; ri < len(g.records) || ai < len(g.arrays); {
		if ri < len(g.records) {
			if err := g.writeRecord(&body, g.records[ri]); err != nil {
				return nil, err
			}
			ri++
		} else {
			if err := g.writeArray(&body, g.arrays[ai]); err != nil {
				return nil, err
			}
			ai++
		}
	}
	var output bytes.Buffer
	output.WriteString("// Code generated by immutago-gen. DO NOT EDIT.\n")
	fmt.Fprintf(&output, "\npackage %s\n", g.pkg.Name())
	if len(g.imports) > 0 {
		output.WriteString("\nimport (\n")
	}
	paths := make([]string, 0, len(g.imports))
	for path := range g.imports {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		fmt.Fprintf(&output, "%s %q\n", g.imports[path], path)
	}
	if len(g.imports) > 0 {
		output.WriteString(")\n\n")
	}
	output.Write(body.Bytes())
	return format.Source(output.Bytes())
}
