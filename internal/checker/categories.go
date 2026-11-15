package checker

import (
	"fmt"
	"go/types"
)

type proofCategory uint8

const (
	frozenCategory proofCategory = iota
	copiedCategory
	readCategory
)

// copiedValue excludes retained data through the same recursive rule for every use.
func (p *proof) copiedValue(result *obligation, t types.Type) {
	p.depend(result, t, "")
	if IsRuntimeType(t, "Ref") {
		return
	}
	switch value := t.Underlying().(type) {
	case *types.Basic:
		if value.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsComplex) != 0 {
			return
		}
	case *types.Array:
		if _, named := t.(*types.Named); !named {
			p.dependCategory(result, value.Elem(), "array element: ", copiedCategory)
			return
		}
	}
	result.reason = "copied values require bool, numbers, arrays, or service references"
}

// readValue proves the general result category used by Local fields and readers.
func (p *proof) readValue(result *obligation, t types.Type) {
	if IsProjection(t) {
		args := t.(*types.Named).TypeArgs()
		for index := 0; index < args.Len(); index++ {
			p.depend(result, args.At(index), fmt.Sprintf("projection payload %d: ", index+1))
		}
		return
	}
	if IsLibraryType(t, "Local") {
		result.reason = "immutable markers are annotations, not data"
		return
	}
	switch value := t.Underlying().(type) {
	case *types.Array:
		if _, named := t.(*types.Named); !named {
			p.dependCategory(result, value.Elem(), "array element: ", readCategory)
			return
		}
	case *types.Struct:
		if Local(t) {
			if MarkerCount(t) != 1 {
				result.reason = "Local struct must embed exactly one marker"
				return
			}
			for field := range value.Fields() {
				if field.Embedded() && IsLibraryType(field.Type(), "Local") {
					continue
				}
				p.dependCategory(result, field.Type(), field.Name()+": ", readCategory)
			}
			return
		}
	}
	p.dependCategory(result, t, "", copiedCategory)
}
