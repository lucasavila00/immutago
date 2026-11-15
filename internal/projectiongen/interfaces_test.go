package projectiongen

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestGenerateInterfaces checks initial generation, local method results, and runtime reads.
func TestGenerateInterfaces(t *testing.T) {
	dir := module(t, `package main
import "github.com/lucasavila00/immutago/pkg/im"
type Scorer interface { im.FrozenInterface; Score([]float32) float64; Self() ScorerProjection }
type Tree struct { im.Frozen; Weight float64 }
func(t *Tree)Score(features []float32)float64{return t.Weight*float64(len(features))}
func(t *Tree)Self()ScorerProjection{var s Scorer=t;return NewScorerProjection(im.ProjectionOf(s))}
type Model struct{im.Frozen;Scorer Scorer;Values [2]Scorer;List im.FrozenSlice[Scorer]}
var current ModelCell
func main(){
 var scorer Scorer=&Tree{Weight:3}
 values:=[]Scorer{scorer}
 frozen:=im.FreezeSlice(values)
 current.Store(Model{Scorer:scorer,Values:[2]Scorer{scorer},List:frozen})
 local:=current.Load()
 n:=local.Scorer().Self().Score([]float32{1,2})
 array:=local.Values().At(0).Score([]float32{1})
 list:=NewScorerProjection(local.List().At(0)).Score([]float32{1})
 if (ScorerProjection{}).IsNil(){println(int(n+array+list))}
}
`)
	first, err := Generate(dir)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, Filename), string(first))
	second, err := Generate(dir)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("interface regeneration differs: %v", err)
	}
	command := exec.Command("go", "run", ".")
	command.Dir = dir
	out, err := command.CombinedOutput()
	if err != nil || string(out) != "12\n" {
		t.Fatalf("interface readers: %v\n%s", err, out)
	}
}

// TestInterfaceProjectionCost checks the stored representation and allocation contract.
func TestInterfaceProjectionCost(t *testing.T) {
	command := exec.Command("go", "test", "./testdata/interfacecost", "-run", "^TestProjectionCost$", "-count=1")
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("interface projection cost: %v\n%s", err, out)
	}
}

// TestInheritedPrivateResult rejects a method type that the generated file cannot name.
func TestInheritedPrivateResult(t *testing.T) {
	dir := module(t, `package reader
import "example.test/projections/hidden"
type Reader interface { hidden.Reader }
`)
	hidden := filepath.Join(dir, "hidden")
	if err := os.Mkdir(hidden, 0700); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(hidden, "reader.go"), `package hidden
import "github.com/lucasavila00/immutago/pkg/im"
type number int
type Reader interface { im.FrozenInterface; Read() (int, number) }
`)
	output, err := Generate(hidden)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(hidden, Filename), string(output))
	if _, err := Generate(dir); err == nil {
		t.Fatal("an inherited private result escaped the generator's compiler check")
	}
}
