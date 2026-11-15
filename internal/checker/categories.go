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
	case *types.Struct:
		if Copied(t) {
			return
		}
	}
	result.reason = "copied values require bool, numbers, Copied records, arrays, or service references"
}

// copiedRecord proves each field through the shared Copied rule.
func (p *proof) copiedRecord(result *obligation, t types.Type, structure *types.Struct) {
	if MarkerCount(t) != 1 {
		result.reason = "Copied record must embed exactly one marker"
		return
	}
	for field := range structure.Fields() {
		if field.Embedded() && IsLibraryType(field.Type(), "Copied") {
			continue
		}
		if p.queries.Handles(field.Type()) {
			result.reason = field.Name() + ": Copied records cannot contain service references"
			return
		}
		p.dependCategory(result, field.Type(), field.Name()+": ", copiedCategory)
	}
}

// readValue proves the general result category used by Local fields and readers.
func (p *proof) readValue(result *obligation, t types.Type) {
	switch {
	case IsUncheckedValue(t):
		p.depend(result, t, "")
		return
	case IsProjection(t):
		args := t.(*types.Named).TypeArgs()
		for index := 0; index < args.Len(); index++ {
			p.depend(result, args.At(index), fmt.Sprintf("projection payload %d: ", index+1))
		}
		return
	case IsLibraryType(t, "Local"):
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
