// Algorithms adapted from rustc at 80196be0e787008b6fb14d2990aba92083cd68e1.
// Source: compiler/rustc_borrowck/src/places_conflict.rs (lines 99-526).
// See third_party/rustc for source attribution and license terms.

package borrow

import (
	"fmt"
	"strings"
)

// Projection selects a field or an array element within one allocation.
// Kind is 'f' for a field, 'i' for a constant index, '?' for an unknown index,
// 'd' for a symbolic dereference, and '*' for an opaque recursive tail.
type Projection struct {
	Kind  byte
	Index int64
}

// Place identifies storage independently of the pointer variables that refer to it.
// Go aliases are resolved to allocation roots before the rustc comparison runs.
type Place struct {
	Root int
	Path []Projection
}

// project appends a component without changing another alias's path.
func (p Place) project(kind byte, index int64) Place {
	if len(p.Path) > 0 && p.Path[len(p.Path)-1].Kind == '*' {
		return p
	}
	path := append([]Projection(nil), p.Path...)
	return Place{Root: p.Root, Path: append(path, Projection{kind, index})}
}

// key returns a stable identity for a storage cell in the provenance environment.
func (p Place) key() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d", p.Root)
	for _, component := range p.Path {
		fmt.Fprintf(&b, "/%c%d", component.Kind, component.Index)
	}
	return b.String()
}

// Overlap ports rustc's projection comparison with explicit may/must bias.
// Possible overlap is used for access checks; definite overlap is used for kills.
// Dereferences are already resolved to canonical allocation roots by the Go adapter.
func Overlap(a, b Place, definite bool) bool {
	if a.Root != b.Root {
		return false
	}
	for _, path := range [][]Projection{a.Path, b.Path} {
		for _, projection := range path {
			if projection.Kind == '*' {
				return !definite
			}
		}
	}
	for i := 0; i < len(a.Path) && i < len(b.Path); i++ {
		x, y := a.Path[i], b.Path[i]
		if x.Kind == '?' || y.Kind == '?' {
			if definite {
				return false
			}
			continue
		}
		if x.Kind != y.Kind {
			return !definite
		}
		if x.Index != y.Index {
			return false
		}
	}
	return true
}
