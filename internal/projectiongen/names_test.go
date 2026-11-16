package projectiongen

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestPrivateProjectionNames checks source names, repeated generation, and unchanged layouts.
func TestPrivateProjectionNames(t *testing.T) {
	dir := module(t, `package main
import (
 immutable "github.com/lucasavila00/immutago/pkg/im"
 "unsafe"
)
type im int
var im2 int
func immutagoLocal() {}
const immutagoLocal2 = 2
type immutagoLocal3 int
type Marker = immutable.Frozen
type Record struct {
 Marker
 Local int
 immutagoLocal4 int
 immutagoProjection int
 immutagoProjection2 int
 Grid [2][3]int
 Reader Reader
}
type Reader interface { immutable.FrozenInterface; Local() int; Unsafe() int }
type Number struct { immutable.Frozen; N int }
func (n *Number) Local() int { return n.N }
func (n *Number) Unsafe() int { return n.N+1 }
var current RecordCell
func main() {
 current.Store(Record{Local:1, immutagoLocal4:2, immutagoProjection:3,
  immutagoProjection2:4, Grid:[2][3]int{{},{0,0,5}}, Reader:&Number{N:6}})
 p := current.Load()
 println(p.Local(), p.immutagoLocal4(), p.immutagoProjection(),
  p.immutagoProjection2(), p.Grid().At(1).At(2), p.Reader().Local(), p.Reader().Unsafe())
 if unsafe.Sizeof(RecordProjection{}) != unsafe.Sizeof(uintptr(0)) ||
  unsafe.Sizeof(RecordGridProjection{}) != unsafe.Sizeof(uintptr(0)) ||
  unsafe.Sizeof(RecordGridElementProjection{}) != 2*unsafe.Sizeof(uintptr(0)) ||
  unsafe.Sizeof(ReaderProjection{}) != 2*unsafe.Sizeof(uintptr(0)) { panic("projection layout changed") }
}
`)
	first, err := Generate(dir)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, Filename), string(first))
	second, err := Generate(dir)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("private names changed after generation: %v", err)
	}
	command := exec.Command("go", "run", ".")
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil || string(output) != "1 2 3 4 5 6 7\n" {
		t.Fatalf("generated name access: %v\n%s", err, output)
	}
}

// TestForeignPrivateRecords reports the declaration rule before writing inaccessible names.
func TestForeignPrivateRecords(t *testing.T) {
	for _, field := range []string{"hidden.Public", "*hidden.Public", "[2]hidden.Public", "[2]*hidden.Public"} {
		t.Run(field, func(t *testing.T) {
			dir := module(t, `package consumer
import (
 "example.test/projections/hidden"
 "github.com/lucasavila00/immutago/pkg/im"
)
type Record struct { im.Frozen; Item `+field+` }
`)
			hidden := filepath.Join(dir, "hidden")
			if err := os.Mkdir(hidden, 0700); err != nil {
				t.Fatal(err)
			}
			write(t, filepath.Join(hidden, "record.go"), `package hidden
import "github.com/lucasavila00/immutago/pkg/im"
type private struct { im.Frozen; N int }
type Public = private
`)
			output, err := Generate(hidden)
			if err != nil {
				t.Fatal(err)
			}
			write(t, filepath.Join(hidden, Filename), string(output))
			_, err = Generate(dir)
			if err == nil || !strings.Contains(err.Error(),
				"Frozen record hidden.private requires an exported declaration for a foreign projection") {
				t.Fatalf("private record error: %v", err)
			}
		})
	}
}

// TestImportProjectionNames keeps an imported package name clear of generated array types.
func TestImportProjectionNames(t *testing.T) {
	dir := module(t, `package consumer
import (
 foreign "example.test/projections/foreign"
 "github.com/lucasavila00/immutago/pkg/im"
)
type Record struct { im.Frozen; Number foreign.Number; Grid [2][3]int }
`)
	foreign := filepath.Join(dir, "foreign")
	if err := os.Mkdir(foreign, 0700); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(foreign, "number.go"), "package RecordGridElementProjection\ntype Number int\n")
	output, err := Generate(dir)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, Filename), string(output))
	second, err := Generate(dir)
	if err != nil || !bytes.Equal(output, second) {
		t.Fatalf("import name changed after generation: %v", err)
	}
}

// TestImportLocalNames prevents generated locals and imports from hiding each other.
func TestImportLocalNames(t *testing.T) {
	for _, name := range []string{"v", "c", "value", "arg0", "index0", "index1", "int", "bool", "nil", "panic"} {
		t.Run(name, func(t *testing.T) {
			dir := module(t, `package consumer
import (
 foreign "example.test/projections/foreign"
 "github.com/lucasavila00/immutago/pkg/im"
)
type Record struct { im.Frozen; Field foreign.Point; Grid [2][3]foreign.Point }
`)
			foreign := filepath.Join(dir, "foreign")
			if err := os.Mkdir(foreign, 0700); err != nil {
				t.Fatal(err)
			}
			write(t, filepath.Join(foreign, "point.go"), "package "+name+`
import "github.com/lucasavila00/immutago/pkg/im"
type Point struct { im.Frozen; Number int }
`)
			output, err := Generate(foreign)
			if err != nil {
				t.Fatal(err)
			}
			write(t, filepath.Join(foreign, Filename), string(output))
			if _, err := Generate(dir); err != nil {
				t.Fatal(err)
			}
		})
	}
}
