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
			result.repair = "use an unnamed array type; to keep a named type, " +
				"put the unnamed array in a Frozen, Local, or Copied record"
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

// rejectCopied states the complete field rule and the owner change for other read-result fields.
func rejectCopied(result *obligation, t types.Type) {
	result.reason = fmt.Sprintf("%s is not a Copied value", types.TypeString(t, nil))
	result.repair = "use bool, numbers, unnamed arrays, or Copied records; " +
		"for Local fields or results, use matching projections; " +
		"if a Copied record needs text, UncheckedValue, or Local fields, " +
		"replace its im.Copied marker with im.Local and use read-result fields such as im.StringProjection"
}

// rejectRead states the complete result rule shared by Local fields and service results.
func rejectRead(result *obligation, t types.Type) {
	result.reason = fmt.Sprintf("%s is not a read result", types.TypeString(t, nil))
	result.repair = "use a read result: a Copied value, a Local record or matching projection, im.UncheckedValue, " +
		"a named Frozen record with one named im.UncheckedValue field and no other data fields, " +
		"a pointer to that record, or an unnamed array of read results"
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
	case Copied(t):
		p.dependCategory(result, t, "", copiedCategory)
		return
	case IsProjection(t):
		args := t.(*types.Named).TypeArgs()
		for index := 0; index < args.Len(); index++ {
			p.depend(result, args.At(index), fmt.Sprintf("projection payload %d: ", index+1))
		}
		return
	case IsLibraryType(t, "FrozenString"):
		rejectRead(result, t)
		result.repair = "for Local text, use text.Projection()"
		return
	}
	switch value := t.Underlying().(type) {
	case *types.Pointer:
		if IsHandle(value.Elem()) {
			p.dependCategory(result, t, "", frozenCategory)
			return
		}

	case *types.Basic:
		switch {
		case value.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsComplex) != 0:
			p.dependCategory(result, t, "", copiedCategory)
			return
		case value.Info()&types.IsString != 0:
			rejectRead(result, t)
			result.repair = "for Local text, use im.ProjectString(text)"
			return
		}
	case *types.Array:
		if _, named := t.(*types.Named); !named {
			p.dependCategory(result, value.Elem(), "array element: ", readCategory)
		} else {
			p.dependCategory(result, t, "", copiedCategory)
		}
		return
	case *types.Struct:
		if !Local(t) {
			break
		}
		for field := range value.Fields() {
			if field.Embedded() && IsLibraryType(field.Type(), "Local") {
				continue
			}
			p.dependCategory(result, field.Type(), field.Name()+": ", readCategory)
		}
		return
	}
	rejectRead(result, t)
}
