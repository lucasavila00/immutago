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
	switch value := t.Underlying().(type) {
	case *types.Basic:
		if value.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsComplex) != 0 {
			return
		}
	case *types.Array:
		if _, named := t.(*types.Named); !named {
			p.dependCategory(result, value.Elem(), "array element: ", copiedCategory)
		} else {
			result.reason = "named array must be wrapped in a struct with an immutable marker"
			result.repair = "use an unnamed array type; to keep a named type, put the unnamed array in a marked struct"
		}
		return
	case *types.Struct:
		if Copied(t) {
			p.depend(result, t, "")
			return
		}
	}
	rejectCopied(result, t)
}

// rejectCopied states the complete field rule and the required owner change for Local data.
func rejectCopied(result *obligation, t types.Type) {
	result.reason = fmt.Sprintf("%s is not a Copied value", types.TypeString(t, nil))
	result.repair = "use bool, numbers, unnamed arrays, or Copied records; " +
		"for Local fields or results, use matching projections; " +
		"if a Copied record needs text or Local fields, replace its im.Copied marker with im.Local " +
		"and use read-result fields such as im.StringProjection"
}

// copiedRecord proves each field through the shared Copied rule.
func (p *proof) copiedRecord(result *obligation, structure *types.Struct) {
	for field := range structure.Fields() {
		if field.Embedded() && IsLibraryType(field.Type(), "Copied") {
			continue
		}
		p.dependCategory(result, field.Type(), field.Name()+": ", copiedCategory)
	}
}

// readValue proves the general result category used by Local fields and readers.
func (p *proof) readValue(result *obligation, t types.Type) {
	switch {
	case IsUncheckedValue(t), IsHandle(t):
		p.dependCategory(result, t, "", frozenCategory)
		return
	case IsProjection(t):
		args := t.(*types.Named).TypeArgs()
		for index := 0; index < args.Len(); index++ {
			p.depend(result, args.At(index), fmt.Sprintf("projection payload %d: ", index+1))
		}
		return
	case IsLibraryType(t, "FrozenString"):
		rejectCopied(result, t)
		result.repair = "for Local text, use text.Projection()"
		return
	}
	switch value := t.Underlying().(type) {
	case *types.Basic:
		if value.Info()&types.IsString != 0 {
			rejectCopied(result, t)
			result.repair = "for Local text, use im.ProjectString(text)"
			return
		}
	case *types.Array:
		if _, named := t.(*types.Named); !named {
			p.dependCategory(result, value.Elem(), "array element: ", readCategory)
			return
		}
	case *types.Struct:
		if Local(t) {
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
