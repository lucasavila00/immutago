// Package borrow checks shared loans over Go compiler-resolved storage places.
// Its conflict, region, liveness, and dataflow algorithms are ports of the pinned
// rustc implementation. Go-specific provenance connects pointers to their roots.
package borrow

import (
 "go/token"
 "go/types"

 "github.com/lucasavila00/immutago/internal/borrow/flow"
 "golang.org/x/tools/go/ssa"
)

// Effect describes operations on storage reachable from a function input.
type Effect struct { Read, Write, Capture bool }

// Result describes a returned pointer's source or a fresh allocation.
type Result struct { Input int; Fresh bool; FreshID int; Shared bool; Protected bool; Path []Projection }

// Summary is a checked function's portable reference-effect contract.
type Summary struct { Inputs []Effect; Results [][]Result }

// AFact marks the summary as a go/analysis object fact.
func (*Summary) AFact() {}

// root records allocation identity and the permission of direct accesses.
type root struct {
 typ types.Type
 protected bool
 global bool
 input int
 fresh bool
 binding int
}

// reference preserves alias identity, its shared loan, and optional closure origin.
type reference struct {
 place Place
 loan int
 shared bool
 closure *ssa.Function
 binding int
 result int
}

// cell holds references stored at an addressable local location.
type cell struct { place Place; refs []reference }

// state is the forward provenance environment at a CFG point.
type state struct {
 values map[ssa.Value][]reference
 cells map[string]cell
}

// loan records one shared reference's creation and use-region seeds.
type loan struct { place Place; point int; uses flow.Set; external bool; deferred bool }

// access records a storage operation after provenance resolution.
type access struct { point int; pos token.Pos; ref reference; write bool }

// engine holds one function's Go adapter and ported loan analyses.
type engine struct {
 program *program
 fn *ssa.Function
 roots []root
 allocations map[ssa.Value]int
 points map[ssa.Instruction]int
 instructions []ssa.Instruction
 graph flow.Graph
 inputs []state
 outputs []state
 loans []loan
 loanIDs map[string]int
 accesses []access
 summary Summary
 diagnose bool
 record bool
 captured map[int]bool
 callRoots map[string]int
}

// emptyState allocates an empty provenance environment.
func emptyState() state { return state{values:make(map[ssa.Value][]reference),cells:make(map[string]cell)} }

// sameReference compares provenance without relying on slice comparability.
func sameReference(a,b reference) bool {
 return a.place.key()==b.place.key() && a.loan==b.loan && a.shared==b.shared &&
  a.closure==b.closure && a.binding==b.binding && a.result==b.result
}

// mergeReferences unions aliases and reports whether the destination changed.
func mergeReferences(dst []reference,src []reference) ([]reference,bool) {
 changed:=false
 for _,r:=range src {
  found:=false
  for _,existing:=range dst { if sameReference(existing,r) { found=true;break } }
  if !found { if !changed { dst=append([]reference(nil),dst...) }; dst=append(dst,r);changed=true }
 }
 return dst,changed
}

// clone copies environment maps while leaving immutable reference slices shared.
func (s state) clone() state {
 out:=emptyState()
 for k,v:=range s.values { out.values[k]=v }
 for k,v:=range s.cells { out.cells[k]=v }
 return out
}

// join forms the may-alias union at a control-flow merge.
func (s state) join(other state) bool {
 changed:=false
 for k,v:=range other.values { next,c:=mergeReferences(s.values[k],v);s.values[k]=next;changed=changed||c }
 for k,v:=range other.cells {
  old:=s.cells[k]
  next,c:=mergeReferences(old.refs,v.refs)
  s.cells[k]=cell{place:v.place,refs:next};changed=changed||c
 }
 return changed
}
